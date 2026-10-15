// Package renode é um cliente mínimo do monitor do Renode (porta TCP aberta
// com `renode --port N`). A planta usa o monitor para conduzir a emulação em
// passos (lockstep): ajusta as tensões do ADC, roda um intervalo de tempo
// virtual e lê os registradores do injetor.
package renode

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Monitor é uma conexão com o monitor. Não é seguro para uso concorrente.
type Monitor struct {
	conn    net.Conn
	r       *bufio.Reader
	Timeout time.Duration // por comando; 0 = sem limite
}

var (
	promptRe = regexp.MustCompile(`\(([\w.\-]+)\) $`)
	hexRe    = regexp.MustCompile(`^0x[0-9A-Fa-f]+$`)
	errTexts = []string{"There was an error", "No such command", "Could not", "error executing"}
)

// Dial conecta e espera o primeiro prompt. Tenta de novo até o prazo, porque
// o Renode demora alguns segundos para abrir a porta (e, atrás do proxy do
// Docker, a conexão pode ser aceita e fechada antes disso).
func Dial(addr string, wait time.Duration) (*Monitor, error) {
	deadline := time.Now().Add(wait)
	for {
		m, err := dialOnce(addr)
		if err == nil {
			return m, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("renode: conectando em %s: %w", addr, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func dialOnce(addr string) (*Monitor, error) {
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return nil, err
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
	}
	m := &Monitor{conn: conn, r: bufio.NewReader(conn), Timeout: 30 * time.Second}
	// Numa reconexão o Renode não reenvia o banner: uma linha vazia faz o
	// prompt aparecer. Depois descartamos tudo até a conexão ficar quieta,
	// para nenhum prompt sobrando ser confundido com a resposta do 1º comando.
	if _, err := conn.Write([]byte("\n")); err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := m.readUntilPrompt(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("esperando o prompt: %w", err)
	}
	for {
		_ = conn.SetDeadline(time.Now().Add(300 * time.Millisecond))
		if _, err := m.readUntilPrompt(); err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				break
			}
			conn.Close()
			return nil, err
		}
	}
	_ = conn.SetDeadline(time.Time{})
	return m, nil
}

// Close fecha a conexão (o Renode continua rodando).
func (m *Monitor) Close() error { return m.conn.Close() }

// Exec roda um comando (ou vários, separados por ";") e devolve as linhas de
// saída, sem o eco e sem o prompt. Mensagens de erro do monitor viram error.
func (m *Monitor) Exec(cmd string) ([]string, error) {
	if strings.ContainsAny(cmd, "\r\n") {
		return nil, errors.New("renode: comando com quebra de linha")
	}
	if m.Timeout > 0 {
		_ = m.conn.SetDeadline(time.Now().Add(m.Timeout))
	}
	if _, err := m.conn.Write([]byte(cmd + "\n")); err != nil {
		return nil, err
	}
	raw, err := m.readUntilPrompt()
	if err != nil {
		return nil, err
	}
	lines := splitLines(raw)
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == strings.TrimSpace(cmd) {
		lines = lines[1:] // eco
	}
	for _, l := range lines {
		for _, e := range errTexts {
			if strings.Contains(l, e) {
				return lines, fmt.Errorf("renode: %q: %s", cmd, strings.Join(lines, " | "))
			}
		}
	}
	return lines, nil
}

// readUntilPrompt lê até a saída terminar em "(nome) ", descartando a
// negociação telnet (bytes IAC).
func (m *Monitor) readUntilPrompt() (string, error) {
	var buf bytes.Buffer
	for {
		b, err := m.r.ReadByte()
		if err != nil {
			return buf.String(), err
		}
		if b == 0xFF { // IAC: comando telnet de 3 bytes (ou IAC IAC)
			cmd, err := m.r.ReadByte()
			if err != nil {
				return buf.String(), err
			}
			if cmd >= 0xFB && cmd <= 0xFE {
				if _, err := m.r.ReadByte(); err != nil {
					return buf.String(), err
				}
			}
			continue
		}
		buf.WriteByte(b)
		if b == ' ' && promptRe.Match(tail(buf.Bytes(), 64)) && m.r.Buffered() == 0 {
			out := buf.String()
			loc := promptRe.FindStringIndex(out)
			return out[:loc[0]], nil
		}
	}
}

func tail(b []byte, n int) []byte {
	if len(b) > n {
		return b[len(b)-n:]
	}
	return b
}

func splitLines(s string) []string {
	var out []string
	for _, l := range strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' }) {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// ParseUint32 lê um valor como "0x00000672" devolvido por ReadDoubleWord.
func ParseUint32(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	if !hexRe.MatchString(s) {
		return 0, fmt.Errorf("renode: valor inesperado %q", s)
	}
	v, err := strconv.ParseUint(s[2:], 16, 32)
	return uint32(v), err
}

// Hexes devolve, em ordem, as linhas que são valores hexadecimais.
func Hexes(lines []string) ([]uint32, error) {
	var out []uint32
	for _, l := range lines {
		if hexRe.MatchString(strings.TrimSpace(l)) {
			v, err := ParseUint32(l)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
	}
	return out, nil
}

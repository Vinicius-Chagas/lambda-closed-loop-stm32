package renode

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeMonitor imita o protocolo do monitor: negociação telnet, eco do
// comando, saída com \n\r e prompt "(ecu) ".
func fakeMonitor(t *testing.T, handle func(cmd string) string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.Write([]byte("\xff\xfd\x00\xff\xfb\x01\xff\xfc\"enode, version 1.17.0\n\r\n\r(monitor) \n\r(monitor) "))
		r := bufio.NewReader(conn)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.TrimRight(line, "\n")
			out := ""
			if cmd != "" { // linha vazia: o Renode só devolve o prompt
				out = handle(cmd)
			}
			conn.Write([]byte(cmd + "\n\r" + out + "(ecu) "))
		}
	}()
	return ln.Addr().String()
}

func TestExecRemoveEcoEPrompt(t *testing.T) {
	addr := fakeMonitor(t, func(cmd string) string {
		if cmd == "sysbus ReadDoubleWord 0x40000034" {
			return "0x00000672\r\r\n"
		}
		return ""
	})
	m, err := Dial(addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	lines, err := m.Exec("sysbus ReadDoubleWord 0x40000034")
	if err != nil {
		t.Fatal(err)
	}
	v, err := Hexes(lines)
	if err != nil || len(v) != 1 || v[0] != 0x672 {
		t.Fatalf("Hexes = %v, %v; linhas %q", v, err, lines)
	}

	lines, err = m.Exec("sysbus.adc.lambdaSensor Volts 1.2")
	if err != nil || len(lines) != 0 {
		t.Fatalf("comando sem saída: %q, %v", lines, err)
	}
}

func TestExecDetectaErro(t *testing.T) {
	addr := fakeMonitor(t, func(cmd string) string {
		return "No such command or device: " + strings.Fields(cmd)[0] + "\r\n"
	})
	m, err := Dial(addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if _, err := m.Exec("comando_errado 1"); err == nil {
		t.Fatal("esperado erro")
	}
}

func TestECUStepComandoComposto(t *testing.T) {
	var recebido string
	addr := fakeMonitor(t, func(cmd string) string {
		recebido = cmd
		return "0x00000BB8 0x0000EA5F\r\r\n"
	})
	m, err := Dial(addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	ecu, err := NewECU(m, "")
	if err != nil {
		t.Fatal(err)
	}
	inj, err := ecu.Step([]float64{1.65, 0.3, 0.33}, 0.005)
	if err != nil {
		t.Fatal(err)
	}
	if inj.PulseUs != 3000 || inj.PeriodUs != 60000 {
		t.Errorf("injetor = %+v", inj)
	}
	want := `lockstep 1.6500 0.3000 0.3300 0.005`
	if recebido != want {
		t.Errorf("comando:\n%s\nesperado:\n%s", recebido, want)
	}
	if _, err := ecu.Step([]float64{1}, 0.005); err == nil {
		t.Error("esperado erro com número errado de tensões")
	}
}

func TestDialDesisteNoPrazo(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	start := time.Now()
	if _, err := Dial(addr, 300*time.Millisecond); err == nil {
		t.Fatal("esperado erro")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("Dial demorou demais para desistir")
	}
}

package plant

import (
	"encoding/csv"
	"io"
	"strconv"
)

// CSVWriter grava uma linha por amostra, para os gráficos do relatório.
type CSVWriter struct {
	w *csv.Writer
}

// CSVHeader são as colunas gravadas.
var CSVHeader = []string{
	"tempo_s", "tps_pct", "rpm", "lambda_cilindro", "lambda_escape", "lambda_sonda",
	"ti_us", "ciclo_us", "fator_bico", "vazamento_mg", "atraso_ms",
}

// NewCSVWriter escreve o cabeçalho e devolve o gravador.
func NewCSVWriter(w io.Writer) (*CSVWriter, error) {
	c := &CSVWriter{w: csv.NewWriter(w)}
	return c, c.w.Write(CSVHeader)
}

// Write grava uma amostra.
func (c *CSVWriter) Write(s Sample) error {
	f := func(x float64, prec int) string { return strconv.FormatFloat(x, 'f', prec, 64) }
	return c.w.Write([]string{
		f(s.Tempo, 3), f(s.TPS, 1), f(s.RPM, 0), f(s.LambdaCil, 4), f(s.LambdaEscape, 4), f(s.LambdaSonda, 4),
		f(s.Injector.PulseUs, 0), f(s.Injector.PeriodUs, 0), f(s.FatorBico, 3), f(s.VazamentoMg, 1),
		f(s.Atraso*1000, 1),
	})
}

// Flush grava o que estiver no buffer.
func (c *CSVWriter) Flush() error {
	c.w.Flush()
	return c.w.Error()
}

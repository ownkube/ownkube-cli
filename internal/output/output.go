package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// Printer formats output in different formats.
type Printer struct {
	format string
	w      io.Writer
}

// New creates a Printer for the given format ("table", "json", "yaml").
func New(w io.Writer, format string) *Printer {
	return &Printer{format: format, w: w}
}

// Print outputs data in the configured format.
// For table format, data should be a [][]string where the first row is headers.
func (p *Printer) Print(data any) error {
	switch p.format {
	case "json":
		enc := json.NewEncoder(p.w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case "yaml":
		return yaml.NewEncoder(p.w).Encode(data)
	default:
		return p.printTable(data)
	}
}

func (p *Printer) printTable(data any) error {
	rows, ok := data.([][]string)
	if !ok {
		return fmt.Errorf("table format requires [][]string data")
	}
	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			fmt.Fprint(tw, col)
		}
		fmt.Fprintln(tw)
	}
	return tw.Flush()
}

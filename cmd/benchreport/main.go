// Command benchreport renders a two-column UTF-8 validation benchmark report.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type row struct {
	Name       string
	Stdlib     float64
	SIMD       float64
	HasStdlib  bool
	HasSIMD    bool
	StdWinner  bool
	SIMDWinner bool
}

type report struct {
	Rows      []row
	Runtime   string
	Hardware  string
	Count     int
	BenchTime string
}

var page = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>UTF-8 validation benchmark</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 2rem; color: #1f2937; }
table { border-collapse: collapse; min-width: 42rem; }
th, td { border: 1px solid #d1d5db; padding: .65rem .8rem; text-align: right; }
th:first-child, td:first-child { text-align: left; }
th { background: #f3f4f6; }
.winner { background: #dcfce7; color: #166534; font-weight: 700; }
.missing { color: #6b7280; }
</style></head><body><h1>UTF-8 validation: lower is better</h1>
<p><strong>Runtime:</strong> {{.Runtime}}<br><strong>Hardware:</strong> {{.Hardware}}<br><strong>Run:</strong> {{.Count}} sample(s), <code>-benchtime={{.BenchTime}}</code></p>
<p>Inputs: Latin, Cyrillic, Chinese, and emoji at 2 B, 8 B, 64 B, 512 B, 4 KiB, 64 KiB, and 128 KiB; plus empty input and a 64 KiB early error. Multibyte inputs end at a UTF-8 boundary.</p>
<p>Median of each implementation's <code>ns/op</code> samples. Green marks the winner.</p>
<table><thead><tr><th>Input</th><th>gosimd/utf8.Valid</th><th>stdlib/utf8.Valid</th></tr></thead>
<tbody>{{range .Rows}}<tr><td>{{.Name}}</td>
<td {{if .SIMDWinner}}class="winner"{{end}}{{if not .HasSIMD}}class="missing"{{end}}>{{if .HasSIMD}}{{printf "%.2f ns/op" .SIMD}}{{else}}—{{end}}</td>
<td {{if .StdWinner}}class="winner"{{end}}{{if not .HasStdlib}}class="missing"{{end}}>{{if .HasStdlib}}{{printf "%.2f ns/op" .Stdlib}}{{else}}—{{end}}</td>
</tr>{{end}}</tbody></table></body></html>`))

func main() {
	input := flag.String("input", "-", "Go benchmark output file, or - for stdin")
	output := flag.String("output", "bench/valid.html", "HTML report path")
	hardware := flag.String("hardware", "unspecified", "hardware description included in the report")
	benchTime := flag.String("benchtime", "unspecified", "value passed to go test -benchtime")
	count := flag.Int("count", 1, "value passed to go test -count")
	flag.Parse()

	data, err := readInput(*input)
	if err != nil {
		fatal(err)
	}
	rows, err := parse(data)
	if err != nil {
		fatal(err)
	}

	f, err := os.Create(*output)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	if err := page.Execute(f, report{
		Rows:      rows,
		Runtime:   runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH,
		Hardware:  *hardware,
		Count:     *count,
		BenchTime: *benchTime,
	}); err != nil {
		fatal(err)
	}
}

func readInput(name string) ([]byte, error) {
	if name == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(name)
}

func parse(data []byte) ([]row, error) {
	results := make(map[string]map[string][]float64)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.HasPrefix(fields[0], "BenchmarkValid/") {
			continue
		}
		parts := strings.Split(fields[0], "/")
		if len(parts) != 4 {
			continue
		}
		implementation := strings.Split(parts[3], "-")[0]
		if implementation != "stdlib" && implementation != "simd" {
			continue
		}
		value, ok := nsPerOp(fields)
		if !ok {
			continue
		}
		name := parts[1] + "/" + parts[2]
		if results[name] == nil {
			results[name] = make(map[string][]float64)
		}
		results[name][implementation] = append(results[name][implementation], value)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no BenchmarkValid stdlib/simd results found")
	}

	rows := make([]row, 0, len(results))
	for name, implementations := range results {
		result := row{Name: name}
		if samples := implementations["stdlib"]; len(samples) != 0 {
			result.Stdlib, result.HasStdlib = median(samples), true
		}
		if samples := implementations["simd"]; len(samples) != 0 {
			result.SIMD, result.HasSIMD = median(samples), true
		}
		result.StdWinner = result.HasStdlib && (!result.HasSIMD || result.Stdlib <= result.SIMD)
		result.SIMDWinner = result.HasSIMD && (!result.HasStdlib || result.SIMD <= result.Stdlib)
		rows = append(rows, result)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows, nil
}

func nsPerOp(fields []string) (float64, bool) {
	for i := 1; i < len(fields); i++ {
		if fields[i] != "ns/op" {
			continue
		}
		value, err := strconv.ParseFloat(fields[i-1], 64)
		return value, err == nil
	}
	return 0, false
}

func median(samples []float64) float64 {
	ordered := append([]float64(nil), samples...)
	sort.Float64s(ordered)
	middle := len(ordered) / 2
	if len(ordered)%2 != 0 {
		return ordered[middle]
	}
	return (ordered[middle-1] + ordered[middle]) / 2
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "benchreport:", err)
	os.Exit(1)
}

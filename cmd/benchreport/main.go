// Command benchreport runs the stable UTF-8 and UTF-16 publication benchmark
// matrix on the local machine and writes a commit-ready Markdown report.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

var operationOrder = []string{
	"utf8.Valid",
	"utf8.RuneCount",
	"utf8.Encode-full",
	"utf8.Encode-core",
	"utf8.Decode-full",
	"utf8.Decode-core",
	"utf16.Encode-full",
	"utf16.Encode-core",
	"utf16.Decode-full",
	"utf16.Decode-core",
}

var scenarioOrder = []string{"ascii-only", "mixed", "russian", "chinese"}

type sample struct {
	nsPerOp    float64
	inputBytes float64
	chars      float64
	bytesAlloc float64
	allocs     float64
}

type measurement struct {
	operation  string
	scenario   string
	impl       string
	nsPerOp    float64
	inputBytes float64
	chars      float64
	bytesAlloc float64
	allocs     float64
}

type machineInfo struct {
	CPU         string
	Frequency   string
	SIMD        string
	LogicalCPUs int
}

type reportInfo struct {
	Generated time.Time
	GoVersion string
	GOOS      string
	GOARCH    string
	Revision  string
	BenchTime string
	Count     int
	Machine   machineInfo
	Command   string
	Results   map[string]measurement
}

func main() {
	defaultOutput := filepath.Join("bench", "results-"+runtime.GOOS+"-"+runtime.GOARCH+".md")
	goCommand := flag.String("go", filepath.Join(runtime.GOROOT(), "bin", goExecutable()), "Go command used to run benchmarks")
	output := flag.String("output", defaultOutput, "Markdown report path")
	benchTime := flag.String("benchtime", "1s", "value passed to go test -benchtime")
	count := flag.Int("count", 5, "number of benchmark samples")
	cpu := flag.String("cpu", "", "override the detected CPU name")
	frequency := flag.String("frequency", "", "override the detected CPU frequency")
	simd := flag.String("simd", "", "override the detected active SIMD backend")
	flag.Parse()

	if *count < 1 {
		fatal(fmt.Errorf("count must be at least 1"))
	}

	machine := detectMachine()
	if *simd == "" && strings.HasPrefix(machine.SIMD, "disabled") {
		fatal(fmt.Errorf("benchreport must itself be built with GOEXPERIMENT=simd; use make bench-report or prefix go run with GOEXPERIMENT=simd"))
	}
	if *cpu != "" {
		machine.CPU = *cpu
	}
	if *frequency != "" {
		machine.Frequency = *frequency
	}
	if *simd != "" {
		machine.SIMD = *simd
	}

	data, command, err := runBenchmarks(*goCommand, *benchTime, *count)
	if err != nil {
		fatal(err)
	}
	results, err := parse(data)
	if err != nil {
		fatal(err)
	}
	if err := validateCoreAllocations(results); err != nil {
		fatal(err)
	}

	info := reportInfo{
		Generated: time.Now().UTC(),
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		Revision:  gitRevision(),
		BenchTime: *benchTime,
		Count:     *count,
		Machine:   machine,
		Command:   command,
		Results:   results,
	}
	if err := writeReport(*output, info); err != nil {
		fatal(err)
	}
	fmt.Fprintln(os.Stderr, "benchreport: wrote", *output)
}

func goExecutable() string {
	if runtime.GOOS == "windows" {
		return "go.exe"
	}
	return "go"
}

func runBenchmarks(goCommand, benchTime string, count int) ([]byte, string, error) {
	args := []string{
		"test", "-run=^$", "-bench=^BenchmarkReport$", "-benchmem",
		"-benchtime=" + benchTime, "-count=" + strconv.Itoa(count),
	}
	packages := []string{"./utf8", "./utf16"}
	var output bytes.Buffer
	for _, pkg := range packages {
		fmt.Fprintln(os.Stderr, "benchreport: benchmarking", pkg)
		cmd := exec.Command(goCommand, append(args, pkg)...)
		cmd.Env = withEnvironment(os.Environ(), "GOEXPERIMENT", "simd")
		chunk, err := cmd.CombinedOutput()
		output.Write(chunk)
		if err != nil {
			return nil, "", fmt.Errorf("%s %s: %w\n%s", goCommand, strings.Join(append(args, pkg), " "), err, chunk)
		}
	}
	displayCommand := reproductionCommand(goCommand, args, packages)
	return output.Bytes(), displayCommand, nil
}

func reproductionCommand(goCommand string, args, packages []string) string {
	commands := make([]string, 0, len(packages)+1)
	prefix := "GOEXPERIMENT=simd "
	if runtime.GOOS == "windows" {
		commands = append(commands, "$env:GOEXPERIMENT = 'simd'")
		prefix = ""
	}
	for _, pkg := range packages {
		commands = append(commands, prefix+goCommand+" "+strings.Join(args, " ")+" "+pkg)
	}
	return strings.Join(commands, "\n")
}

func withEnvironment(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, item := range environment {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return append(result, prefix+value)
}

func parse(data []byte) (map[string]measurement, error) {
	samples := make(map[string][]sample)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.HasPrefix(fields[0], "BenchmarkReport/") {
			continue
		}
		parts := strings.Split(fields[0], "/")
		if len(parts) != 4 {
			continue
		}
		implementation := strings.SplitN(parts[3], "-", 2)[0]
		if implementation != "gosimd" && implementation != "stdlib" {
			continue
		}
		metrics := parseMetrics(fields)
		current := sample{
			nsPerOp:    metrics["ns/op"],
			inputBytes: metrics["input_bytes/op"],
			chars:      metrics["chars/op"],
			bytesAlloc: metrics["B/op"],
			allocs:     metrics["allocs/op"],
		}
		if current.nsPerOp <= 0 || current.inputBytes <= 0 || current.chars <= 0 {
			return nil, fmt.Errorf("incomplete metrics in benchmark row %q", line)
		}
		key := resultKey(parts[1], parts[2], implementation)
		samples[key] = append(samples[key], current)
	}

	results := make(map[string]measurement, len(operationOrder)*len(scenarioOrder)*2)
	for _, operation := range operationOrder {
		for _, scenario := range scenarioOrder {
			for _, implementation := range []string{"gosimd", "stdlib"} {
				key := resultKey(operation, scenario, implementation)
				values := samples[key]
				if len(values) == 0 {
					return nil, fmt.Errorf("missing benchmark result for %s/%s/%s", operation, scenario, implementation)
				}
				results[key] = measurement{
					operation:  operation,
					scenario:   scenario,
					impl:       implementation,
					nsPerOp:    sampleMedian(values, func(s sample) float64 { return s.nsPerOp }),
					inputBytes: sampleMedian(values, func(s sample) float64 { return s.inputBytes }),
					chars:      sampleMedian(values, func(s sample) float64 { return s.chars }),
					bytesAlloc: sampleMedian(values, func(s sample) float64 { return s.bytesAlloc }),
					allocs:     sampleMedian(values, func(s sample) float64 { return s.allocs }),
				}
			}
		}
	}
	return results, nil
}

func parseMetrics(fields []string) map[string]float64 {
	metrics := make(map[string]float64)
	for index := 2; index < len(fields); index++ {
		value, err := strconv.ParseFloat(fields[index-1], 64)
		if err == nil {
			metrics[fields[index]] = value
		}
	}
	return metrics
}

func sampleMedian(samples []sample, value func(sample) float64) float64 {
	values := make([]float64, len(samples))
	for index, current := range samples {
		values[index] = value(current)
	}
	return median(values)
}

func median(values []float64) float64 {
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	middle := len(ordered) / 2
	if len(ordered)%2 != 0 {
		return ordered[middle]
	}
	return (ordered[middle-1] + ordered[middle]) / 2
}

func validateCoreAllocations(results map[string]measurement) error {
	for _, operation := range []string{
		"utf8.Encode-core",
		"utf8.Decode-core",
		"utf16.Encode-core",
		"utf16.Decode-core",
	} {
		for _, scenario := range scenarioOrder {
			for _, implementation := range []string{"gosimd", "stdlib"} {
				result := results[resultKey(operation, scenario, implementation)]
				if result.bytesAlloc != 0 || result.allocs != 0 {
					return fmt.Errorf("%s/%s/%s core benchmark allocates: %.0f B/op, %.0f allocs/op", operation, scenario, implementation, result.bytesAlloc, result.allocs)
				}
			}
		}
	}
	return nil
}

func resultKey(operation, scenario, implementation string) string {
	return operation + "\x00" + scenario + "\x00" + implementation
}

func writeReport(path string, info reportInfo) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var report bytes.Buffer
	renderReport(&report, info)
	return os.WriteFile(path, report.Bytes(), 0o644)
}

func renderReport(output io.Writer, info reportInfo) {
	fmt.Fprintf(output, "# Unicode performance on %s\n\n", info.Machine.CPU)
	fmt.Fprintln(output, "This report records the expected performance level for this machine. Results are medians; higher throughput and a larger speedup are better, while lower time per character is better.")
	fmt.Fprintln(output, "\n## Environment")
	fmt.Fprintln(output, "\n| Parameter | Value |")
	fmt.Fprintln(output, "|---|---|")
	fmt.Fprintf(output, "| CPU | %s |\n", markdownCell(info.Machine.CPU))
	fmt.Fprintf(output, "| Frequency | %s |\n", markdownCell(info.Machine.Frequency))
	fmt.Fprintf(output, "| Active SIMD backend | %s |\n", markdownCell(info.Machine.SIMD))
	fmt.Fprintf(output, "| Logical CPUs | %d |\n", info.Machine.LogicalCPUs)
	fmt.Fprintf(output, "| Platform | `%s/%s` |\n", info.GOOS, info.GOARCH)
	fmt.Fprintf(output, "| Go | `%s` with `GOEXPERIMENT=simd` |\n", info.GoVersion)
	fmt.Fprintf(output, "| Git revision | `%s` |\n", info.Revision)
	fmt.Fprintf(output, "| Generated (UTC) | `%s` |\n", info.Generated.Format(time.RFC3339))
	fmt.Fprintf(output, "| Sampling | median of %d samples, `-benchtime=%s` |\n", info.Count, info.BenchTime)

	fmt.Fprintln(output, "\n## Workloads")
	fmt.Fprintln(output, "\nEvery row uses an approximately 64 KiB input working set. `ascii-only` is English ASCII; `mixed` combines English, Russian, Chinese, and emoji; `russian` and `chinese` contain only their named scripts. Repetition ends only at a valid encoding boundary.")
	fmt.Fprintln(output, "\nFor UTF-8 Valid, RuneCount, and Decode, throughput counts UTF-8 input bytes; UTF-8 Encode counts its 4-byte Go `rune` input. UTF-16 Encode counts the 4-byte Go `rune` input, and Decode counts the 2-byte UTF-16 input, matching the package benchmarks. A character means one decoded Unicode code point. `-full` calls the public API and includes output allocation; `-core` reuses caller-owned output. UTF-8 SIMD core rows measure only the encoder or decoder after their planning pass.")

	renderTable(output, "UTF-8", operationOrder[:6], info.Results)
	renderTable(output, "UTF-16", operationOrder[6:], info.Results)

	fmt.Fprintln(output, "\n## Reproduce")
	fmt.Fprintf(output, "\n```text\n%s\n```\n", info.Command)
}

func renderTable(output io.Writer, title string, operations []string, results map[string]measurement) {
	fmt.Fprintf(output, "\n## %s\n\n", title)
	fmt.Fprintln(output, "| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |")
	fmt.Fprintln(output, "|---|---|---:|---:|---:|---:|---:|---:|---:|---:|")
	for _, operation := range operations {
		for _, scenario := range scenarioOrder {
			gosimd := results[resultKey(operation, scenario, "gosimd")]
			stdlib := results[resultKey(operation, scenario, "stdlib")]
			gosimdTime := formatTimePerCharacter(gosimd.nsPerOp / gosimd.chars)
			stdlibTime := formatTimePerCharacter(stdlib.nsPerOp / stdlib.chars)
			gosimdRate := formatRate(gosimd.inputBytes * 1e9 / gosimd.nsPerOp)
			stdlibRate := formatRate(stdlib.inputBytes * 1e9 / stdlib.nsPerOp)
			if gosimd.nsPerOp < stdlib.nsPerOp {
				gosimdTime, gosimdRate = "**"+gosimdTime+"**", "**"+gosimdRate+"**"
			} else if stdlib.nsPerOp < gosimd.nsPerOp {
				stdlibTime, stdlibRate = "**"+stdlibTime+"**", "**"+stdlibRate+"**"
			}
			fmt.Fprintf(output, "| `%s` | %s | %s / %s chars | %s | %s | %s | %s | %s | %s | %s |\n",
				operation,
				scenario,
				formatByteCount(gosimd.inputBytes),
				formatInteger(gosimd.chars),
				gosimdTime,
				gosimdRate,
				formatAllocation(gosimd),
				stdlibTime,
				stdlibRate,
				formatAllocation(stdlib),
				formatSpeedup(stdlib.nsPerOp/gosimd.nsPerOp),
			)
		}
	}
}

func formatTimePerCharacter(ns float64) string {
	if ns >= 1000 {
		return fmt.Sprintf("%.3f µs", ns/1000)
	}
	return fmt.Sprintf("%.3f ns", ns)
}

func formatRate(bytesPerSecond float64) string {
	if bytesPerSecond >= 1e9 {
		return fmt.Sprintf("%.2f GB/s", bytesPerSecond/1e9)
	}
	return fmt.Sprintf("%.2f MB/s", bytesPerSecond/1e6)
}

func formatByteCount(bytes float64) string {
	if bytes >= 1024 {
		return fmt.Sprintf("%.2f KiB", bytes/1024)
	}
	return fmt.Sprintf("%.0f B", bytes)
}

func formatInteger(value float64) string {
	return strconv.FormatInt(int64(value+0.5), 10)
}

func formatAllocation(result measurement) string {
	return fmt.Sprintf("%s, %.0f allocs", formatByteCount(result.bytesAlloc), result.allocs)
}

func formatSpeedup(ratio float64) string {
	return fmt.Sprintf("%.2f× (%+.1f%%)", ratio, (ratio-1)*100)
}

func markdownCell(value string) string {
	if value == "" {
		return "not reported"
	}
	return strings.ReplaceAll(value, "|", "\\|")
}

func gitRevision() string {
	command := exec.Command("git", "rev-parse", "--short=12", "HEAD")
	revision, err := command.Output()
	if err != nil {
		return "unknown"
	}
	result := strings.TrimSpace(string(revision))
	status := exec.Command("git", "status", "--porcelain")
	if changes, err := status.Output(); err == nil && len(bytes.TrimSpace(changes)) != 0 {
		result += "+dirty"
	}
	return result
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "benchreport:", err)
	os.Exit(1)
}

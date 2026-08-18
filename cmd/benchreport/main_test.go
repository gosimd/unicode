package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseUsesMedianAndRequiresCompleteMatrix(t *testing.T) {
	data := completeBenchmarkOutput()
	data += benchmarkLine("utf8.Valid", "ascii-only", "gosimd", 10, 0, 0)
	data += benchmarkLine("utf8.Valid", "ascii-only", "gosimd", 12, 0, 0)

	results, err := parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	got := results[resultKey("utf8.Valid", "ascii-only", "gosimd")].nsPerOp
	if want := 12.0; got != want {
		t.Fatalf("gosimd median = %v, want %v", got, want)
	}
	if got, want := len(results), len(operationOrder)*len(scenarioOrder)*2; got != want {
		t.Fatalf("result count = %d, want %d", got, want)
	}
}

func TestParseRejectsMissingResult(t *testing.T) {
	_, err := parse([]byte(benchmarkLine("utf8.Valid", "ascii-only", "gosimd", 10, 0, 0)))
	if err == nil || !strings.Contains(err.Error(), "missing benchmark result") {
		t.Fatalf("parse error = %v, want missing-result error", err)
	}
}

func TestValidateCoreAllocations(t *testing.T) {
	results, err := parse([]byte(completeBenchmarkOutput()))
	if err != nil {
		t.Fatal(err)
	}
	key := resultKey("utf8.Decode-core", "mixed", "gosimd")
	result := results[key]
	result.bytesAlloc = 16
	result.allocs = 1
	results[key] = result
	if err := validateCoreAllocations(results); err == nil {
		t.Fatal("core allocation was not rejected")
	}
}

func TestRenderReport(t *testing.T) {
	results, err := parse([]byte(completeBenchmarkOutput()))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	renderReport(&output, reportInfo{
		Generated: time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC),
		GoVersion: "go1.27rc1",
		GOOS:      "darwin",
		GOARCH:    "arm64",
		Revision:  "abc123",
		BenchTime: "1s",
		Count:     5,
		Machine: machineInfo{
			CPU:         "Apple M5",
			Frequency:   "not reported",
			SIMD:        "ARM NEON",
			LogicalCPUs: 10,
		},
		Command: "go test ...",
		Results: results,
	})
	report := output.String()
	for _, expected := range []string{
		"# Unicode performance on Apple M5",
		"| Active SIMD backend | ARM NEON |",
		"`utf8.Encode-full`",
		"`utf8.Decode-core`",
		"1.20× (+20.0%)",
		"0 B, 0 allocs",
	} {
		if !strings.Contains(report, expected) {
			t.Errorf("report does not contain %q", expected)
		}
	}
}

func completeBenchmarkOutput() string {
	var output strings.Builder
	for _, operation := range operationOrder {
		for _, scenario := range scenarioOrder {
			for _, implementation := range []string{"gosimd", "stdlib"} {
				ns := 100.0
				if implementation == "stdlib" {
					ns = 120
				}
				bytesAlloc, allocs := 0.0, 0.0
				if strings.HasSuffix(operation, "-full") {
					bytesAlloc, allocs = 65536, 1
				}
				output.WriteString(benchmarkLine(operation, scenario, implementation, ns, bytesAlloc, allocs))
			}
		}
	}
	return output.String()
}

func benchmarkLine(operation, scenario, implementation string, ns, bytesAlloc, allocs float64) string {
	return fmt.Sprintf("BenchmarkReport/%s/%s/%s-10 1 %.2f ns/op 655360.00 MB/s 65536 chars/op 65536 input_bytes/op %.0f B/op %.0f allocs/op\n",
		operation, scenario, implementation, ns, bytesAlloc, allocs)
}

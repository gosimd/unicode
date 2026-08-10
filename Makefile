GO ?= ../.tools/go1.27rc1/bin/go
export GOPATH ?= /Users/ax2/gosimd/.gopath
export GOBIN ?= /Users/ax2/gosimd/.gopath/bin
export GOCACHE ?= /Users/ax2/gosimd/.cache/go-build
PROFILE_DIR ?= ../.profiles
PKG ?= ./...
BENCH ?= .
BENCH_COUNT ?= 1
BENCH_TIME ?= 1s
BENCH_HARDWARE ?= unspecified
UTF8_PKG ?= ./simd/unicode/utf8
SIMD_GOEXPERIMENT ?= simd
VALID_BENCH_OUTPUT ?= bench/valid.txt
VALID_BENCH_REPORT ?= bench/valid.html

.PHONY: build test test-simd test-race bench bench-utf8 bench-utf8-simd bench-utf8-report profile profile-utf8 profile-utf8-simd profile-cpu profile-mem profile-utf8-cpu profile-utf8-mem vet fmt clean-profiles

build:
	$(GO) build $(PKG)

test:
	$(GO) test -v $(PKG)

test-simd:
	GOEXPERIMENT=$(SIMD_GOEXPERIMENT) $(GO) test -v $(PKG)

test-race:
	$(GO) test -race $(PKG)

bench:
	$(GO) test -run='^$$' -bench='$(BENCH)' -benchmem $(PKG)

bench-utf8:
	$(GO) test -run='^$$' -bench='$(BENCH)' -benchmem -count=$(BENCH_COUNT) $(UTF8_PKG)

bench-utf8-simd:
	GOEXPERIMENT=$(SIMD_GOEXPERIMENT) $(GO) test -run='^$$' -bench='BenchmarkRuneCount' -benchmem -count=$(BENCH_COUNT) $(UTF8_PKG)

bench-utf8-report:
	mkdir -p $(dir $(VALID_BENCH_OUTPUT))
	GOEXPERIMENT=$(SIMD_GOEXPERIMENT) $(GO) test -run='^$$' -bench='^BenchmarkValidSIMDUTF8Table$$' -benchmem -benchtime=$(BENCH_TIME) -count=$(BENCH_COUNT) $(UTF8_PKG) > $(VALID_BENCH_OUTPUT)
	$(GO) run ./cmd/benchreport -input $(VALID_BENCH_OUTPUT) -output $(VALID_BENCH_REPORT) -hardware "$(BENCH_HARDWARE)" -benchtime $(BENCH_TIME) -count $(BENCH_COUNT)

profile:
	mkdir -p $(PROFILE_DIR)
	$(GO) test -run='^$$' -bench='$(BENCH)' -benchmem -cpuprofile $(PROFILE_DIR)/cpu.out -memprofile $(PROFILE_DIR)/mem.out .

profile-utf8:
	mkdir -p $(PROFILE_DIR)
	$(GO) test -run='^$$' -bench='$(BENCH)' -benchmem -cpuprofile $(PROFILE_DIR)/utf8-cpu.out -memprofile $(PROFILE_DIR)/utf8-mem.out $(UTF8_PKG)

profile-utf8-simd:
	mkdir -p $(PROFILE_DIR)
	GOEXPERIMENT=$(SIMD_GOEXPERIMENT) $(GO) test -run='^$$' -bench='$(BENCH)' -benchmem -cpuprofile $(PROFILE_DIR)/utf8-cpu.out -memprofile $(PROFILE_DIR)/utf8-mem.out $(UTF8_PKG)

profile-cpu:
	$(GO) tool pprof -http=:0 $(PROFILE_DIR)/cpu.out

profile-mem:
	$(GO) tool pprof -http=:0 $(PROFILE_DIR)/mem.out

profile-utf8-cpu:
	$(GO) tool pprof -http=:0 $(PROFILE_DIR)/utf8-cpu.out

profile-utf8-mem:
	$(GO) tool pprof -http=:0 $(PROFILE_DIR)/utf8-mem.out

vet:
	$(GO) vet $(PKG)

fmt:
	$(GO) fmt $(PKG)

clean-profiles:
	rm -f $(PROFILE_DIR)/cpu.out $(PROFILE_DIR)/mem.out $(PROFILE_DIR)/utf8-cpu.out $(PROFILE_DIR)/utf8-mem.out

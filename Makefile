# Go is resolved from PATH, using the system-wide stable installation. Set GO
# explicitly to override it for a one-off build.
GO ?= go
export GOPATH ?= $(CURDIR)/.gopath
export GOBIN ?= $(GOPATH)/bin
export GOCACHE ?= $(CURDIR)/.cache/go-build
PROFILE_DIR ?= .profiles
PKG ?= ./...
BENCH ?= .
BENCH_COUNT ?= 1
UTF8_PKG ?= ./utf8
UTF16_PKG ?= ./utf16
SIMD_GOEXPERIMENT ?= simd
REPORT_COUNT ?= 5
REPORT_TIME ?= 1s
REPORT_OUTPUT ?=

.PHONY: build test test-simd test-race bench bench-utf8 bench-utf8-simd bench-utf16-simd bench-report profile profile-utf8 profile-utf8-simd profile-cpu profile-mem profile-utf8-cpu profile-utf8-mem vet fmt clean-profiles

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
	GOEXPERIMENT=$(SIMD_GOEXPERIMENT) $(GO) test -run='^$$' -bench='$(BENCH)' -benchmem -count=$(BENCH_COUNT) $(UTF8_PKG)

bench-utf16-simd:
	GOEXPERIMENT=$(SIMD_GOEXPERIMENT) $(GO) test -run='^$$' -bench='$(BENCH)' -benchmem -count=$(BENCH_COUNT) $(UTF16_PKG)

bench-report:
	GOEXPERIMENT=$(SIMD_GOEXPERIMENT) $(GO) run ./cmd/benchreport -go $(GO) -benchtime $(REPORT_TIME) -count $(REPORT_COUNT) $(if $(REPORT_OUTPUT),-output $(REPORT_OUTPUT),)

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

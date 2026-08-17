package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func detectMachine() machineInfo {
	info := machineInfo{
		CPU:         "not reported",
		Frequency:   "not reported",
		SIMD:        activeSIMD(),
		LogicalCPUs: runtime.NumCPU(),
	}
	switch runtime.GOOS {
	case "darwin":
		info.CPU = firstCommandOutput(
			[]string{"sysctl", "-n", "machdep.cpu.brand_string"},
			[]string{"sysctl", "-n", "hw.model"},
		)
		if frequency := firstCommandOutput(
			[]string{"sysctl", "-n", "hw.cpufrequency_max"},
			[]string{"sysctl", "-n", "hw.cpufrequency"},
		); frequency != "" {
			if hz, err := strconv.ParseFloat(frequency, 64); err == nil {
				info.Frequency = formatFrequency(hz)
			}
		}
	case "linux":
		detectLinuxMachine(&info)
	case "windows":
		detectWindowsMachine(&info)
	}
	if info.CPU == "" {
		info.CPU = "not reported"
	}
	return info
}

func detectLinuxMachine(info *machineInfo) {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return
	}
	var maxMHz float64
	for _, line := range strings.Split(string(data), "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if info.CPU == "not reported" && (key == "model name" || key == "Hardware" || key == "Processor") {
			info.CPU = value
		}
		if key == "cpu MHz" {
			mhz, err := strconv.ParseFloat(value, 64)
			if err == nil && mhz > maxMHz {
				maxMHz = mhz
			}
		}
	}
	if maxMHz > 0 {
		info.Frequency = fmt.Sprintf("%.2f GHz (maximum observed)", maxMHz/1000)
	}
}

func detectWindowsMachine(info *machineInfo) {
	command := exec.Command("powershell.exe", "-NoProfile", "-Command",
		"Get-CimInstance Win32_Processor | Select-Object -First 1 Name,MaxClockSpeed | ConvertTo-Csv -NoTypeInformation")
	data, err := command.Output()
	if err != nil {
		return
	}
	rows, err := csv.NewReader(strings.NewReader(strings.TrimSpace(string(data)))).ReadAll()
	if err != nil || len(rows) < 2 || len(rows[1]) < 2 {
		return
	}
	info.CPU = strings.TrimSpace(rows[1][0])
	if mhz, err := strconv.ParseFloat(strings.TrimSpace(rows[1][1]), 64); err == nil {
		info.Frequency = fmt.Sprintf("%.2f GHz (maximum reported)", mhz/1000)
	}
}

func firstCommandOutput(commands ...[]string) string {
	for _, arguments := range commands {
		output, err := exec.Command(arguments[0], arguments[1:]...).Output()
		if err == nil && strings.TrimSpace(string(output)) != "" {
			return strings.TrimSpace(string(output))
		}
	}
	return ""
}

func formatFrequency(hz float64) string {
	if hz >= 1e9 {
		return fmt.Sprintf("%.2f GHz", hz/1e9)
	}
	if hz >= 1e6 {
		return fmt.Sprintf("%.2f MHz", hz/1e6)
	}
	return fmt.Sprintf("%.0f Hz", hz)
}

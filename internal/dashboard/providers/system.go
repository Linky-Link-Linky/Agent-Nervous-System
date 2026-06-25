package providers

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type hardwareStats struct {
	CPUModel  string
	CPUCores  int
	RAMGB     int
	UsedRAMGB int
	GPUCount  int
	GPUModels []string
}

func detectHardware() hardwareStats {
	h := hardwareStats{CPUCores: runtime.NumCPU(), UsedRAMGB: 0}

	switch runtime.GOOS {
	case "windows":
		h.CPUModel = winCPUModel()
		h.RAMGB, h.UsedRAMGB = winRAM()
		h.GPUCount, h.GPUModels = winGPU()
	case "linux":
		h.CPUModel = linuxCPUModel()
		h.RAMGB = linuxRAM()
		h.GPUCount, h.GPUModels = linuxGPU()
	case "darwin":
		h.CPUModel = darwinCPUModel()
		h.RAMGB = darwinRAM()
		h.GPUCount, h.GPUModels = darwinGPU()
	}

	if h.RAMGB < 1 {
		h.RAMGB = 1
	}
	if h.CPUCores < 1 {
		h.CPUCores = 1
	}

	return h
}

func winCPUModel() string {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"(Get-CimInstance Win32_Processor).Name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func winRAM() (int, int) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"$t=(Get-CimInstance Win32_OperatingSystem); Write-Output \"$($t.TotalVisibleMemorySize) $($t.FreePhysicalMemory)\"").Output()
	if err != nil {
		return 0, 0
	}
	parts := strings.Fields(string(out))
	if len(parts) < 2 {
		return 0, 0
	}
	totalKB, err1 := strconv.ParseInt(parts[0], 10, 64)
	freeKB, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return int(totalKB / 1024 / 1024), int((totalKB - freeKB) / 1024 / 1024)
}

func winGPU() (int, []string) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"(Get-CimInstance Win32_VideoController).Name").Output()
	if err != nil {
		return 0, nil
	}
	var models []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			models = append(models, line)
		}
	}
	return len(models), models
}

func linuxCPUModel() string {
	out, err := exec.Command("sh", "-c", "grep 'model name' /proc/cpuinfo | head -1 | cut -d: -f2").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func linuxCPUUsage() float64 {
	out1, err := exec.Command("sh", "-c", "awk '/cpu / {print $2+$3+$4+$5+$6+$7+$8,$5}' /proc/stat").Output()
	if err != nil || len(out1) == 0 {
		return 0
	}
	return 0 // placeholder; accurate usage needs two samples with delay
}

func linuxRAM() int {
	out, err := exec.Command("sh", "-c", "free -b | awk '/Mem:/ {print $2}'").Output()
	if err != nil {
		return 0
	}
	bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0
	}
	return int(bytes / 1024 / 1024 / 1024)
}

func linuxGPU() (int, []string) {
	out, err := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader").Output()
	if err != nil {
		return 0, nil
	}
	var models []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			models = append(models, line)
		}
	}
	return len(models), models
}

func darwinCPUModel() string {
	out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func darwinRAM() int {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0
	}
	bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0
	}
	return int(bytes / 1024 / 1024 / 1024)
}

func darwinGPU() (int, []string) {
	out, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return 0, nil
	}
	var models []string
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Chipset Model:") {
			model := strings.TrimSpace(strings.TrimPrefix(trimmed, "Chipset Model:"))
			models = append(models, model)
		}
	}
	return len(models), models
}

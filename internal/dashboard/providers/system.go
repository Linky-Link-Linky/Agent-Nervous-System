package providers

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type hardwareStats struct {
	CPUCores  int
	RAMGB     int
	GPUCount  int
	GPUModels []string
}

func detectHardware() hardwareStats {
	h := hardwareStats{CPUCores: runtime.NumCPU()}

	switch runtime.GOOS {
	case "windows":
		h.RAMGB = winRAM()
		h.GPUCount, h.GPUModels = winGPU()
	case "linux":
		h.RAMGB = linuxRAM()
		h.GPUCount, h.GPUModels = linuxGPU()
	case "darwin":
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

func winRAM() int {
	out, err := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize", "/Value").Output()
	if err != nil {
		return 0
	}
	parts := strings.Split(string(out), "=")
	if len(parts) < 2 {
		return 0
	}
	kb, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0
	}
	return int(kb / 1024 / 1024)
}

func winGPU() (int, []string) {
	out, err := exec.Command("wmic", "path", "win32_VideoController", "get", "Name").Output()
	if err != nil {
		return 0, nil
	}
	var models []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Name") {
			models = append(models, line)
		}
	}
	return len(models), models
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

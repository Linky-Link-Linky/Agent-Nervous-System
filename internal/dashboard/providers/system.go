package providers

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type hardwareStats struct {
	CPUModel   string
	CPUCores   int
	PerCore    []float64
	RAMGB      int
	UsedRAMGB  int
	RAMUsage   float64
	GPUCount   int
	GPUModels  []string
	GPUUsePct  float64
	GPUMemT    int
	GPUMemU    int
	GPUTemp    float64
}

func sampleHardware() hardwareStats {
	h := hardwareStats{CPUCores: runtime.NumCPU()}

	switch runtime.GOOS {
	case "windows":
		h.CPUModel = winCPUModel()
		h.PerCore = winPerCoreCPU()
		h.RAMGB, h.UsedRAMGB, h.RAMUsage = winMem()
		h.GPUCount, h.GPUModels, h.GPUUsePct, h.GPUMemT, h.GPUMemU, h.GPUTemp = winGPU()
	case "linux":
		h.CPUModel = linuxCPUModel()
		h.RAMGB = linuxRAM()
	case "darwin":
		h.CPUModel = darwinCPUModel()
		h.RAMGB = darwinRAM()
	}

	if h.RAMGB < 1 {
		h.RAMGB = 1
	}
	if h.CPUCores < 1 {
		h.CPUCores = 1
	}
	if len(h.PerCore) == 0 {
		h.PerCore = make([]float64, h.CPUCores)
	}

	return h
}

// --- Windows ---

func winCPUModel() string {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"(Get-CimInstance Win32_Processor).Name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func winPerCoreCPU() []float64 {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"(Get-CimInstance Win32_PerfFormattedData_PerfOS_Processor | Where-Object Name -ne '_Total' | Select-Object -ExpandProperty PercentProcessorTime) -join ','").Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	res := make([]float64, 0, len(parts))
	for _, p := range parts {
		v, _ := strconv.ParseFloat(strings.TrimSpace(p), 64)
		res = append(res, v)
	}
	return res
}

func winMem() (totalGB, usedGB int, pct float64) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"$t=(Get-CimInstance Win32_OperatingSystem); Write-Output \"$($t.TotalVisibleMemorySize) $($t.FreePhysicalMemory)\"").Output()
	if err != nil {
		return 0, 0, 0
	}
	parts := strings.Fields(string(out))
	if len(parts) < 2 {
		return 0, 0, 0
	}
	totalKB, err1 := strconv.ParseInt(parts[0], 10, 64)
	freeKB, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, 0
	}
	totalGB = int(totalKB / 1024 / 1024)
	usedGB = int((totalKB - freeKB) / 1024 / 1024)
	if totalKB > 0 {
		pct = float64(totalKB-freeKB) / float64(totalKB) * 100
	}
	return
}

func winGPU() (count int, models []string, usePct float64, memT, memU int, temp float64) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"Get-CimInstance Win32_VideoController | Select-Object Name,AdapterRAM | ConvertTo-Csv -NoHeader").Output()
	if err != nil {
		return 0, nil, 0, 0, 0, 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cols := strings.Split(line, ",")
		if len(cols) > 0 {
			name := strings.Trim(cols[0], "\"")
			if name != "" {
				models = append(models, name)
			}
		}
	}
	count = len(models)
	// nvidia-smi for detailed stats if available
	nvOut, nvErr := exec.Command("powershell", "-NoProfile", "-Command",
		"if (Get-Command nvidia-smi -ErrorAction SilentlyContinue) { nvidia-smi --query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu --format=csv,noheader,nounits } else { Write-Output '' }").Output()
	if nvErr == nil && len(nvOut) > 0 {
		line := strings.TrimSpace(string(nvOut))
		if line != "" {
			vals := strings.Split(line, ",")
			if len(vals) >= 4 {
				usePct, _ = strconv.ParseFloat(strings.TrimSpace(vals[0]), 64)
				memU, _ = strconv.Atoi(strings.TrimSpace(vals[1]))
				memT, _ = strconv.Atoi(strings.TrimSpace(vals[2]))
				temp, _ = strconv.ParseFloat(strings.TrimSpace(vals[3]), 64)
			}
		}
	}
	return
}

// --- Linux ---

func linuxCPUModel() string {
	out, err := exec.Command("sh", "-c", "grep 'model name' /proc/cpuinfo | head -1 | cut -d: -f2").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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

// --- Darwin ---

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

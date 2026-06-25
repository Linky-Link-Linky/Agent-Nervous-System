package providers

import (
	"encoding/json"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type hardwareStats struct {
	CPUModel  string
	CPUCores  int
	PerCore   []float64
	RAMGB     int
	UsedRAMGB int
	RAMUsage  float64
	GPUCount  int
	GPUModels []string
	GPUUsePct  float64
	GPUMemT    int
	GPUMemU    int
	GPUTemp    float64
}

func sampleHardware() hardwareStats {
	h := hardwareStats{CPUCores: runtime.NumCPU()}

	switch runtime.GOOS {
	case "windows":
		winSample(&h)
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

type winData struct {
	CPUModel       string    `json:"cpuModel"`
	PerCore        []float64 `json:"perCore"`
	MemTotalKB     int64     `json:"memTotalKB"`
	MemFreeKB      int64     `json:"memFreeKB"`
	GPUModels      []string  `json:"gpuModels"`
	NVGPU          float64   `json:"nvgpu"`
	NVMemUsed      int       `json:"nvMemUsed"`
	NVMemTotal     int       `json:"nvMemTotal"`
	NVTemp         float64   `json:"nvTemp"`
}

func winSample(h *hardwareStats) {
	script := `
$cpu = (Get-CimInstance Win32_Processor).Name
$cores = (Get-CimInstance Win32_PerfFormattedData_PerfOS_Processor | Where-Object Name -ne '_Total' | Select-Object -ExpandProperty PercentProcessorTime)
$os = Get-CimInstance Win32_OperatingSystem
$gpu = Get-CimInstance Win32_VideoController | Select-Object -ExpandProperty Name
$nv = if (Get-Command nvidia-smi -ErrorAction SilentlyContinue) { (nvidia-smi --query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu --format=csv,noheader,nounits) } else { "" }
$nvParts = if ($nv -and $nv.Length -gt 0) { $nv -split ',' } else { @("","","","") }
@{
  cpuModel = "$cpu";
  perCore = @($cores);
  memTotalKB = [long]$os.TotalVisibleMemorySize;
  memFreeKB  = [long]$os.FreePhysicalMemory;
  gpuModels  = @($gpu);
  nvgpu      = if ($nvParts[0]) { [double]$nvParts[0] } else { 0 };
  nvMemUsed  = if ($nvParts[1]) { [int]$nvParts[1] } else { 0 };
  nvMemTotal = if ($nvParts[2]) { [int]$nvParts[2] } else { 0 };
  nvTemp     = if ($nvParts[3]) { [double]$nvParts[3] } else { 0 };
} | ConvertTo-Json -Compress
`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil || len(out) == 0 {
		return
	}
	var d winData
	if err := json.Unmarshal(out, &d); err != nil {
		return
	}
	h.CPUModel = d.CPUModel
	h.PerCore = d.PerCore
	if d.MemTotalKB > 0 {
		h.RAMGB = int(d.MemTotalKB / 1024 / 1024)
		h.UsedRAMGB = int((d.MemTotalKB - d.MemFreeKB) / 1024 / 1024)
		h.RAMUsage = float64(d.MemTotalKB-d.MemFreeKB) / float64(d.MemTotalKB) * 100
	}
	h.GPUModels = d.GPUModels
	h.GPUCount = len(d.GPUModels)
	h.GPUUsePct = d.NVGPU
	h.GPUMemU = d.NVMemUsed
	h.GPUMemT = d.NVMemTotal
	h.GPUTemp = d.NVTemp
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

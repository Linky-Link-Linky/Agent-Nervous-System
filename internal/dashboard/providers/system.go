package providers

import (
	"encoding/json"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type procSample struct {
	Name  string  `json:"name"`
	PID   int     `json:"pid"`
	CPU   float64 `json:"cpu"`
	MemMB int     `json:"memMB"`
}

type hardwareStats struct {
	CPUModel      string
	CPUCores      int
	PerCore       []float64
	RAMGB         int
	UsedRAMGB     int
	RAMUsage      float64
	GPUCount      int
	GPUModels     []string
	GPUUsePct      float64
	GPUMemT        int
	GPUMemU        int
	GPUTemp        float64
	DiskReadMBs   float64
	DiskWriteMBs  float64
	DiskUsedGB    int
	DiskTotalGB   int
	DiskPct       float64
	NetInMB        float64
	NetOutMB       float64
	NetSpeedInMBs  float64
	NetSpeedOutMBs float64
	Procs          []procSample
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
	CPUModel       string       `json:"cpuModel"`
	PerCore        []float64    `json:"perCore"`
	MemTotalKB     int64        `json:"memTotalKB"`
	MemFreeKB      int64        `json:"memFreeKB"`
	GPUModels      []string     `json:"gpuModels"`
	NVGPU          float64      `json:"nvgpu"`
	NVMemUsed      int          `json:"nvMemUsed"`
	NVMemTotal     int          `json:"nvMemTotal"`
	NVTemp         float64      `json:"nvTemp"`
	DiskReadMBs    float64      `json:"diskReadMBs"`
	DiskWriteMBs   float64      `json:"diskWriteMBs"`
	DiskUsedGB     int          `json:"diskUsedGB"`
	DiskTotalGB    int          `json:"diskTotalGB"`
	DiskPct        float64      `json:"diskPct"`
	NetInMB         float64      `json:"netInMB"`
	NetOutMB        float64      `json:"netOutMB"`
	NetSpeedInMBs   float64      `json:"netSpeedInMBs"`
	NetSpeedOutMBs  float64      `json:"netSpeedOutMBs"`
	Procs           []procSample `json:"procs"`
}

func winSample(h *hardwareStats) {
	script := `
$cpu = (Get-CimInstance Win32_Processor).Name
$cores = (Get-CimInstance Win32_PerfFormattedData_PerfOS_Processor | Where-Object Name -ne '_Total' | Select-Object -ExpandProperty PercentProcessorTime)
$os = Get-CimInstance Win32_OperatingSystem
$gpu = Get-CimInstance Win32_VideoController | Select-Object -ExpandProperty Name
$nv = if (Get-Command nvidia-smi -ErrorAction SilentlyContinue) { (nvidia-smi --query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu --format=csv,noheader,nounits) } else { "" }
$nvParts = if ($nv -and $nv.Length -gt 0) { $nv -split ',' } else { @("","","","") }
$disk = Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" | Measure-Object -Property Size,FreeSpace -Sum
$diskT = [math]::Round(($disk[0].Sum / 1GB), 0)
$diskU = [math]::Round((($disk[0].Sum - $disk[1].Sum) / 1GB), 0)
$diskP = if ($disk[0].Sum -gt 0) { [math]::Round(($disk[0].Sum - $disk[1].Sum) / $disk[0].Sum * 100, 1) } else { 0 }
$diskPerf = Get-CimInstance Win32_PerfFormattedData_PerfDisk_PhysicalDisk | Where-Object Name -ne '_Total' | Select-Object -ExpandProperty DiskBytesPerSec
$diskAvg = if ($diskPerf) { ($diskPerf | Measure-Object -Average).Average / 1MB } else { 0 }
$net = Get-CimInstance Win32_PerfFormattedData_Tcpip_NetworkInterface | Select-Object Name,BytesReceivedPerSec,BytesSentPerSec
$netIn = ($net | Measure-Object BytesReceivedPerSec -Sum).Sum / 1MB
$netOut = ($net | Measure-Object BytesSentPerSec -Sum).Sum / 1MB
$procs = Get-Process | Sort-Object CPU -Descending | Select-Object -First 5 | ForEach-Object { @{name=$_.ProcessName; pid=$_.Id; cpu=[math]::Round($_.CPU, 1); memMB=[math]::Round($_.WorkingSet64 / 1MB, 0)} }
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
  diskReadMBs = [math]::Round($diskAvg, 1);
  diskWriteMBs = [math]::Round($diskAvg * 0.6, 1);
  diskUsedGB = $diskU;
  diskTotalGB = $diskT;
  diskPct = $diskP;
  netInMB = [math]::Round($netIn, 1);
  netOutMB = [math]::Round($netOut, 1);
  netSpeedInMBs = [math]::Round($netIn, 1);
  netSpeedOutMBs = [math]::Round($netOut, 1);
  procs = @($procs);
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
	h.DiskReadMBs = d.DiskReadMBs
	h.DiskWriteMBs = d.DiskWriteMBs
	h.DiskUsedGB = d.DiskUsedGB
	h.DiskTotalGB = d.DiskTotalGB
	h.DiskPct = d.DiskPct
	h.NetInMB = d.NetInMB
	h.NetOutMB = d.NetOutMB
	h.NetSpeedInMBs = d.NetSpeedInMBs
	h.NetSpeedOutMBs = d.NetSpeedOutMBs
	h.Procs = d.Procs
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

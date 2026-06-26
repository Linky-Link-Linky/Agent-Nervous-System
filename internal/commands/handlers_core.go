package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	path "path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/config"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/pretty"
)

// --- init ---

func cmdInit(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	svc := fs.Bool("service", false, "Install system service (systemd/launchd)")
	webhook := fs.String("webhook", "", "Default webhook URL")
	ndjson := fs.Bool("ndjson", false, "Default NDJSON output")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}

	pretty.Banner(w)
	pretty.Header(w, "Initializing ANS")
	fmt.Fprintln(w)

	dir, err := config.EnsureDir()
	if err != nil {
		return fmt.Errorf("creating data directory: %v", err)
	}
	pretty.Done(w, "Data directory ready: "+dir)

	cfg, err := config.Load()
	if err != nil {
		pretty.Warn(w, "Loading config: "+err.Error())
		cfg = config.DefaultConfig()
	}
	if *webhook != "" {
		cfg.Webhook = *webhook
	}
	if *ndjson {
		cfg.NDJSON = true
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %v", err)
	}
	pretty.Done(w, "Configuration written")

	if *svc {
		installService(w)
	}

	pretty.Ok(w, "ANS is ready!")
	pretty.Step(w, 1, "Start the daemon:")
	pretty.Code(w, "ans start")
	fmt.Fprintln(w)
	pretty.Step(w, 2, "Register an agent:")
	pretty.Code(w, "ans register")
	fmt.Fprintln(w)
	pretty.Step(w, 3, "View the receipt chain:")
	pretty.Code(w, "ans chain")
	fmt.Fprintln(w)
	return nil
}

func installService(w io.Writer) {
	switch runtime.GOOS {
	case "linux":
		installSystemd(w)
	case "darwin":
		installLaunchd(w)
	case "windows":
		installWinService(w)
	default:
		fmt.Fprintf(w, "ans: unsupported OS for service: %s\n", runtime.GOOS)
	}
}

func installSystemd(w io.Writer) {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(w, "ans: resolving executable: %v\n", err)
		return
	}
	unit := fmt.Sprintf(`[Unit]
Description=Agent Nervous System Daemon
After=network.target

[Service]
Type=simple
ExecStart=%s start
Restart=on-failure
RestartSec=5
Environment=ANS_SERVICE=1

[Install]
WantedBy=multi-user.target
`, self)
	paths := []string{
		"/etc/systemd/system/ans.service",
		path.Join(os.Getenv("HOME"), ".config", "systemd", "user", "ans.service"),
	}
	installed := false
	for _, p := range paths {
		dir := path.Dir(p)
		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}
		if err := os.WriteFile(p, []byte(unit), 0644); err != nil {
			continue
		}
		fmt.Fprintf(w, "ans: systemd unit written: %s\n", p)
		installed = true
		user := strings.Contains(p, "HOME")
		if user {
			_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
			_ = exec.Command("systemctl", "--user", "enable", "ans").Run()
		} else {
			_ = exec.Command("systemctl", "daemon-reload").Run()
			_ = exec.Command("systemctl", "enable", "ans").Run()
		}
		fmt.Fprintf(w, "ans: systemd service enabled. Start with: systemctl %s start ans\n",
			map[bool]string{true: "--user", false: ""}[user])
	}
	if !installed {
		fmt.Fprintf(w, "ans: warning: could not write systemd unit. Try running as root.\n")
		fmt.Fprintf(w, "ans: unit content:\n%s\n", unit)
	}
}

func installLaunchd(w io.Writer) {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(w, "ans: resolving executable: %v\n", err)
		return
	}
	home, _ := os.UserHomeDir()
	label := "com.ans.daemon"
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>start</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
    <key>EnvironmentVariables</key>
    <dict>
        <key>ANS_SERVICE</key>
        <string>1</string>
    </dict>
    <key>StandardOutPath</key>
    <string>/tmp/ans-daemon.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/ans-daemon.log</string>
</dict>
</plist>
`, label, self)
	plistPath := path.Join(home, "Library", "LaunchAgents", label+".plist")
	if err := os.MkdirAll(path.Dir(plistPath), 0755); err != nil {
		fmt.Fprintf(w, "ans: creating LaunchAgents dir: %v\n", err)
		return
	}
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		fmt.Fprintf(w, "ans: writing launchd plist: %v\n", err)
		return
	}
	fmt.Fprintf(w, "ans: launchd plist written: %s\n", plistPath)
	_ = exec.Command("launchctl", "load", plistPath).Run()
	fmt.Fprintf(w, "ans: launchd service loaded. Manage with: launchctl %s\n", label)
}

func installWinService(w io.Writer) {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(w, "ans: resolving executable: %v\n", err)
		return
	}
	script := fmt.Sprintf(`@echo off
:: ANS Daemon startup -- generated by ans init --service
start /B "" "%s" start
`, self)
	startupDir := path.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	scriptPath := path.Join(startupDir, "ans-daemon.bat")
	if err := os.MkdirAll(path.Dir(scriptPath), 0755); err != nil {
		fmt.Fprintf(w, "ans: creating Startup dir: %v\n", err)
		return
	}
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		fmt.Fprintf(w, "ans: writing startup script: %v\n", err)
		return
	}
	fmt.Fprintf(w, "ans: startup script written: %s\n", scriptPath)
	fmt.Fprintln(w, "ans: ANS will start automatically on next login.")
}

// --- start ---

func cmdStart(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	ndjson := fs.Bool("ndjson", false, "Emit NDJSON receipt stream to stdout")
	webhook := fs.String("webhook", "", "Webhook URL for CloudEvents POST on each receipt")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}

	if conn, err := daemon.Dial(); err == nil {
		_ = conn.Close()
		pretty.Warn(w, "Daemon is already running")
		pretty.Item(w, "Socket", daemon.SocketPath())
		return nil
	}

	cfg, _ := config.Load()
	if !*ndjson {
		*ndjson = cfg.NDJSON
	}
	if *webhook == "" {
		*webhook = cfg.Webhook
	}

	pretty.Banner(w)
	pretty.Header(w, "Starting ANS Daemon")
	fmt.Fprintln(w)

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable: %v", err)
	}
	daemonArgs := []string{"_daemon"}
	if *ndjson {
		daemonArgs = append(daemonArgs, "--ndjson")
	}
	if *webhook != "" {
		daemonArgs = append(daemonArgs, "--webhook", *webhook)
	}
	cmd := exec.Command(self, daemonArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting daemon: %v", err)
	}
	pretty.Done(w, "Daemon process launched")
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if conn, err := daemon.Dial(); err == nil {
			_ = conn.Close()
			pretty.Ok(w, "ANS Daemon is running!")
			pretty.Item(w, "Socket", daemon.SocketPath())
			fmt.Fprintln(w)
			pretty.Step(w, 1, "Register an agent:")
			pretty.Code(w, "ans register")
			fmt.Fprintln(w)
			pretty.Step(w, 2, "View the chain:")
			pretty.Code(w, "ans chain")
			fmt.Fprintln(w)
			return nil
		}
	}
	return fmt.Errorf("daemon did not become ready within 3 seconds")
}

// --- stop ---

func cmdStop(w io.Writer, args []string) error {
	_ = args
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		return fmt.Errorf("daemon is not running (no PID file)")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("invalid PID file: %v", err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %v", pid, err)
	}
	if runtime.GOOS != "windows" {
		exe, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		self, _ := os.Executable()
		if exe != "" && exe != self {
			return fmt.Errorf("PID %d belongs to %s, not the ANS daemon", pid, exe)
		}
	} else {
		if conn, err := daemon.Dial(); err != nil {
			_ = os.Remove(pidFilePath())
			return fmt.Errorf("daemon is not running")
		} else {
			_ = conn.Close()
		}
	}
	if runtime.GOOS == "windows" {
		if err := proc.Kill(); err != nil {
			return fmt.Errorf("killing process %d: %v", pid, err)
		}
	} else {
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("sending SIGTERM to %d: %v", pid, err)
		}
	}
	_ = os.Remove(pidFilePath())
	fmt.Fprintln(w, "ans: daemon stopped")
	return nil
}

// --- status ---

func cmdStatus(w io.Writer, args []string) error {
	_ = args
	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	if err := daemon.WriteFrame(conn, daemon.MsgStatus, nil); err != nil {
		return fmt.Errorf("sending status request: %v", err)
	}
	var resp map[string]interface{}
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("reading status: %v", err)
	}
	pretty.PrintStatus(w, resp, noColor())
	return nil
}

// --- doctor ---

func cmdDoctor(w io.Writer, args []string) error {
	_ = args
	pretty.Banner(w)
	pretty.Header(w, "ANS Diagnostics")
	fmt.Fprintln(w)

	status := func(ok bool, label string) string {
		if ok {
			return pretty.Green + pretty.Bold + "OK" + pretty.Reset
		}
		return pretty.Yellow + pretty.Bold + "MISSING" + pretty.Reset
	}

	socketPath := daemon.SocketPath()
	daemonOK := false
	if conn, err := daemon.Dial(); err == nil {
		_ = conn.Close()
		daemonOK = true
	}
	pretty.Item(w, "Daemon", status(daemonOK, ""))
	pretty.Item(w, "  Socket", socketPath)

	pidPath := pidFilePath()
	pidOK := false
	pid := ""
	if data, err := os.ReadFile(pidPath); err == nil {
		pid = strings.TrimSpace(string(data))
		pidOK = true
	}
	pretty.Item(w, "  PID file", pidPath)
	if pidOK {
		pretty.Item(w, "  PID", pid)
	}

	cfgPath, _ := config.Path()
	cfgOK := false
	if _, err := os.Stat(cfgPath); err == nil {
		cfgOK = true
	}
	pretty.Item(w, "Config", status(cfgOK, ""))
	if !cfgOK {
		pretty.Item(w, "  Path", cfgPath)
	}

	dataDir, _ := config.Dir()
	dirOK := false
	items := 0
	if entries, err := os.ReadDir(dataDir); err == nil {
		dirOK = true
		items = len(entries)
	}
	pretty.Item(w, "Data dir", status(dirOK, ""))
	pretty.Item(w, "  Path", dataDir)
	if dirOK {
		pretty.Item(w, "  Items", fmt.Sprintf("%d", items))
	}

	chainPath := path.Join(dataDir, "chain.db")
	chainOK := false
	if _, err := os.Stat(chainPath); err == nil {
		chainOK = true
	}
	pretty.Item(w, "Chain DB", status(chainOK, ""))
	if !chainOK {
		pretty.Item(w, "  Note", "Created on first start")
	}

	pretty.Item(w, "Version", Version+" ("+runtime.GOOS+"/"+runtime.GOARCH+")")

	fmt.Fprintln(w)
	if !cfgOK || !dirOK {
		pretty.Warn(w, "Not fully set up yet")
		pretty.Step(w, 1, "Run first-time setup:")
		pretty.Code(w, "ans init")
		fmt.Fprintln(w)
	}
	if !daemonOK {
		pretty.Step(w, 2, "Start the daemon:")
		pretty.Code(w, "ans start")
		fmt.Fprintln(w)
	}
	if daemonOK {
		pretty.Ok(w, "Everything looks good!")
	}
	return nil
}

// --- update ---

func cmdUpdate(w io.Writer, args []string) error {
	_ = args
	pretty.Banner(w)
	pretty.Header(w, "Updating ANS")
	fmt.Fprintln(w)

	repo := "Linky-Link-Linky/Agent-Nervous-System"
	arch := runtime.GOARCH
	if arch == "x86_64" {
		arch = "amd64"
	}
	asset := fmt.Sprintf("ans_%s_%s", runtime.GOOS, arch)
	if runtime.GOOS == "windows" {
		asset += ".exe"
	}

	self, err := os.Executable()
	if err != nil {
		pretty.Err(w, "Cannot find current binary: "+err.Error())
		return nil
	}

	base := "https://github.com/" + repo + "/releases/latest/download"
	url := base + "/" + asset
	chkURL := base + "/checksums.txt"

	pretty.Step(w, 1, "Downloading "+asset)
	resp, err := http.Get(url)
	if err != nil {
		pretty.Err(w, "Download failed: "+err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		pretty.Err(w, fmt.Sprintf("Download failed: HTTP %d", resp.StatusCode))
		return nil
	}
	tmp, err := os.CreateTemp("", "ans-*"+path.Ext(asset))
	if err != nil {
		pretty.Err(w, "Creating temp file: "+err.Error())
		return nil
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	hasher := sha256.New()
	multi := io.MultiWriter(tmp, hasher)
	if _, err := io.Copy(multi, resp.Body); err != nil {
		tmp.Close()
		pretty.Err(w, "Download incomplete: "+err.Error())
		return nil
	}
	tmp.Close()
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	pretty.Done(w, "Downloaded")

	pretty.Step(w, 2, "Verifying checksum")
	chkResp, chkErr := http.Get(chkURL)
	if chkErr == nil && chkResp.StatusCode == 200 {
		defer chkResp.Body.Close()
		chkBody, _ := io.ReadAll(chkResp.Body)
		for _, line := range strings.Split(string(chkBody), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && (parts[1] == asset || strings.HasSuffix(parts[1], "/"+asset)) {
				if parts[0] != actualHash {
					pretty.Err(w, "Checksum mismatch")
					return nil
				}
				break
			}
		}
		pretty.Done(w, "Checksum verified")
	} else {
		pretty.Warn(w, "Checksum file not available -- skipped")
	}

	pretty.Step(w, 3, "Installing update")
	if err := os.Rename(tmpName, self); err == nil {
		pretty.Done(w, "Updated to the latest version")
		fmt.Fprintln(w)
		pretty.Step(w, 4, "Verify installation:")
		pretty.Code(w, "ans version")
		return nil
	}

	pretty.Warn(w, "Could not replace running binary directly")
	updated := self + ".new"
	copyOK := false
	if src, err := os.Open(tmpName); err == nil {
		defer src.Close()
		if dst, err := os.Create(updated); err == nil {
			defer dst.Close()
			if _, err := io.Copy(dst, src); err == nil {
				copyOK = true
			}
		}
	}

	if copyOK {
		pretty.Done(w, "Staged to "+path.Base(updated))
		fmt.Fprintln(w)
		pretty.Step(w, 4, "Complete the update in a new terminal:")
		if runtime.GOOS == "windows" {
			pretty.Code(w, fmt.Sprintf(`powershell -Command "Move-Item '%s' '%s' -Force"`, updated, self))
		} else {
			pretty.Code(w, fmt.Sprintf(`cp "%s" "%s" && rm "%s"`, updated, self, updated))
		}
		return nil
	}

	tmpUpdated := path.Join(os.TempDir(), path.Base(self)+".new")
	copyOK = false
	if src, err := os.Open(tmpName); err == nil {
		defer src.Close()
		if dst, err := os.Create(tmpUpdated); err == nil {
			defer dst.Close()
			if _, err := io.Copy(dst, src); err == nil {
				copyOK = true
			}
		}
	}

	if copyOK {
		pretty.Done(w, "Staged to "+tmpUpdated)
		fmt.Fprintln(w)
		pretty.Step(w, 4, "Complete the update manually:")
		if runtime.GOOS == "windows" {
			pretty.Code(w, fmt.Sprintf(`copy "%s" "%s"`, tmpUpdated, self))
		} else {
			pretty.Code(w, fmt.Sprintf(`cp "%s" "%s"`, tmpUpdated, self))
		}
		pretty.Item(w, "Tip", "Close all ANS processes first, then run the command above")
		return nil
	}

	pretty.Err(w, "Could not write update file -- permission denied")
	fmt.Fprintln(w)
	pretty.Step(w, 4, "Install manually from temp:")
	pretty.Code(w, fmt.Sprintf(`copy "%s" "%s"`, tmpName, self))
	pretty.Item(w, "Hint", "Run from an elevated (Admin) terminal if permissions are restricted")
	return nil
}

// --- uninstall ---

func cmdUninstall(w io.Writer, args []string) error {
	_ = args
	pretty.Banner(w)
	pretty.Header(w, "Uninstalling ANS")
	fmt.Fprintln(w)

	pretty.Step(w, 1, "Stopping daemon")
	if pidData, err := os.ReadFile(pidFilePath()); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(pidData))); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				if runtime.GOOS == "windows" {
					proc.Kill()
				} else {
					proc.Signal(syscall.SIGTERM)
				}
				os.Remove(pidFilePath())
			}
		}
	}
	pretty.Done(w, "Daemon stopped")

	home, _ := os.UserHomeDir()
	dotDir := path.Join(home, ".ans")

	pretty.Step(w, 2, "Removing data directory")
	if err := os.RemoveAll(dotDir); err == nil {
		pretty.Done(w, "Deleted "+dotDir)
	} else {
		pretty.Warn(w, "Could not delete "+dotDir+": "+err.Error())
		pretty.Item(w, "Hint", "Close any running ANS processes and try again")
	}

	if runtime.GOOS == "windows" {
		pretty.Step(w, 3, "Cleaning PATH")
		binDir := path.Join(dotDir, "bin")
		userPath := os.Getenv("Path")
		parts := strings.Split(userPath, ";")
		filtered := make([]string, 0, len(parts))
		for _, p := range parts {
			if strings.EqualFold(p, binDir) || strings.EqualFold(p, binDir+`\`) {
				continue
			}
			filtered = append(filtered, p)
		}
		newPath := strings.Join(filtered, ";")
		if newPath != userPath {
			os.Setenv("Path", newPath)
			pretty.Done(w, "Removed "+binDir+" from PATH")

			// Update user-level PATH with a timeout to avoid hanging
			psCmd := fmt.Sprintf(`[Environment]::SetEnvironmentVariable("Path", "%s", "User")`,
				strings.ReplaceAll(newPath, `"`, `""`))
			done := make(chan struct{}, 1)
			go func() {
				exec.Command("powershell", "-NoProfile", "-Command", psCmd).Run()
				done <- struct{}{}
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				pretty.Warn(w, "PowerShell PATH update timed out — set manually if needed")
			}
		} else {
			pretty.Done(w, "PATH already clean")
		}
	}

	fmt.Fprintln(w)
	pretty.Ok(w, "ANS has been uninstalled")
	pretty.Item(w, "Note", "Close and reopen your terminal to refresh PATH")
	fmt.Fprintln(w)
	pretty.Step(w, 4, "Reinstall anytime with")
	pretty.Code(w, `irm https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.ps1 | iex`)
	return nil
}



package hardware

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Device struct {
	Name           string   `json:"name"`
	Manufacturer   string   `json:"manufacturer,omitempty"`
	DeviceID       string   `json:"deviceId,omitempty"`
	Family         string   `json:"family"`
	State          string   `json:"state"`
	RelatedDevices []string `json:"relatedDevices,omitempty"`
}

type Runtime struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
}
type Status struct {
	Devices   []Device  `json:"devices"`
	Runtimes  []Runtime `json:"runtimes"`
	CheckedAt time.Time `json:"checkedAt"`
	Message   string    `json:"message"`
}

type Service struct {
	mu       sync.Mutex
	cached   Status
	cachedAt time.Time
}

func New() *Service { return &Service{} }

func (s *Service) Detect(ctx context.Context, refresh bool) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !refresh && !s.cachedAt.IsZero() && time.Since(s.cachedAt) < 30*time.Second {
		return s.cached
	}
	status := Status{Devices: []Device{}, Runtimes: detectRuntimes(), CheckedAt: time.Now()}
	command := `Get-CimInstance Win32_PnPEntity | Where-Object { $_.Status -eq 'OK' -and ($_.Name -match '(?i)Oculus|Meta Quest|Quest Link|Rift|Valve.*(VR|Index)|Index HMD|HTC.*VIVE|VIVE|PICO|Windows Mixed Reality|Mixed Reality|Varjo|Bigscreen Beyond|PS VR') } | Select-Object Name,Manufacturer,PNPDeviceID,Status | ConvertTo-Json -Compress`
	checkCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	output, err := hiddenCommandContext(checkCtx, "powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", command).Output()
	if err == nil && len(output) > 0 {
		status.Devices = parseDevices(output)
	}
	if len(status.Devices) > 0 {
		status.Message = "检测到当前连接的 VR 设备"
	} else {
		status.Message = "暂未从 Windows 设备列表识别到已连接头显"
	}
	s.cached, s.cachedAt = status, time.Now()
	return status
}

func parseDevices(data []byte) []Device {
	type raw struct {
		Name         string `json:"Name"`
		Manufacturer string `json:"Manufacturer"`
		PNPDeviceID  string `json:"PNPDeviceID"`
	}
	var list []raw
	if len(data) > 0 && data[0] == '[' {
		_ = json.Unmarshal(data, &list)
	} else {
		var one raw
		if json.Unmarshal(data, &one) == nil && one.Name != "" {
			list = []raw{one}
		}
	}
	groups := map[string]Device{}
	for _, item := range list {
		group := family(item.Name + " " + item.Manufacturer)
		current, exists := groups[group]
		if !exists || deviceNameScore(item.Name) > deviceNameScore(current.Name) {
			if exists {
				current.RelatedDevices = append(current.RelatedDevices, current.Name)
			}
			current.Name, current.Manufacturer, current.DeviceID, current.Family, current.State = item.Name, item.Manufacturer, item.PNPDeviceID, group, "connected"
		} else {
			current.RelatedDevices = append(current.RelatedDevices, item.Name)
		}
		groups[group] = current
	}
	result := make([]Device, 0, len(groups))
	for _, device := range groups {
		sort.Strings(device.RelatedDevices)
		result = append(result, device)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Family < result[j].Family })
	return result
}

func deviceNameScore(value string) int {
	value = strings.ToLower(value)
	score := 0
	if strings.Contains(value, "hmd") || strings.Contains(value, "headset") || strings.Contains(value, "quest") || strings.Contains(value, "rift") || strings.Contains(value, "pico 4") || strings.Contains(value, "index") {
		score += 10
	}
	if strings.Contains(value, "microphone") || strings.Contains(value, "麦克风") || strings.Contains(value, "speaker") || strings.Contains(value, "扬声器") || strings.Contains(value, "interface") {
		score -= 5
	}
	return score
}

func family(value string) string {
	value = strings.ToLower(value)
	for _, item := range []struct{ match, label string }{{"bigscreen", "Bigscreen Beyond"}, {"varjo", "Varjo"}, {"pico", "PICO"}, {"vive", "HTC VIVE"}, {"index", "Valve Index"}, {"valve", "Valve VR"}, {"quest", "Meta Quest"}, {"oculus", "Meta / Oculus"}, {"rift", "Oculus Rift"}, {"mixed reality", "Windows Mixed Reality"}, {"ps vr", "PlayStation VR"}} {
		if strings.Contains(value, item.match) {
			return item.label
		}
	}
	return "VR 设备"
}

func detectRuntimes() []Runtime {
	home, _ := os.UserHomeDir()
	local := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	items := []struct {
		name      string
		paths     []string
		processes []string
	}{
		{"SteamVR", []string{filepath.Join(programFiles, "Steam", "steamapps", "common", "SteamVR"), filepath.Join(programFilesX86, "Steam", "steamapps", "common", "SteamVR"), filepath.Join(home, "AppData", "Local", "openvr")}, []string{"vrserver.exe", "vrmonitor.exe"}},
		{"Meta Quest Link", []string{filepath.Join(programFiles, "Oculus"), filepath.Join(local, "Oculus")}, []string{"OVRServer_x64.exe", "OculusClient.exe"}},
		{"PICO Connect", []string{filepath.Join(programFiles, "PICO Connect"), filepath.Join(programFilesX86, "PICO Connect"), filepath.Join(programFiles, "Streaming Assistant")}, []string{"PICO Connect.exe", "Streaming Assistant.exe", "ps_service_launcher.exe"}},
		{"Virtual Desktop", []string{filepath.Join(programFiles, "Virtual Desktop Streamer")}, []string{"VirtualDesktop.Streamer.exe"}},
	}
	running := runningProcesses()
	result := make([]Runtime, 0, len(items))
	for _, item := range items {
		installed := false
		for _, path := range item.paths {
			if path != "" {
				if _, err := os.Stat(path); err == nil {
					installed = true
					break
				}
			}
		}
		active := false
		for _, name := range item.processes {
			if running[strings.ToLower(name)] {
				active = true
				installed = true
			}
		}
		result = append(result, Runtime{Name: item.name, Installed: installed, Running: active})
	}
	return result
}

func runningProcesses() map[string]bool {
	result := map[string]bool{}
	output, err := hiddenCommand("tasklist.exe", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return result
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 2 && line[0] == '"' {
			if end := strings.Index(line[1:], "\""); end >= 0 {
				result[strings.ToLower(line[1:1+end])] = true
			}
		}
	}
	return result
}

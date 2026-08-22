package hardware

import "testing"

func TestParseDevicesAndFamily(t *testing.T) {
	devices := parseDevices([]byte(`[{"Name":"Oculus Quest 3","Manufacturer":"Meta","PNPDeviceID":"USB\\A"},{"Name":"Valve Index HMD","Manufacturer":"Valve","PNPDeviceID":"USB\\B"}]`))
	if len(devices) != 2 || devices[0].Family == "" || devices[1].Family == "" {
		t.Fatalf("devices=%#v", devices)
	}
	pico := parseDevices([]byte(`[{"Name":"PICO 4","Manufacturer":"Pico","PNPDeviceID":"USB\\A"},{"Name":"Pico Streaming Speaker","Manufacturer":"Pico","PNPDeviceID":"USB\\B"}]`))
	if len(pico) != 1 || pico[0].Name != "PICO 4" || len(pico[0].RelatedDevices) != 1 {
		t.Fatalf("pico=%#v", pico)
	}
}

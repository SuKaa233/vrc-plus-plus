package vrchat

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/storage"
)

func TestBuildTransportValidatesModesAndLocalProxy(t *testing.T) {
	for _, config := range []model.NetworkConfig{
		{Mode: "system"}, {Mode: "direct"},
		{Mode: "http", ProxyURL: "http://127.0.0.1:7890"},
		{Mode: "socks5", ProxyURL: "socks5://localhost:7891"},
	} {
		if _, err := buildTransport(config); err != nil {
			t.Fatalf("buildTransport(%#v): %v", config, err)
		}
	}
	for _, config := range []model.NetworkConfig{
		{Mode: "unknown"},
		{Mode: "http", ProxyURL: "http://proxy.example:7890"},
		{Mode: "http", ProxyURL: "http://user:password@127.0.0.1:7890"},
		{Mode: "socks5", ProxyURL: "socks5://127.0.0.1"},
	} {
		if _, err := buildTransport(config); err == nil {
			t.Fatalf("buildTransport(%#v) unexpectedly succeeded", config)
		}
	}
}

func TestNetworkConfigRoundTrip(t *testing.T) {
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "harbor.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	client, err := NewClient("https://api.vrchat.cloud/api/1", "Test/1 contact@example.com", store, testProtector{})
	if err != nil {
		t.Fatal(err)
	}
	config := model.NetworkConfig{Mode: "http", ProxyURL: "http://127.0.0.1:7890"}
	if _, err := client.ApplyNetworkConfig(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadNetworkConfig(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != config {
		t.Fatalf("LoadNetworkConfig() = %#v, want %#v", loaded, config)
	}
}

package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateMediaURL(t *testing.T) {
	for _, value := range []string{
		"https://api.vrchat.cloud/api/1/file/file_test/1/file",
		"https://assets.vrcdn.cloud/image.png",
	} {
		if _, err := validateMediaURL(value); err != nil {
			t.Fatalf("validateMediaURL(%q): %v", value, err)
		}
	}
	for _, value := range []string{
		"http://api.vrchat.cloud/image.png",
		"https://vrchat.cloud.evil.example/image.png",
		"https://127.0.0.1/image.png",
		"https://user:password@api.vrchat.cloud/image.png",
	} {
		if _, err := validateMediaURL(value); err == nil {
			t.Fatalf("validateMediaURL(%q) unexpectedly succeeded", value)
		}
	}
}

func TestStatsAndClearOnlyMediaArtifacts(t *testing.T) {
	directory := t.TempDir()
	for name, payload := range map[string]string{
		"avatar.bin":  "12345",
		"avatar.type": "image/png",
		"partial.tmp": "pending",
		"keep.txt":    "do not remove",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	service := &Service{directory: directory}
	files, bytes, err := service.Stats()
	if err != nil || files != 1 || bytes != 5 {
		t.Fatalf("Stats() = %d, %d, %v", files, bytes, err)
	}
	files, bytes, err = service.Clear()
	if err != nil || files != 1 || bytes != 5 {
		t.Fatalf("Clear() = %d, %d, %v", files, bytes, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "keep.txt")); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
	for _, name := range []string{"avatar.bin", "avatar.type", "partial.tmp"} {
		if _, err := os.Stat(filepath.Join(directory, name)); !os.IsNotExist(err) {
			t.Fatalf("%s still exists after Clear(): %v", name, err)
		}
	}
}

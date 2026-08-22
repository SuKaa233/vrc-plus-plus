package photos

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestListDetailRotateRenameAndDelete(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "2026-08")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(nested, "VRChat_2026-08-22_12-30-00.000_4x2.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	source := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	source.Set(0, 0, color.NRGBA{R: 255, A: 255})
	if err = png.Encode(file, source); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	service := &Service{root: root, cache: filepath.Join(root, "cache"), thumbSlots: make(chan struct{}, 2)}
	if err := os.MkdirAll(service.cache, 0o700); err != nil {
		t.Fatal(err)
	}
	listing, err := service.List("VRChat", 0, 10)
	if err != nil || listing.Total != 1 {
		t.Fatalf("listing=%#v err=%v", listing, err)
	}
	item := listing.Items[0]
	detail, err := service.Detail(item.ID)
	if err != nil || detail.Width != 4 || detail.Height != 2 || detail.CapturedAt == nil || detail.SHA256 == "" {
		t.Fatalf("detail=%#v err=%v", detail, err)
	}
	thumb, thumbInfo, err := service.Thumbnail(item.ID)
	if err != nil || thumbInfo.Size() == 0 {
		t.Fatalf("thumbnail=%#v err=%v", thumbInfo, err)
	}
	thumb.Close()
	rotated, err := service.Rotate(item.ID, true)
	if err != nil || rotated.Width != 2 || rotated.Height != 4 {
		t.Fatalf("rotated=%#v err=%v", rotated, err)
	}
	renamed, err := service.Rename(item.ID, "summer.png")
	if err != nil || renamed.Name != "summer.png" {
		t.Fatalf("renamed=%#v err=%v", renamed, err)
	}
	result, err := service.Delete(renamed.ID)
	if err != nil || !result.Deleted {
		t.Fatalf("deleted=%#v err=%v", result, err)
	}
	if _, err = service.Detail(renamed.ID); !os.IsNotExist(err) {
		t.Fatalf("deleted photo still available: %v", err)
	}
}

func TestResolveRejectsTraversal(t *testing.T) {
	service := &Service{root: t.TempDir(), thumbSlots: make(chan struct{}, 2)}
	id := base64.RawURLEncoding.EncodeToString([]byte(`..\secret.png`))
	if _, _, err := service.resolve(id); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

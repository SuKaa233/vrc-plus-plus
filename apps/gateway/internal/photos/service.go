package photos

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxDetailBytes = 64 << 20

type Service struct {
	root       string
	cache      string
	thumbSlots chan struct{}
}

type Photo struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	RelativePath string     `json:"relativePath"`
	Extension    string     `json:"extension"`
	ContentType  string     `json:"contentType"`
	Size         int64      `json:"size"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	ModifiedAt   time.Time  `json:"modifiedAt"`
	CapturedAt   *time.Time `json:"capturedAt,omitempty"`
}

type Detail struct {
	Photo
	AbsolutePath string            `json:"absolutePath"`
	AspectRatio  string            `json:"aspectRatio"`
	Megapixels   float64           `json:"megapixels"`
	SHA256       string            `json:"sha256,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type Listing struct {
	Directory string  `json:"directory"`
	Items     []Photo `json:"items"`
	Total     int     `json:"total"`
	Offset    int     `json:"offset"`
	Limit     int     `json:"limit"`
	Message   string  `json:"message,omitempty"`
}

type DeleteResult struct {
	Deleted bool   `json:"deleted"`
	Message string `json:"message"`
}

func New(cacheDirectory string) (*Service, error) {
	if err := os.MkdirAll(cacheDirectory, 0o700); err != nil {
		return nil, err
	}
	return &Service{root: defaultDirectory(), cache: cacheDirectory, thumbSlots: make(chan struct{}, 2)}, nil
}
func (s *Service) Directory() string { return s.root }

func defaultDirectory() string {
	home, _ := os.UserHomeDir()
	candidates := []string{filepath.Join(home, "Pictures", "VRChat")}
	if oneDrive := strings.TrimSpace(os.Getenv("OneDrive")); oneDrive != "" {
		candidates = append(candidates, filepath.Join(oneDrive, "Pictures", "VRChat"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return candidates[0]
}

func (s *Service) List(query string, offset, limit int) (Listing, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 120
	}
	if limit > 500 {
		limit = 500
	}
	result := Listing{Directory: s.root, Offset: offset, Limit: limit, Items: []Photo{}}
	if info, err := os.Stat(s.root); err != nil || !info.IsDir() {
		result.Message = "还没有找到 VRChat 相片目录；拍摄第一张照片后会自动出现。"
		return result, nil
	}
	needle := strings.ToLower(strings.TrimSpace(query))
	items := make([]Photo, 0, 256)
	err := filepath.WalkDir(s.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if path != s.root && strings.HasPrefix(entry.Name(), ".vrcpp-") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			return nil
		}
		relative, err := filepath.Rel(s.root, path)
		if err != nil {
			return nil
		}
		if needle != "" && !strings.Contains(strings.ToLower(relative), needle) {
			return nil
		}
		photo, err := inspect(path, relative, false)
		if err == nil {
			items = append(items, photo.Photo)
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ModifiedAt.After(items[j].ModifiedAt) })
	result.Total = len(items)
	if offset >= len(items) {
		return result, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	result.Items = items[offset:end]
	return result, nil
}

func (s *Service) Detail(id string) (Detail, error) {
	path, relative, err := s.resolve(id)
	if err != nil {
		return Detail{}, err
	}
	return inspect(path, relative, true)
}

func (s *Service) Open(id string) (*os.File, fs.FileInfo, error) {
	path, _, err := s.resolve(id)
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	return file, info, nil
}

func (s *Service) Thumbnail(id string) (*os.File, fs.FileInfo, error) {
	path, _, err := s.resolve(id)
	if err != nil {
		return nil, nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	key := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", path, info.Size(), info.ModTime().UnixNano())))
	target := filepath.Join(s.cache, hex.EncodeToString(key[:])+".jpg")
	openCached := func() (*os.File, fs.FileInfo, error) {
		file, err := os.Open(target)
		if err != nil {
			return nil, nil, err
		}
		cached, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, nil, err
		}
		return file, cached, nil
	}
	if file, cached, openErr := openCached(); openErr == nil {
		return file, cached, nil
	}
	s.thumbSlots <- struct{}{}
	defer func() { <-s.thumbSlots }()
	if file, cached, openErr := openCached(); openErr == nil {
		return file, cached, nil
	}
	sourceFile, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	source, _, err := image.Decode(io.LimitReader(sourceFile, maxDetailBytes+1))
	sourceFile.Close()
	if err != nil {
		return nil, nil, err
	}
	bounds := source.Bounds()
	width := 480
	if bounds.Dx() < width {
		width = bounds.Dx()
	}
	height := bounds.Dy() * width / bounds.Dx()
	if height < 1 {
		height = 1
	}
	thumbnail := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		sourceY := bounds.Min.Y + y*bounds.Dy()/height
		for x := 0; x < width; x++ {
			sourceX := bounds.Min.X + x*bounds.Dx()/width
			thumbnail.Set(x, y, source.At(sourceX, sourceY))
		}
	}
	temporary, err := os.CreateTemp(s.cache, "thumb-*.tmp")
	if err != nil {
		return nil, nil, err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err = jpeg.Encode(temporary, thumbnail, &jpeg.Options{Quality: 82}); err != nil {
		temporary.Close()
		return nil, nil, err
	}
	if err = temporary.Close(); err != nil {
		return nil, nil, err
	}
	if err = os.Rename(temporaryPath, target); err != nil {
		return nil, nil, err
	}
	return openCached()
}

func (s *Service) Rotate(id string, clockwise bool) (Detail, error) {
	path, relative, err := s.resolve(id)
	if err != nil {
		return Detail{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return Detail{}, err
	}
	source, format, err := image.Decode(io.LimitReader(file, maxDetailBytes+1))
	file.Close()
	if err != nil || (format != "png" && format != "jpeg") {
		return Detail{}, fmt.Errorf("这张图片暂不支持旋转")
	}
	bounds := source.Bounds()
	rotated := image.NewNRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if clockwise {
				rotated.Set(bounds.Max.Y-1-y, x-bounds.Min.X, source.At(x, y))
			} else {
				rotated.Set(y-bounds.Min.Y, bounds.Max.X-1-x, source.At(x, y))
			}
		}
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".vrcpp-edit-*")
	if err != nil {
		return Detail{}, err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if format == "png" {
		err = png.Encode(temporary, rotated)
	} else {
		err = jpeg.Encode(temporary, rotated, &jpeg.Options{Quality: 95})
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return Detail{}, err
	}
	backupDir := filepath.Join(s.root, ".vrcpp-backups")
	if err = os.MkdirAll(backupDir, 0o700); err != nil {
		return Detail{}, err
	}
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405.000"), filepath.Base(path)))
	if err = os.Rename(path, backupPath); err != nil {
		return Detail{}, err
	}
	if err = os.Rename(temporaryPath, path); err != nil {
		_ = os.Rename(backupPath, path)
		return Detail{}, err
	}
	return inspect(path, relative, true)
}

func (s *Service) Rename(id, name string) (Detail, error) {
	path, _, err := s.resolve(id)
	if err != nil {
		return Detail{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `<>:"/\|?*`) {
		return Detail{}, fmt.Errorf("文件名包含不允许的字符")
	}
	originalExt := strings.ToLower(filepath.Ext(path))
	if filepath.Ext(name) == "" {
		name += originalExt
	}
	if strings.ToLower(filepath.Ext(name)) != originalExt {
		return Detail{}, fmt.Errorf("重命名时不能改变图片格式")
	}
	target := filepath.Join(filepath.Dir(path), name)
	if _, err := os.Stat(target); err == nil {
		return Detail{}, fmt.Errorf("同名文件已经存在")
	}
	if err := os.Rename(path, target); err != nil {
		return Detail{}, err
	}
	relative, _ := filepath.Rel(s.root, target)
	return inspect(target, relative, true)
}

func (s *Service) Delete(id string) (DeleteResult, error) {
	path, _, err := s.resolve(id)
	if err != nil {
		return DeleteResult{}, err
	}
	trash := filepath.Join(s.root, ".vrcpp-trash")
	if err := os.MkdirAll(trash, 0o700); err != nil {
		return DeleteResult{}, err
	}
	name := fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405.000"), filepath.Base(path))
	if err := os.Rename(path, filepath.Join(trash, name)); err != nil {
		return DeleteResult{}, err
	}
	return DeleteResult{Deleted: true, Message: "照片已移入 VRC++ 回收区，可在相片目录中找回。"}, nil
}

func (s *Service) Reveal(id string) error {
	path, _, err := s.resolve(id)
	if err != nil {
		return err
	}
	return exec.Command("explorer.exe", "/select,", path).Start()
}

func (s *Service) resolve(id string) (string, string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return "", "", fmt.Errorf("图片标识无效")
	}
	relative := filepath.Clean(string(decoded))
	if relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("图片路径无效")
	}
	path := filepath.Join(s.root, relative)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", "", os.ErrNotExist
	}
	return path, relative, nil
}

func inspect(path, relative string, detailed bool) (Detail, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Detail{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return Detail{}, err
	}
	config, format, err := image.DecodeConfig(bufio.NewReader(io.LimitReader(file, 8<<20)))
	file.Close()
	if err != nil {
		return Detail{}, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "image/" + format
	}
	photo := Photo{ID: base64.RawURLEncoding.EncodeToString([]byte(relative)), Name: filepath.Base(path), RelativePath: relative, Extension: strings.TrimPrefix(ext, "."), ContentType: contentType, Size: info.Size(), Width: config.Width, Height: config.Height, ModifiedAt: info.ModTime()}
	photo.CapturedAt = captureTime(photo.Name)
	detail := Detail{Photo: photo, AbsolutePath: path, AspectRatio: aspect(config.Width, config.Height), Megapixels: float64(config.Width*config.Height) / 1_000_000}
	if !detailed {
		return detail, nil
	}
	if info.Size() <= maxDetailBytes {
		file, err = os.Open(path)
		if err != nil {
			return Detail{}, err
		}
		defer file.Close()
		hash := sha256.New()
		if _, err = io.Copy(hash, file); err == nil {
			detail.SHA256 = hex.EncodeToString(hash.Sum(nil))
		}
	}
	if ext == ".png" {
		detail.Metadata = pngMetadata(path)
	}
	return detail, nil
}

func captureTime(name string) *time.Time {
	stem := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	formats := []string{"VRChat_2006-01-02_15-04-05.000_3840x2160", "VRChat_2006-01-02_15-04-05.000", "2006-01-02_15-04-05"}
	for _, format := range formats {
		if len(stem) >= len(format) {
			if value, err := time.ParseInLocation(format, stem[:len(format)], time.Local); err == nil {
				return &value
			}
		}
	}
	return nil
}

func aspect(width, height int) string {
	if width <= 0 || height <= 0 {
		return "—"
	}
	divisor := gcd(width, height)
	return fmt.Sprintf("%d:%d", width/divisor, height/divisor)
}
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func pngMetadata(path string) map[string]string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	signature := make([]byte, 8)
	if _, err = io.ReadFull(file, signature); err != nil || !bytes.Equal(signature, []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return nil
	}
	result := map[string]string{}
	for count := 0; count < 64; count++ {
		var length uint32
		if err = binary.Read(file, binary.BigEndian, &length); err != nil {
			return result
		}
		kind := make([]byte, 4)
		if _, err = io.ReadFull(file, kind); err != nil {
			return result
		}
		if length > 1<<20 {
			return result
		}
		data := make([]byte, length)
		if _, err = io.ReadFull(file, data); err != nil {
			return result
		}
		if _, err = io.CopyN(io.Discard, file, 4); err != nil {
			return result
		}
		if string(kind) == "tEXt" {
			parts := bytes.SplitN(data, []byte{0}, 2)
			if len(parts) == 2 && len(result) < 24 {
				result[string(parts[0])] = string(parts[1])
			}
		}
		if string(kind) == "IEND" {
			break
		}
	}
	return result
}

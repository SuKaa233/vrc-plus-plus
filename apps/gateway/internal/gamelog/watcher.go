package gamelog

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/events"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
)

var userSuffix = regexp.MustCompile(`^(.*?)\s*\((usr_[^)]+)\)\s*$`)

type Watcher struct {
	directory string
	bus       *events.Bus
	logger    *slog.Logger
	mu        sync.RWMutex
	status    model.GameLogStatus
	cancel    context.CancelFunc
}

func New(directory string, bus *events.Bus, logger *slog.Logger) *Watcher {
	return &Watcher{directory: directory, bus: bus, logger: logger, status: model.GameLogStatus{State: "starting", Directory: directory}}
}

func DefaultDirectory() string {
	if configured := strings.TrimSpace(os.Getenv("VRC_HARBOR_GAME_LOG_DIR")); configured != "" {
		return configured
	}
	base := os.Getenv("USERPROFILE")
	if base == "" {
		base, _ = os.UserHomeDir()
	}
	return filepath.Join(base, "AppData", "LocalLow", "VRChat", "VRChat")
}

func (w *Watcher) Start() {
	w.mu.Lock()
	if w.cancel != nil {
		w.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.mu.Unlock()
	go w.run(ctx)
}

func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}
func (w *Watcher) Status() model.GameLogStatus { w.mu.RLock(); defer w.mu.RUnlock(); return w.status }

func (w *Watcher) setStatus(update func(*model.GameLogStatus)) {
	w.mu.Lock()
	update(&w.status)
	w.mu.Unlock()
}

func (w *Watcher) run(ctx context.Context) {
	var current string
	var offset int64
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()
	for {
		path, err := newestLog(w.directory)
		if err != nil {
			w.setStatus(func(s *model.GameLogStatus) {
				s.State = "waiting"
				s.Message = "等待 VRChat 创建 output_log 日志"
			})
		} else {
			if path != current {
				current, offset = path, 0
			}
			next, count, readErr := w.readFrom(ctx, path, offset)
			if readErr != nil && !errors.Is(readErr, context.Canceled) {
				w.setStatus(func(s *model.GameLogStatus) { s.State = "error"; s.Message = readErr.Error() })
			} else {
				offset = next
				now := time.Now().UTC()
				w.setStatus(func(s *model.GameLogStatus) {
					s.State = "watching"
					s.File = filepath.Base(path)
					s.LastReadAt = &now
					s.Events += count
					s.Message = "只保存结构化事件，不保存原始日志正文"
				})
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func newestLog(directory string) (string, error) {
	entries, err := filepath.Glob(filepath.Join(directory, "output_log_*.txt"))
	if err != nil || len(entries) == 0 {
		return "", os.ErrNotExist
	}
	sort.Slice(entries, func(i, j int) bool {
		ai, _ := os.Stat(entries[i])
		aj, _ := os.Stat(entries[j])
		return ai.ModTime().After(aj.ModTime())
	})
	return entries[0], nil
}

func (w *Watcher) readFrom(ctx context.Context, path string, offset int64) (int64, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return offset, 0, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return offset, 0, err
	}
	if stat.Size() < offset {
		offset = 0
	}
	if offset == 0 && stat.Size() > 4<<20 {
		offset = stat.Size() - (4 << 20)
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return offset, 0, err
	}
	reader := bufio.NewReaderSize(file, 64<<10)
	if offset > 0 {
		_, _ = reader.ReadString('\n')
	}
	count := 0
	for {
		select {
		case <-ctx.Done():
			return offset, count, ctx.Err()
		default:
		}
		before, _ := file.Seek(0, io.SeekCurrent)
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return offset, count, err
		}
		if line == "" {
			break
		}
		after, _ := file.Seek(0, io.SeekCurrent)
		if after > before {
			offset = after - int64(reader.Buffered())
		}
		if event, ok := ParseLine(filepath.Base(path), offset, line); ok {
			w.bus.Publish(event)
			count++
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return offset, count, nil
}

func ParseLine(file string, offset int64, line string) (model.DomainEvent, bool) {
	line = strings.TrimSpace(line)
	timestamp := parseTime(line)
	typeName, displayName, userID, location, worldName, sensitiveLocation := "", "", "", "", "", ""
	switch {
	case strings.Contains(line, "[Behaviour] OnPlayerJoined") && !strings.Contains(line, "] OnPlayerJoined:"):
		typeName = "game.player-joined"
		displayName, userID = parseUser(after(line, "] OnPlayerJoined"))
	case strings.Contains(line, "[Behaviour] OnPlayerLeft") && !strings.Contains(line, "OnPlayerLeftRoom") && !strings.Contains(line, "] OnPlayerLeft:"):
		typeName = "game.player-left"
		displayName, userID = parseUser(after(line, "] OnPlayerLeft"))
	case strings.Contains(line, "[Behaviour] Entering Room: "):
		typeName = "game.world-entering"
		worldName = after(line, "] Entering Room: ")
	case strings.Contains(line, "[Behaviour] Joining ") && !strings.Contains(line, "Joining or Creating Room") && !strings.Contains(line, "Joining friend"):
		typeName = "game.location"
		sensitiveLocation = strings.TrimSpace(after(line, "] Joining "))
		location = sanitizeLocation(sensitiveLocation)
	case strings.Contains(line, "VRCApplication: HandleApplicationQuit"):
		typeName = "game.quit-clean"
	default:
		return model.DomainEvent{}, false
	}
	if typeName == "" || (strings.Contains(typeName, "player-") && displayName == "" && userID == "") {
		return model.DomainEvent{}, false
	}
	content, _ := json.Marshal(map[string]string{"displayName": displayName, "userId": userID, "location": location, "worldName": worldName})
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s", file, offset, typeName)))
	return model.DomainEvent{ID: "log_" + hex.EncodeToString(digest[:10]), Type: typeName, ObservedAt: timestamp, Content: content, SensitiveLocation: sensitiveLocation}, true
}

func after(value, marker string) string {
	if index := strings.LastIndex(value, marker); index >= 0 {
		return strings.TrimSpace(value[index+len(marker):])
	}
	return ""
}
func parseUser(value string) (string, string) {
	value = strings.TrimSpace(value)
	if match := userSuffix.FindStringSubmatch(value); len(match) == 3 {
		return strings.TrimSpace(match[1]), match[2]
	}
	return value, ""
}
func sanitizeLocation(value string) string {
	if index := strings.Index(value, "~nonce("); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}
func parseTime(line string) time.Time {
	if len(line) >= 19 {
		if value, err := time.ParseInLocation("2006.01.02 15:04:05", line[:19], time.Local); err == nil {
			return value.UTC()
		}
	}
	return time.Now().UTC()
}

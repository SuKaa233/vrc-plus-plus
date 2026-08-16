package diagnostics

import (
	"context"
	"sync"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/storage"
	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/vrchat"
)

type Service struct {
	store  *storage.Store
	client *vrchat.Client
	mu     sync.Mutex
	last   model.Diagnostics
	at     time.Time
}

func New(store *storage.Store, client *vrchat.Client) *Service {
	return &Service{store: store, client: client}
}

func (s *Service) Run(ctx context.Context, refresh bool) model.Diagnostics {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !refresh && !s.at.IsZero() && time.Since(s.at) < 30*time.Second {
		return s.last
	}
	diagnostics := model.Diagnostics{Overall: model.CheckOK}
	diagnostics.Database.Path = s.store.Path()
	diagnostics.Database.Ready = s.store.Ready(ctx)
	diagnostics.VRChat.BaseURL = s.client.BaseURL()
	diagnostics.VRChat.UserAgent = s.client.UserAgent()
	for _, probe := range []struct{ name, endpoint string }{
		{"VRChat API 配置", "config"},
		{"VRChat API 健康", "health"},
	} {
		result := s.client.Probe(ctx, probe.name, probe.endpoint)
		diagnostics.Checks = append(diagnostics.Checks, result)
		if result.State == model.CheckError {
			diagnostics.Overall = model.CheckError
		} else if result.State == model.CheckDegraded && diagnostics.Overall == model.CheckOK {
			diagnostics.Overall = model.CheckDegraded
		}
	}
	if !diagnostics.Database.Ready {
		diagnostics.Overall = model.CheckError
	}
	s.last, s.at = diagnostics, time.Now()
	return diagnostics
}

func (s *Service) Invalidate() {
	s.mu.Lock()
	s.at = time.Time{}
	s.mu.Unlock()
}

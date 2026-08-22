package localapi

import (
	"net/http"
	"time"
)

func (s *Server) getGuardianStatus(writer http.ResponseWriter, _ *http.Request) {
	if s.guardian == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_GUARDIAN_UNAVAILABLE", "VRChat 守护服务未启用", true)
		return
	}
	writeJSON(writer, http.StatusOK, s.guardian.Status())
}

func (s *Server) resumeGuardianSession(writer http.ResponseWriter, _ *http.Request) {
	if s.guardian == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_GUARDIAN_UNAVAILABLE", "VRChat 守护服务未启用", true)
		return
	}
	target, err := s.guardian.Resume()
	if err != nil {
		writeAPIError(writer, http.StatusConflict, "LOCAL_GUARDIAN_RESUME_FAILED", err.Error(), true)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"launched": true, "target": target})
}

func (s *Server) dismissGuardianRecovery(writer http.ResponseWriter, _ *http.Request) {
	if s.guardian == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_GUARDIAN_UNAVAILABLE", "VRChat 守护服务未启用", true)
		return
	}
	if err := s.guardian.Dismiss(); err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, s.guardian.Status())
}

func (s *Server) launchGuardianLocation(writer http.ResponseWriter, request *http.Request) {
	if s.guardian == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_GUARDIAN_UNAVAILABLE", "VRChat 守护服务未启用", true)
		return
	}
	var input struct {
		Location string `json:"location"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, err)
		return
	}
	target, err := s.guardian.LaunchLocation(input.Location)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_GUARDIAN_LAUNCH_FAILED", err.Error(), false)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"launched": true, "target": target})
}

func (s *Server) startGuardianSlotWatch(writer http.ResponseWriter, request *http.Request) {
	if s.guardian == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_GUARDIAN_UNAVAILABLE", "VRChat 守护服务未启用", true)
		return
	}
	var input struct {
		Location        string `json:"location"`
		WorldName       string `json:"worldName"`
		DurationMinutes int    `json:"durationMinutes"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, err)
		return
	}
	if input.DurationMinutes == 0 {
		input.DurationMinutes = 120
	}
	if err := s.guardian.StartSlotWatch(input.Location, input.WorldName, time.Duration(input.DurationMinutes)*time.Minute); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_GUARDIAN_SLOT_WATCH_INVALID", err.Error(), false)
		return
	}
	writeJSON(writer, http.StatusOK, s.guardian.Status())
}

func (s *Server) stopGuardianSlotWatch(writer http.ResponseWriter, _ *http.Request) {
	if s.guardian == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_GUARDIAN_UNAVAILABLE", "VRChat 守护服务未启用", true)
		return
	}
	if err := s.guardian.StopSlotWatch(); err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, s.guardian.Status())
}

func (s *Server) startGuardianMigrationWatch(writer http.ResponseWriter, request *http.Request) {
	if s.guardian == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_GUARDIAN_UNAVAILABLE", "VRChat 守护服务未启用", true)
		return
	}
	var input struct {
		DurationMinutes int `json:"durationMinutes"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, err)
		return
	}
	if input.DurationMinutes == 0 {
		input.DurationMinutes = 30
	}
	if err := s.guardian.StartMigrationWatch(time.Duration(input.DurationMinutes) * time.Minute); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_GUARDIAN_MIGRATION_WATCH_INVALID", err.Error(), false)
		return
	}
	writeJSON(writer, http.StatusOK, s.guardian.Status())
}

func (s *Server) stopGuardianMigrationWatch(writer http.ResponseWriter, _ *http.Request) {
	if s.guardian == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_GUARDIAN_UNAVAILABLE", "VRChat 守护服务未启用", true)
		return
	}
	if err := s.guardian.StopMigrationWatch(); err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, s.guardian.Status())
}

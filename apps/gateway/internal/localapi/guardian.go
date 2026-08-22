package localapi

import (
	"net/http"
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

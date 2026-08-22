package localapi

import (
	"mime"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) listPhotos(writer http.ResponseWriter, request *http.Request) {
	if s.photos == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_PHOTOS_UNAVAILABLE", "相片服务未启用", true)
		return
	}
	offset, _ := strconv.Atoi(request.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(request.URL.Query().Get("limit"))
	result, err := s.photos.List(request.URL.Query().Get("q"), offset, limit)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) getPhotoDetail(writer http.ResponseWriter, request *http.Request) {
	result, err := s.photos.Detail(request.PathValue("photoID"))
	if err != nil {
		writeAPIError(writer, http.StatusNotFound, "LOCAL_PHOTO_NOT_FOUND", "相片不存在或已移动", false)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}
func (s *Server) servePhoto(writer http.ResponseWriter, request *http.Request) {
	file, info, err := s.photos.Open(request.PathValue("photoID"))
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	defer file.Close()
	contentType := mime.TypeByExtension(strings.ToLower(filepathExtension(info.Name())))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	writer.Header().Set("Content-Type", contentType)
	writer.Header().Set("Cache-Control", "private, max-age=300")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(writer, request, info.Name(), info.ModTime(), file)
}

func (s *Server) servePhotoThumbnail(writer http.ResponseWriter, request *http.Request) {
	file, info, err := s.photos.Thumbnail(request.PathValue("photoID"))
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	defer file.Close()
	writer.Header().Set("Content-Type", "image/jpeg")
	writer.Header().Set("Cache-Control", "private, max-age=86400")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(writer, request, info.Name(), info.ModTime(), file)
}

func (s *Server) rotatePhoto(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Direction string `json:"direction"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, err)
		return
	}
	result, err := s.photos.Rotate(request.PathValue("photoID"), input.Direction != "left")
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_PHOTO_EDIT_FAILED", err.Error(), false)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}
func (s *Server) renamePhoto(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, err)
		return
	}
	result, err := s.photos.Rename(request.PathValue("photoID"), input.Name)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_PHOTO_RENAME_FAILED", err.Error(), false)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}
func (s *Server) revealPhoto(writer http.ResponseWriter, request *http.Request) {
	if err := s.photos.Reveal(request.PathValue("photoID")); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_PHOTO_REVEAL_FAILED", "无法打开相片所在位置", true)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]bool{"opened": true})
}
func (s *Server) deletePhoto(writer http.ResponseWriter, request *http.Request) {
	result, err := s.photos.Delete(request.PathValue("photoID"))
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "LOCAL_PHOTO_DELETE_FAILED", "无法删除这张相片", true)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}
func (s *Server) getVRHardware(writer http.ResponseWriter, request *http.Request) {
	if s.hardware == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "LOCAL_HARDWARE_UNAVAILABLE", "设备检测未启用", true)
		return
	}
	writeJSON(writer, http.StatusOK, s.hardware.Detect(request.Context(), request.URL.Query().Get("refresh") == "1"))
}

func filepathExtension(name string) string {
	if index := strings.LastIndex(name, "."); index >= 0 {
		return name[index:]
	}
	return ""
}

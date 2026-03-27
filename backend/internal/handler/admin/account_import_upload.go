package admin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	accountImportUploadChunkSizeDefault = int64(4 * 1024 * 1024)
	accountImportUploadSessionTTL       = 2 * time.Hour
)

type accountImportUploadStatus string

const (
	accountImportUploadStatusPending   accountImportUploadStatus = "pending"
	accountImportUploadStatusUploading accountImportUploadStatus = "uploading"
	accountImportUploadStatusUploaded  accountImportUploadStatus = "uploaded"
	accountImportUploadStatusFinalized accountImportUploadStatus = "finalized"
)

type createAccountImportUploadSessionRequest struct {
	Filename             string  `json:"filename" binding:"required"`
	TotalBytes           int64   `json:"total_bytes" binding:"required"`
	GroupIDs             []int64 `json:"group_ids"`
	SkipDefaultGroupBind *bool   `json:"skip_default_group_bind"`
}

type accountImportUploadSessionState struct {
	SessionID            string                    `json:"session_id"`
	Filename             string                    `json:"filename"`
	TotalBytes           int64                     `json:"total_bytes"`
	ReceivedBytes        int64                     `json:"received_bytes"`
	ChunkSize            int64                     `json:"chunk_size"`
	Status               accountImportUploadStatus `json:"status"`
	GroupIDs             []int64                   `json:"group_ids,omitempty"`
	SkipDefaultGroupBind *bool                     `json:"skip_default_group_bind,omitempty"`
	TaskID               string                    `json:"task_id,omitempty"`
	CreatedAt            time.Time                 `json:"created_at"`
	UpdatedAt            time.Time                 `json:"updated_at"`
	ExpiresAt            time.Time                 `json:"expires_at"`
}

type accountImportUploadSession struct {
	state    accountImportUploadSessionState
	filepath string
}

type accountImportUploadSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*accountImportUploadSession
}

var defaultAccountImportUploadSessions = &accountImportUploadSessionManager{
	sessions: make(map[string]*accountImportUploadSession),
}

func defaultAccountImportUploadSessionManager() *accountImportUploadSessionManager {
	return defaultAccountImportUploadSessions
}

func accountImportUploadSessionCacheKey(sessionID string) string {
	return "upload:session:account_import:" + sessionID
}

func (m *accountImportUploadSessionManager) createSession(req createAccountImportUploadSessionRequest) (*accountImportUploadSessionState, string, error) {
	if m == nil {
		return nil, "", errors.New("upload session manager is not ready")
	}
	totalBytes := req.TotalBytes
	if totalBytes <= 0 {
		return nil, "", errors.New("total_bytes must be positive")
	}
	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		return nil, "", errors.New("filename is required")
	}
	tmp, err := os.CreateTemp("", "sub2api-account-upload-*.json")
	if err != nil {
		return nil, "", err
	}
	if err := tmp.Truncate(totalBytes); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, "", err
	}
	_ = tmp.Close()

	now := time.Now().UTC()
	sessionID := uuid.NewString()
	session := &accountImportUploadSession{
		state: accountImportUploadSessionState{
			SessionID:            sessionID,
			Filename:             filename,
			TotalBytes:           totalBytes,
			ReceivedBytes:        0,
			ChunkSize:            accountImportUploadChunkSizeDefault,
			Status:               accountImportUploadStatusPending,
			GroupIDs:             normalizeInt64IDList(req.GroupIDs),
			SkipDefaultGroupBind: req.SkipDefaultGroupBind,
			CreatedAt:            now,
			UpdatedAt:            now,
			ExpiresAt:            now.Add(accountImportUploadSessionTTL),
		},
		filepath: tmp.Name(),
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()
	storeTaskStateJSON(context.Background(), accountImportUploadSessionCacheKey(sessionID), accountImportUploadSessionTTL, session.state)

	state := session.state
	return &state, session.filepath, nil
}

func (m *accountImportUploadSessionManager) getSession(sessionID string) (*accountImportUploadSession, bool) {
	if m == nil || strings.TrimSpace(sessionID) == "" {
		return nil, false
	}
	m.mu.RLock()
	session := m.sessions[sessionID]
	m.mu.RUnlock()
	if session == nil {
		return nil, false
	}
	if time.Now().UTC().After(session.state.ExpiresAt) {
		m.deleteSession(sessionID)
		return nil, false
	}
	return session, true
}

func (m *accountImportUploadSessionManager) deleteSession(sessionID string) {
	if m == nil || sessionID == "" {
		return
	}
	m.mu.Lock()
	session := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	deleteTaskStateJSON(context.Background(), accountImportUploadSessionCacheKey(sessionID))
	if session != nil && session.filepath != "" {
		_ = os.Remove(session.filepath)
	}
}

func (m *accountImportUploadSessionManager) updateSession(sessionID string, mutate func(*accountImportUploadSession)) (*accountImportUploadSessionState, error) {
	if m == nil || strings.TrimSpace(sessionID) == "" || mutate == nil {
		return nil, errors.New("invalid upload session update")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session := m.sessions[sessionID]
	if session == nil {
		return nil, errors.New("upload session not found")
	}
	mutate(session)
	storeTaskStateJSON(context.Background(), accountImportUploadSessionCacheKey(sessionID), accountImportUploadSessionTTL, session.state)
	state := session.state
	return &state, nil
}

func (h *AccountHandler) CreateImportUploadSession(c *gin.Context) {
	if h == nil || h.uploadSessionManager == nil {
		response.Error(c, http.StatusServiceUnavailable, "Import upload session manager not available")
		return
	}
	var req createAccountImportUploadSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	state, _, err := h.uploadSessionManager.createSession(req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Created(c, state)
}

func (h *AccountHandler) GetImportUploadSession(c *gin.Context) {
	if h == nil || h.uploadSessionManager == nil {
		response.Error(c, http.StatusServiceUnavailable, "Import upload session manager not available")
		return
	}
	sessionID := strings.TrimSpace(c.Param("session_id"))
	session, ok := h.uploadSessionManager.getSession(sessionID)
	if !ok || session == nil {
		if cached, hit := loadTaskStateJSON[accountImportUploadSessionState](c.Request.Context(), accountImportUploadSessionCacheKey(sessionID)); hit && cached != nil {
			response.Success(c, cached)
			return
		}
		response.NotFound(c, "Upload session not found")
		return
	}
	state := session.state
	response.Success(c, state)
}

func (h *AccountHandler) UploadImportChunk(c *gin.Context) {
	if h == nil || h.uploadSessionManager == nil {
		response.Error(c, http.StatusServiceUnavailable, "Import upload session manager not available")
		return
	}
	sessionID := strings.TrimSpace(c.Param("session_id"))
	session, ok := h.uploadSessionManager.getSession(sessionID)
	if !ok || session == nil {
		response.NotFound(c, "Upload session not found")
		return
	}

	offset, err := strconv.ParseInt(strings.TrimSpace(c.Query("offset")), 10, 64)
	if err != nil || offset < 0 {
		response.BadRequest(c, "invalid offset")
		return
	}
	if offset != session.state.ReceivedBytes {
		response.Error(c, http.StatusConflict, fmt.Sprintf("offset mismatch: expected=%d got=%d", session.state.ReceivedBytes, offset))
		return
	}
	if session.state.Status == accountImportUploadStatusFinalized {
		response.Error(c, http.StatusConflict, "upload session already finalized")
		return
	}

	file, err := os.OpenFile(filepath.Clean(session.filepath), os.O_WRONLY, 0o600)
	if err != nil {
		response.InternalError(c, "failed to open upload file")
		return
	}
	defer func() { _ = file.Close() }()
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		response.InternalError(c, "failed to seek upload file")
		return
	}
	written, err := io.Copy(file, c.Request.Body)
	if err != nil {
		response.InternalError(c, "failed to write upload chunk")
		return
	}
	if written <= 0 {
		response.BadRequest(c, "chunk body is empty")
		return
	}
	nextReceived := offset + written
	if nextReceived > session.state.TotalBytes {
		response.BadRequest(c, "uploaded bytes exceed declared total_bytes")
		return
	}
	state, err := h.uploadSessionManager.updateSession(sessionID, func(s *accountImportUploadSession) {
		s.state.ReceivedBytes = nextReceived
		s.state.UpdatedAt = time.Now().UTC()
		s.state.ExpiresAt = s.state.UpdatedAt.Add(accountImportUploadSessionTTL)
		if nextReceived >= s.state.TotalBytes {
			s.state.Status = accountImportUploadStatusUploaded
		} else {
			s.state.Status = accountImportUploadStatusUploading
		}
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, state)
}

func (h *AccountHandler) FinalizeImportUploadSession(c *gin.Context) {
	if h == nil || h.uploadSessionManager == nil || h.importTaskManager == nil {
		response.Error(c, http.StatusServiceUnavailable, "Import upload session manager not available")
		return
	}
	sessionID := strings.TrimSpace(c.Param("session_id"))
	session, ok := h.uploadSessionManager.getSession(sessionID)
	if !ok || session == nil {
		response.NotFound(c, "Upload session not found")
		return
	}
	if session.state.ReceivedBytes < session.state.TotalBytes {
		response.Error(c, http.StatusConflict, fmt.Sprintf("upload incomplete: received=%d total=%d", session.state.ReceivedBytes, session.state.TotalBytes))
		return
	}
	if session.state.TaskID != "" {
		if task, ok := h.importTaskManager.getTask(session.state.TaskID); ok && task != nil {
			response.Accepted(c, task)
			return
		}
	}

	task := h.importTaskManager.createTask(session.state.Filename, accountImportTaskPayload{
		Filepath:             session.filepath,
		Filename:             session.state.Filename,
		GroupIDs:             session.state.GroupIDs,
		SkipDefaultGroupBind: session.state.SkipDefaultGroupBind,
	})
	if task == nil {
		response.Error(c, http.StatusServiceUnavailable, "Failed to create import task")
		return
	}
	task.execute = func(ctx context.Context, task *accountImportTask, progress func(stage string, current, total int, message string)) (DataImportResult, error) {
		payload, err := loadImportPayloadFromFile(task.payload.Filepath)
		if err != nil {
			return DataImportResult{}, err
		}
		req := DataImportRequest{
			Data:                 payload,
			GroupIDs:             task.payload.GroupIDs,
			SkipDefaultGroupBind: task.payload.SkipDefaultGroupBind,
		}
		if err := validateDataHeader(req.Data); err != nil {
			return DataImportResult{}, err
		}
		return h.importDataWithProgress(ctx, req, progress)
	}
	if err := h.importTaskManager.submitTask(task); err != nil {
		response.Error(c, http.StatusTooManyRequests, err.Error())
		return
	}

	_, _ = h.uploadSessionManager.updateSession(sessionID, func(s *accountImportUploadSession) {
		s.state.Status = accountImportUploadStatusFinalized
		s.state.TaskID = task.state.TaskID
		s.state.UpdatedAt = time.Now().UTC()
		s.filepath = ""
	})

	response.Accepted(c, task.state)
}

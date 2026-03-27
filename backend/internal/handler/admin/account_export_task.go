package admin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type accountExportTaskStatus string

const (
	accountExportTaskQueued    accountExportTaskStatus = "queued"
	accountExportTaskRunning   accountExportTaskStatus = "running"
	accountExportTaskCompleted accountExportTaskStatus = "completed"
	accountExportTaskFailed    accountExportTaskStatus = "failed"
)

const (
	accountExportDownloadTTL       = 30 * time.Minute
	accountExportTaskCleanupPeriod = 5 * time.Minute
	accountExportFailedTaskTTL     = 1 * time.Hour
)

type DataExportTaskFilters struct {
	Platform  string `json:"platform,omitempty"`
	Type      string `json:"type,omitempty"`
	Status    string `json:"status,omitempty"`
	Search    string `json:"search,omitempty"`
	Group     string `json:"group,omitempty"`
	Plan      string `json:"plan,omitempty"`
	OAuthType string `json:"oauth_type,omitempty"`
	TierID    string `json:"tier_id,omitempty"`
}

type DataExportTaskRequest struct {
	IDs            []int64                `json:"ids,omitempty"`
	Filters        *DataExportTaskFilters `json:"filters,omitempty"`
	IncludeProxies *bool                  `json:"include_proxies,omitempty"`
}

type DataExportTaskResult struct {
	Filename      string     `json:"filename,omitempty"`
	AccountCount  int        `json:"account_count"`
	ProxyCount    int        `json:"proxy_count"`
	FileSizeBytes int64      `json:"file_size_bytes"`
	DownloadURL   string     `json:"download_url,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

type accountExportTaskState struct {
	TaskID     string                  `json:"task_id"`
	Status     accountExportTaskStatus `json:"status"`
	Stage      string                  `json:"stage,omitempty"`
	Current    int                     `json:"current"`
	Total      int                     `json:"total"`
	Progress   int                     `json:"progress"`
	Message    string                  `json:"message,omitempty"`
	Result     *DataExportTaskResult   `json:"result,omitempty"`
	CreatedAt  time.Time               `json:"created_at"`
	StartedAt  *time.Time              `json:"started_at,omitempty"`
	FinishedAt *time.Time              `json:"finished_at,omitempty"`
}

type accountExportTaskPayload struct {
	Request DataExportTaskRequest
}

type accountExportArtifact struct {
	Filepath      string
	Filename      string
	AccountCount  int
	ProxyCount    int
	FileSizeBytes int64
}

type accountExportTask struct {
	state         accountExportTaskState
	payload       accountExportTaskPayload
	downloadToken string
	downloadPath  string
	downloadName  string
	expiresAt     *time.Time
	execute       func(context.Context, *accountExportTask, func(stage string, current, total int, message string)) (accountExportArtifact, error)
}

type accountExportTaskManager struct {
	once    sync.Once
	stopCh  chan struct{}
	queue   chan *accountExportTask
	mu      sync.RWMutex
	tasks   map[string]*accountExportTask
	workers sync.WaitGroup
}

var defaultAccountExportTasks = &accountExportTaskManager{
	stopCh: make(chan struct{}),
	queue:  make(chan *accountExportTask, 4),
	tasks:  make(map[string]*accountExportTask),
}

func defaultAccountExportTaskManager() *accountExportTaskManager {
	return defaultAccountExportTasks
}

func accountExportTaskCacheKey(taskID string) string {
	return "task:progress:account_export:" + taskID
}

const accountExportTaskKind = "account_export"

func (m *accountExportTaskManager) ensureStarted() {
	if m == nil {
		return
	}
	m.once.Do(func() {
		m.workers.Add(1)
		go func() {
			defer m.workers.Done()
			ticker := time.NewTicker(accountExportTaskCleanupPeriod)
			defer ticker.Stop()
			for {
				select {
				case <-m.stopCh:
					return
				case <-ticker.C:
					m.cleanupExpiredTasks(time.Now().UTC())
				case task := <-m.queue:
					if task == nil {
						continue
					}
					m.runTaskNow(task)
				}
			}
		}()
	})
}

func (m *accountExportTaskManager) createTask(payload accountExportTaskPayload) *accountExportTask {
	if m == nil {
		return nil
	}
	m.cleanupExpiredTasks(time.Now().UTC())
	now := time.Now().UTC()
	taskID := uuid.NewString()
	task := &accountExportTask{
		state: accountExportTaskState{
			TaskID:    taskID,
			Status:    accountExportTaskQueued,
			Stage:     "queued",
			CreatedAt: now,
		},
		payload:       payload,
		downloadToken: uuid.NewString(),
	}
	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()
	storeTaskStateJSONWithRepo(context.Background(), accountExportTaskCacheKey(taskID), accountExportTaskKind, taskID, accountExportFailedTaskTTL, string(task.state.Status), task.state.FinishedAt, task.state)
	return task
}

func (m *accountExportTaskManager) getTask(taskID string) (*accountExportTaskState, bool) {
	if m == nil || taskID == "" {
		return nil, false
	}
	m.cleanupExpiredTasks(time.Now().UTC())
	m.mu.RLock()
	task := m.tasks[taskID]
	m.mu.RUnlock()
	if task == nil {
		if cached, ok := loadTaskStateJSONWithRepo[accountExportTaskState](context.Background(), accountExportTaskCacheKey(taskID), accountExportTaskKind, taskID); ok && cached != nil {
			return cached, true
		}
		return nil, false
	}
	state := task.state
	return &state, true
}

func (m *accountExportTaskManager) updateTask(taskID string, mutate func(*accountExportTaskState)) {
	if m == nil || taskID == "" || mutate == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	task := m.tasks[taskID]
	if task == nil {
		return
	}
	mutate(&task.state)
	if task.state.Total > 0 {
		progress := int(float64(task.state.Current) / float64(task.state.Total) * 100)
		if progress < 0 {
			progress = 0
		}
		if progress > 100 {
			progress = 100
		}
		task.state.Progress = progress
	}
	storeTaskStateJSONWithRepo(context.Background(), accountExportTaskCacheKey(taskID), accountExportTaskKind, taskID, accountExportFailedTaskTTL, string(task.state.Status), task.state.FinishedAt, task.state)
}

func (m *accountExportTaskManager) submitTask(task *accountExportTask) error {
	if m == nil || task == nil {
		return errors.New("export task manager is not ready")
	}
	m.ensureStarted()
	select {
	case m.queue <- task:
		return nil
	default:
		return errors.New("export task queue is full")
	}
}

func (m *accountExportTaskManager) runTaskNow(task *accountExportTask) {
	if m == nil || task == nil || task.execute == nil {
		return
	}
	now := time.Now().UTC()
	m.updateTask(task.state.TaskID, func(state *accountExportTaskState) {
		state.Status = accountExportTaskRunning
		state.Stage = "running"
		state.StartedAt = &now
	})
	progress := func(stage string, current, total int, message string) {
		m.updateTask(task.state.TaskID, func(state *accountExportTaskState) {
			if stage != "" {
				state.Stage = stage
			}
			if current >= 0 {
				state.Current = current
			}
			if total >= 0 {
				state.Total = total
			}
			if message != "" {
				state.Message = message
			}
		})
	}
	artifact, err := task.execute(context.Background(), task, progress)
	finishedAt := time.Now().UTC()
	if err != nil {
		if artifact.Filepath != "" {
			_ = os.Remove(artifact.Filepath)
		}
		m.updateTask(task.state.TaskID, func(state *accountExportTaskState) {
			state.Status = accountExportTaskFailed
			state.Stage = "failed"
			state.Message = err.Error()
			state.FinishedAt = &finishedAt
		})
		return
	}

	expiresAt := finishedAt.Add(accountExportDownloadTTL)
	downloadURL := buildAccountExportDownloadURL(task.state.TaskID, task.downloadToken)
	m.mu.Lock()
	task.downloadPath = artifact.Filepath
	task.downloadName = artifact.Filename
	task.expiresAt = &expiresAt
	m.mu.Unlock()
	m.updateTask(task.state.TaskID, func(state *accountExportTaskState) {
		state.Status = accountExportTaskCompleted
		state.Stage = "completed"
		state.Message = fmt.Sprintf("Export completed: accounts=%d proxies=%d", artifact.AccountCount, artifact.ProxyCount)
		state.Result = &DataExportTaskResult{
			Filename:      artifact.Filename,
			AccountCount:  artifact.AccountCount,
			ProxyCount:    artifact.ProxyCount,
			FileSizeBytes: artifact.FileSizeBytes,
			DownloadURL:   downloadURL,
			ExpiresAt:     &expiresAt,
		}
		state.Progress = 100
		state.Current = state.Total
		state.FinishedAt = &finishedAt
	})
}

func (m *accountExportTaskManager) resolveDownload(taskID, token string) (path string, filename string, ok bool) {
	if m == nil || strings.TrimSpace(taskID) == "" || strings.TrimSpace(token) == "" {
		return "", "", false
	}
	m.cleanupExpiredTasks(time.Now().UTC())
	m.mu.RLock()
	task := m.tasks[taskID]
	m.mu.RUnlock()
	if task == nil {
		return "", "", false
	}
	if task.downloadToken != token || task.downloadPath == "" || task.downloadName == "" || task.expiresAt == nil {
		return "", "", false
	}
	if !time.Now().UTC().Before(*task.expiresAt) {
		m.cleanupExpiredTasks(time.Now().UTC())
		return "", "", false
	}
	if _, err := os.Stat(filepath.Clean(task.downloadPath)); err != nil {
		return "", "", false
	}
	return task.downloadPath, task.downloadName, true
}

func (m *accountExportTaskManager) cleanupExpiredTasks(now time.Time) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for taskID, task := range m.tasks {
		if !shouldCleanupExportTask(task, now) {
			continue
		}
		deleteTaskStateJSON(context.Background(), accountExportTaskCacheKey(taskID))
		if task != nil && task.downloadPath != "" {
			_ = os.Remove(task.downloadPath)
		}
		delete(m.tasks, taskID)
	}
}

func shouldCleanupExportTask(task *accountExportTask, now time.Time) bool {
	if task == nil {
		return true
	}
	switch task.state.Status {
	case accountExportTaskQueued, accountExportTaskRunning:
		return false
	case accountExportTaskCompleted:
		if task.expiresAt != nil {
			return !now.Before(*task.expiresAt)
		}
		if task.state.FinishedAt != nil {
			return !now.Before(task.state.FinishedAt.Add(accountExportDownloadTTL))
		}
	case accountExportTaskFailed:
		if task.state.FinishedAt != nil {
			return !now.Before(task.state.FinishedAt.Add(accountExportFailedTaskTTL))
		}
	}
	return false
}

func buildAccountExportDownloadURL(taskID, token string) string {
	return fmt.Sprintf("/api/v1/public/account-export-tasks/%s/download?token=%s", taskID, token)
}

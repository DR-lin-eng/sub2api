package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type accountImportTaskStatus string

const (
	accountImportTaskQueued    accountImportTaskStatus = "queued"
	accountImportTaskRunning   accountImportTaskStatus = "running"
	accountImportTaskCompleted accountImportTaskStatus = "completed"
	accountImportTaskFailed    accountImportTaskStatus = "failed"
)

const (
	accountImportTaskCleanupPeriod = 5 * time.Minute
	accountImportTaskTTL           = 1 * time.Hour
)

type accountImportTaskState struct {
	TaskID     string                  `json:"task_id"`
	Status     accountImportTaskStatus `json:"status"`
	Stage      string                  `json:"stage,omitempty"`
	Filename   string                  `json:"filename,omitempty"`
	Current    int                     `json:"current"`
	Total      int                     `json:"total"`
	Progress   int                     `json:"progress"`
	Message    string                  `json:"message,omitempty"`
	Result     *DataImportResult       `json:"result,omitempty"`
	CreatedAt  time.Time               `json:"created_at"`
	StartedAt  *time.Time              `json:"started_at,omitempty"`
	FinishedAt *time.Time              `json:"finished_at,omitempty"`
}

type accountImportTaskPayload struct {
	Filepath             string
	Filename             string
	GroupIDs             []int64
	SkipDefaultGroupBind *bool
}

type accountImportTask struct {
	state   accountImportTaskState
	payload accountImportTaskPayload
	execute func(context.Context, *accountImportTask, func(stage string, current, total int, message string)) (DataImportResult, error)
}

type accountImportTaskManager struct {
	once    sync.Once
	stopCh  chan struct{}
	queue   chan *accountImportTask
	mu      sync.RWMutex
	tasks   map[string]*accountImportTask
	workers sync.WaitGroup
}

func defaultAccountImportTaskManager() *accountImportTaskManager {
	return defaultAccountImportTasks
}

var defaultAccountImportTasks = &accountImportTaskManager{
	stopCh: make(chan struct{}),
	queue:  make(chan *accountImportTask, 8),
	tasks:  make(map[string]*accountImportTask),
}

func (m *accountImportTaskManager) ensureStarted() {
	if m == nil {
		return
	}
	m.once.Do(func() {
		m.workers.Add(1)
		go func() {
			defer m.workers.Done()
			ticker := time.NewTicker(accountImportTaskCleanupPeriod)
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

func (m *accountImportTaskManager) createTask(filename string, payload accountImportTaskPayload) *accountImportTask {
	if m == nil {
		return nil
	}
	m.cleanupExpiredTasks(time.Now().UTC())
	taskID := uuid.NewString()
	task := &accountImportTask{
		state: accountImportTaskState{
			TaskID:    taskID,
			Status:    accountImportTaskQueued,
			Stage:     "queued",
			Filename:  filename,
			CreatedAt: time.Now().UTC(),
		},
		payload: payload,
	}
	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()
	return task
}

func (m *accountImportTaskManager) getTask(taskID string) (*accountImportTaskState, bool) {
	if m == nil || taskID == "" {
		return nil, false
	}
	m.cleanupExpiredTasks(time.Now().UTC())
	m.mu.RLock()
	task := m.tasks[taskID]
	m.mu.RUnlock()
	if task == nil {
		return nil, false
	}
	state := task.state
	return &state, true
}

func (m *accountImportTaskManager) updateTask(taskID string, mutate func(*accountImportTaskState)) {
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
}

func saveImportPayloadToTempFile(raw []byte) (string, error) {
	dir := os.TempDir()
	file, err := os.CreateTemp(dir, "sub2api-account-import-*.json")
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	if _, err := file.Write(raw); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func loadImportPayloadFromFile(path string) (DataPayload, error) {
	var payload DataPayload
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return payload, err
	}
	defer func() { _ = file.Close() }()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func (m *accountImportTaskManager) submitTask(task *accountImportTask) error {
	if m == nil || task == nil {
		return errors.New("import task manager is not ready")
	}
	m.ensureStarted()
	select {
	case m.queue <- task:
		return nil
	default:
		return errors.New("import task queue is full")
	}
}

func (m *accountImportTaskManager) runTaskNow(task *accountImportTask) {
	if m == nil || task == nil || task.execute == nil {
		return
	}
	now := time.Now().UTC()
	m.updateTask(task.state.TaskID, func(state *accountImportTaskState) {
		state.Status = accountImportTaskRunning
		state.Stage = "running"
		state.StartedAt = &now
	})
	defer func() {
		if task.payload.Filepath != "" {
			_ = os.Remove(task.payload.Filepath)
		}
	}()
	progress := func(stage string, current, total int, message string) {
		m.updateTask(task.state.TaskID, func(state *accountImportTaskState) {
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
	result, err := task.execute(context.Background(), task, progress)
	finishedAt := time.Now().UTC()
	if err != nil {
		m.updateTask(task.state.TaskID, func(state *accountImportTaskState) {
			state.Status = accountImportTaskFailed
			state.Stage = "failed"
			state.Message = err.Error()
			state.FinishedAt = &finishedAt
		})
		return
	}
	m.updateTask(task.state.TaskID, func(state *accountImportTaskState) {
		state.Status = accountImportTaskCompleted
		state.Stage = "completed"
		state.Result = &result
		state.Message = fmt.Sprintf("Import completed: created=%d skipped=%d failed=%d", result.AccountCreated, result.AccountSkipped, result.AccountFailed)
		state.Current = state.Total
		state.Progress = 100
		state.FinishedAt = &finishedAt
	})
}

func (m *accountImportTaskManager) cleanupExpiredTasks(now time.Time) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for taskID, task := range m.tasks {
		if !shouldCleanupImportTask(task, now) {
			continue
		}
		delete(m.tasks, taskID)
	}
}

func shouldCleanupImportTask(task *accountImportTask, now time.Time) bool {
	if task == nil {
		return true
	}
	switch task.state.Status {
	case accountImportTaskQueued, accountImportTaskRunning:
		return false
	case accountImportTaskCompleted, accountImportTaskFailed:
		if task.state.FinishedAt != nil {
			return !now.Before(task.state.FinishedAt.Add(accountImportTaskTTL))
		}
	}
	return false
}

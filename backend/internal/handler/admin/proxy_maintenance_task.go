package admin

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

type proxyMaintenanceTaskStatus string

const (
	proxyMaintenanceTaskQueued    proxyMaintenanceTaskStatus = "queued"
	proxyMaintenanceTaskRunning   proxyMaintenanceTaskStatus = "running"
	proxyMaintenanceTaskCompleted proxyMaintenanceTaskStatus = "completed"
	proxyMaintenanceTaskFailed    proxyMaintenanceTaskStatus = "failed"
)

type proxyMaintenanceTaskState struct {
	TaskID     string                          `json:"task_id"`
	Status     proxyMaintenanceTaskStatus      `json:"status"`
	Stage      string                          `json:"stage,omitempty"`
	Message    string                          `json:"message,omitempty"`
	Progress   int                             `json:"progress"`
	Result     *service.ProxyMaintenanceResult `json:"result,omitempty"`
	CreatedAt  time.Time                       `json:"created_at"`
	StartedAt  *time.Time                      `json:"started_at,omitempty"`
	FinishedAt *time.Time                      `json:"finished_at,omitempty"`
}

type proxyMaintenanceTask struct {
	state   proxyMaintenanceTaskState
	execute func(context.Context, *proxyMaintenanceTask) (*service.ProxyMaintenanceResult, error)
}

type proxyMaintenanceTaskManager struct {
	once    sync.Once
	stopCh  chan struct{}
	queue   chan *proxyMaintenanceTask
	mu      sync.RWMutex
	tasks   map[string]*proxyMaintenanceTask
	workers sync.WaitGroup
}

var defaultProxyMaintenanceTasks = &proxyMaintenanceTaskManager{
	stopCh: make(chan struct{}),
	queue:  make(chan *proxyMaintenanceTask, 8),
	tasks:  make(map[string]*proxyMaintenanceTask),
}

func defaultProxyMaintenanceTaskManager() *proxyMaintenanceTaskManager {
	return defaultProxyMaintenanceTasks
}

func proxyMaintenanceTaskCacheKey(taskID string) string {
	return "task:progress:proxy_maintenance:" + taskID
}

const proxyMaintenanceTaskKind = "proxy_maintenance"

func (m *proxyMaintenanceTaskManager) ensureStarted() {
	if m == nil {
		return
	}
	m.once.Do(func() {
		m.workers.Add(1)
		go func() {
			defer m.workers.Done()
			for {
				select {
				case <-m.stopCh:
					return
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

func (m *proxyMaintenanceTaskManager) createTask() *proxyMaintenanceTask {
	if m == nil {
		return nil
	}
	taskID := uuid.NewString()
	task := &proxyMaintenanceTask{
		state: proxyMaintenanceTaskState{
			TaskID:    taskID,
			Status:    proxyMaintenanceTaskQueued,
			Stage:     "queued",
			Progress:  0,
			CreatedAt: time.Now().UTC(),
		},
	}
	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()
	storeTaskStateJSONWithRepo(context.Background(), proxyMaintenanceTaskCacheKey(taskID), proxyMaintenanceTaskKind, taskID, time.Hour, string(task.state.Status), task.state.FinishedAt, task.state)
	return task
}

func (m *proxyMaintenanceTaskManager) submitTask(task *proxyMaintenanceTask) error {
	if m == nil || task == nil {
		return errors.New("proxy maintenance task manager is not ready")
	}
	m.ensureStarted()
	select {
	case m.queue <- task:
		return nil
	default:
		return errors.New("proxy maintenance task queue is full")
	}
}

func (m *proxyMaintenanceTaskManager) getTask(taskID string) (*proxyMaintenanceTaskState, bool) {
	if m == nil || taskID == "" {
		return nil, false
	}
	m.mu.RLock()
	task := m.tasks[taskID]
	m.mu.RUnlock()
	if task == nil {
		if cached, ok := loadTaskStateJSONWithRepo[proxyMaintenanceTaskState](context.Background(), proxyMaintenanceTaskCacheKey(taskID), proxyMaintenanceTaskKind, taskID); ok && cached != nil {
			return cached, true
		}
		return nil, false
	}
	state := task.state
	return &state, true
}

func (m *proxyMaintenanceTaskManager) updateTask(taskID string, mutate func(*proxyMaintenanceTaskState)) {
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
	storeTaskStateJSONWithRepo(context.Background(), proxyMaintenanceTaskCacheKey(taskID), proxyMaintenanceTaskKind, taskID, time.Hour, string(task.state.Status), task.state.FinishedAt, task.state)
}

func (m *proxyMaintenanceTaskManager) runTaskNow(task *proxyMaintenanceTask) {
	if m == nil || task == nil || task.execute == nil {
		return
	}
	now := time.Now().UTC()
	m.updateTask(task.state.TaskID, func(state *proxyMaintenanceTaskState) {
		state.Status = proxyMaintenanceTaskRunning
		state.Stage = "running"
		state.Progress = 10
		state.StartedAt = &now
		state.Message = "proxy maintenance running"
	})
	go func() {
		result, err := task.execute(context.Background(), task)
		finishedAt := time.Now().UTC()
		if err != nil {
			m.updateTask(task.state.TaskID, func(state *proxyMaintenanceTaskState) {
				state.Status = proxyMaintenanceTaskFailed
				state.Stage = "failed"
				state.Progress = 100
				state.Message = err.Error()
				state.FinishedAt = &finishedAt
			})
			return
		}
		m.updateTask(task.state.TaskID, func(state *proxyMaintenanceTaskState) {
			state.Status = proxyMaintenanceTaskCompleted
			state.Stage = "completed"
			state.Progress = 100
			state.Message = result.Summary
			state.Result = result
			state.FinishedAt = &finishedAt
		})
	}()
}

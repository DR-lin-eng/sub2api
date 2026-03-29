package admin

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAccountImportTaskManagerAcceptsBurstSubmissions(t *testing.T) {
	manager := &accountImportTaskManager{
		stopCh:      make(chan struct{}),
		workerCount: 2,
		tasks:       make(map[string]*accountImportTask),
	}
	manager.queueCond = sync.NewCond(&manager.queueMu)
	defer func() {
		close(manager.stopCh)
		manager.queueCond.Broadcast()

		done := make(chan struct{})
		go func() {
			manager.workers.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for import task workers to stop")
		}
	}()

	const taskCount = 32

	release := make(chan struct{})
	var started int32
	var finished int32

	taskIDs := make([]string, 0, taskCount)
	for i := 0; i < taskCount; i++ {
		task := manager.createTask("burst.json", accountImportTaskPayload{})
		require.NotNil(t, task)

		task.execute = func(_ context.Context, _ *accountImportTask, _ func(stage string, current, total int, message string)) (DataImportResult, error) {
			atomic.AddInt32(&started, 1)
			<-release
			atomic.AddInt32(&finished, 1)
			return DataImportResult{AccountEnqueued: 1, PlaceholderCreated: 1}, nil
		}

		require.NoError(t, manager.submitTask(task))
		taskIDs = append(taskIDs, task.state.TaskID)
	}

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&started) >= int32(manager.workerCount)
	}, 2*time.Second, 10*time.Millisecond)

	close(release)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&finished) == taskCount
	}, 5*time.Second, 20*time.Millisecond)

	for _, taskID := range taskIDs {
		state, ok := manager.getTask(taskID)
		require.True(t, ok)
		require.NotNil(t, state)
		require.Equal(t, accountImportTaskCompleted, state.Status)
	}
}

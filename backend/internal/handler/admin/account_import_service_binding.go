package admin

import (
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var (
	defaultAccountImportService   *service.AccountImportService
	defaultAccountImportServiceMu sync.RWMutex
)

func SetDefaultAccountImportService(svc *service.AccountImportService) {
	defaultAccountImportServiceMu.Lock()
	defaultAccountImportService = svc
	defaultAccountImportServiceMu.Unlock()
}

func getDefaultAccountImportService() *service.AccountImportService {
	defaultAccountImportServiceMu.RLock()
	defer defaultAccountImportServiceMu.RUnlock()
	return defaultAccountImportService
}

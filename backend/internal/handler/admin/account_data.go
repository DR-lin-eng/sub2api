package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	dataType       = "sub2api-data"
	legacyDataType = "sub2api-bundle"
	dataVersion    = 1
	dataPageCap    = 1000
)

type DataPayload struct {
	Type       string        `json:"type,omitempty"`
	Version    int           `json:"version,omitempty"`
	ExportedAt string        `json:"exported_at"`
	Proxies    []DataProxy   `json:"proxies"`
	Accounts   []DataAccount `json:"accounts"`
}

type DataProxy struct {
	ProxyKey string `json:"proxy_key"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Status   string `json:"status"`
}

type DataAccount struct {
	Name               string         `json:"name"`
	Notes              *string        `json:"notes,omitempty"`
	Platform           string         `json:"platform"`
	Type               string         `json:"type"`
	Credentials        map[string]any `json:"credentials"`
	Extra              map[string]any `json:"extra,omitempty"`
	GroupIDs           []int64        `json:"group_ids,omitempty"`
	ProxyKey           *string        `json:"proxy_key,omitempty"`
	Concurrency        int            `json:"concurrency"`
	Priority           int            `json:"priority"`
	RateMultiplier     *float64       `json:"rate_multiplier,omitempty"`
	ExpiresAt          *int64         `json:"expires_at,omitempty"`
	AutoPauseOnExpired *bool          `json:"auto_pause_on_expired,omitempty"`
}

type DataImportRequest struct {
	Data                 DataPayload `json:"data"`
	GroupIDs             []int64     `json:"group_ids,omitempty"`
	SkipDefaultGroupBind *bool       `json:"skip_default_group_bind"`
}

type DataImportResult struct {
	ProxyCreated   int               `json:"proxy_created"`
	ProxyReused    int               `json:"proxy_reused"`
	ProxyFailed    int               `json:"proxy_failed"`
	AccountCreated int               `json:"account_created"`
	AccountSkipped int               `json:"account_skipped"`
	AccountFailed  int               `json:"account_failed"`
	Errors         []DataImportError `json:"errors,omitempty"`
}

type DataImportError struct {
	Kind     string `json:"kind"`
	Name     string `json:"name,omitempty"`
	ProxyKey string `json:"proxy_key,omitempty"`
	Message  string `json:"message"`
}

func buildProxyKey(protocol, host string, port int, username, password string) string {
	return fmt.Sprintf("%s|%s|%d|%s|%s", strings.TrimSpace(protocol), strings.TrimSpace(host), port, strings.TrimSpace(username), strings.TrimSpace(password))
}

func (h *AccountHandler) ExportData(c *gin.Context) {
	ctx := c.Request.Context()

	selectedIDs, err := parseAccountIDs(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	includeProxies, err := parseIncludeProxies(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if shouldDownloadExportData(c) && h != nil && h.accountExportService != nil {
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", buildAccountExportFilename()))
		c.Status(http.StatusOK)
		if _, err := h.accountExportService.StreamJSON(ctx, c.Writer, buildAccountExportQueryRequest(c, selectedIDs, includeProxies)); err != nil {
			response.InternalError(c, "Failed to serialize export data")
		}
		return
	}

	accounts, err := h.resolveExportAccounts(ctx, selectedIDs, c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var proxies []service.Proxy
	if includeProxies {
		proxies, err = h.resolveExportProxies(ctx, accounts)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	} else {
		proxies = []service.Proxy{}
	}

	proxyKeyByID := make(map[int64]string, len(proxies))
	dataProxies := make([]DataProxy, 0, len(proxies))
	for i := range proxies {
		p := proxies[i]
		key := buildProxyKey(p.Protocol, p.Host, p.Port, p.Username, p.Password)
		proxyKeyByID[p.ID] = key
		dataProxies = append(dataProxies, DataProxy{
			ProxyKey: key,
			Name:     p.Name,
			Protocol: p.Protocol,
			Host:     p.Host,
			Port:     p.Port,
			Username: p.Username,
			Password: p.Password,
			Status:   p.Status,
		})
	}

	dataAccounts := make([]DataAccount, 0, len(accounts))
	for i := range accounts {
		acc := accounts[i]
		var proxyKey *string
		if acc.ProxyID != nil {
			if key, ok := proxyKeyByID[*acc.ProxyID]; ok {
				proxyKey = &key
			}
		}
		var expiresAt *int64
		if acc.ExpiresAt != nil {
			v := acc.ExpiresAt.Unix()
			expiresAt = &v
		}
		dataAccounts = append(dataAccounts, DataAccount{
			Name:               acc.Name,
			Notes:              acc.Notes,
			Platform:           acc.Platform,
			Type:               acc.Type,
			Credentials:        acc.Credentials,
			Extra:              acc.Extra,
			ProxyKey:           proxyKey,
			Concurrency:        acc.Concurrency,
			Priority:           acc.Priority,
			RateMultiplier:     acc.RateMultiplier,
			ExpiresAt:          expiresAt,
			AutoPauseOnExpired: &acc.AutoPauseOnExpired,
		})
	}

	payload := DataPayload{
		Type:       dataType,
		Version:    dataVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Proxies:    dataProxies,
		Accounts:   dataAccounts,
	}

	response.Success(c, payload)
}

func (h *AccountHandler) CreateExportTask(c *gin.Context) {
	if h == nil || h.exportTaskManager == nil {
		response.Error(c, http.StatusServiceUnavailable, "Export task manager not available")
		return
	}

	req, err := parseAccountExportTaskRequest(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	task := h.exportTaskManager.createTask(accountExportTaskPayload{Request: req})
	if task == nil {
		response.Error(c, http.StatusServiceUnavailable, "Failed to create export task")
		return
	}

	task.execute = func(ctx context.Context, task *accountExportTask, progress func(stage string, current, total int, message string)) (accountExportArtifact, error) {
		return h.buildAccountExportArtifact(ctx, req, progress)
	}

	if err := h.exportTaskManager.submitTask(task); err != nil {
		response.Error(c, http.StatusTooManyRequests, err.Error())
		return
	}

	response.Accepted(c, task.state)
}

func (h *AccountHandler) GetExportTask(c *gin.Context) {
	if h == nil || h.exportTaskManager == nil {
		response.Error(c, http.StatusServiceUnavailable, "Export task manager not available")
		return
	}
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		response.BadRequest(c, "task_id is required")
		return
	}
	task, ok := h.exportTaskManager.getTask(taskID)
	if !ok || task == nil {
		response.NotFound(c, "Export task not found")
		return
	}
	response.Success(c, task)
}

func (h *AccountHandler) DownloadExportTask(c *gin.Context) {
	if h == nil || h.exportTaskManager == nil {
		response.NotFound(c, "Export artifact not found")
		return
	}
	taskID := strings.TrimSpace(c.Param("task_id"))
	token := strings.TrimSpace(c.Query("token"))
	path, filename, ok := h.exportTaskManager.resolveDownload(taskID, token)
	if !ok {
		response.NotFound(c, "Export artifact not found or expired")
		return
	}
	c.FileAttachment(path, filename)
}

func shouldDownloadExportData(c *gin.Context) bool {
	if c == nil {
		return false
	}
	raw := strings.TrimSpace(strings.ToLower(c.Query("download")))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func buildAccountExportFilename() string {
	now := time.Now().UTC()
	return fmt.Sprintf(
		"sub2api-account-%04d%02d%02d%02d%02d%02d.json",
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second(),
	)
}

func parseAccountExportTaskRequest(c *gin.Context) (DataExportTaskRequest, error) {
	var req DataExportTaskRequest
	if c == nil || c.Request == nil {
		return req, errors.New("invalid request")
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, errors.New("Invalid request: " + err.Error())
	}
	req.IDs = normalizeInt64IDList(req.IDs)
	return req, nil
}

func (h *AccountHandler) buildAccountExportArtifact(
	ctx context.Context,
	req DataExportTaskRequest,
	progress func(stage string, current, total int, message string),
) (accountExportArtifact, error) {
	var artifact accountExportArtifact
	if h == nil || h.accountExportService == nil {
		return artifact, fmt.Errorf("account export service unavailable")
	}
	includeProxies := true
	if req.IncludeProxies != nil {
		includeProxies = *req.IncludeProxies
	}
	file, err := os.CreateTemp("", "sub2api-account-export-*.json")
	if err != nil {
		return artifact, err
	}
	defer func() { _ = file.Close() }()
	if progress != nil {
		progress("serializing", 0, 1, "Building export artifact")
	}
	stats, err := h.accountExportService.StreamJSON(ctx, file, buildAccountExportQueryRequestFromTask(req, includeProxies))
	if err != nil {
		_ = os.Remove(file.Name())
		return artifact, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = os.Remove(file.Name())
		return artifact, err
	}
	if progress != nil {
		progress("completed", 3, 3, "Export completed")
	}
	return accountExportArtifact{
		Filepath:      file.Name(),
		Filename:      buildAccountExportFilename(),
		AccountCount:  stats.AccountCount,
		ProxyCount:    stats.ProxyCount,
		FileSizeBytes: info.Size(),
	}, nil
}

func (h *AccountHandler) resolveExportAccountsForTask(ctx context.Context, req DataExportTaskRequest) ([]service.Account, error) {
	if len(req.IDs) > 0 {
		accounts, err := h.adminService.GetAccountsByIDs(ctx, req.IDs)
		if err != nil {
			return nil, err
		}
		out := make([]service.Account, 0, len(accounts))
		for _, acc := range accounts {
			if acc == nil {
				continue
			}
			out = append(out, *acc)
		}
		return out, nil
	}
	filters := req.Filters
	if filters == nil {
		filters = &DataExportTaskFilters{}
	}
	search := strings.TrimSpace(filters.Search)
	if len(search) > 100 {
		search = search[:100]
	}
	return h.listAccountsFiltered(
		ctx,
		strings.TrimSpace(filters.Platform),
		strings.TrimSpace(filters.Type),
		strings.TrimSpace(filters.Status),
		search,
		strings.TrimSpace(filters.Group),
		strings.TrimSpace(filters.Plan),
		strings.TrimSpace(filters.OAuthType),
		strings.TrimSpace(filters.TierID),
	)
}

func buildAccountExportQueryRequest(c *gin.Context, ids []int64, includeProxies bool) service.AccountExportQuery {
	return service.AccountExportQuery{
		IDs: ids,
		Filters: service.AccountExportFilters{
			Platform:  strings.TrimSpace(c.Query("platform")),
			Type:      strings.TrimSpace(c.Query("type")),
			Status:    strings.TrimSpace(c.Query("status")),
			Search:    strings.TrimSpace(c.Query("search")),
			Group:     strings.TrimSpace(c.Query("group")),
			Plan:      strings.TrimSpace(c.Query("plan")),
			OAuthType: strings.TrimSpace(c.Query("oauth_type")),
			TierID:    strings.TrimSpace(c.Query("tier_id")),
		},
		IncludeProxies: includeProxies,
	}
}

func buildAccountExportQueryRequestFromTask(req DataExportTaskRequest, includeProxies bool) service.AccountExportQuery {
	filters := service.AccountExportFilters{}
	if req.Filters != nil {
		filters = service.AccountExportFilters{
			Platform:  strings.TrimSpace(req.Filters.Platform),
			Type:      strings.TrimSpace(req.Filters.Type),
			Status:    strings.TrimSpace(req.Filters.Status),
			Search:    strings.TrimSpace(req.Filters.Search),
			Group:     strings.TrimSpace(req.Filters.Group),
			Plan:      strings.TrimSpace(req.Filters.Plan),
			OAuthType: strings.TrimSpace(req.Filters.OAuthType),
			TierID:    strings.TrimSpace(req.Filters.TierID),
		}
	}
	return service.AccountExportQuery{
		IDs:            normalizeInt64IDList(req.IDs),
		Filters:        filters,
		IncludeProxies: includeProxies,
	}
}

func (h *AccountHandler) ImportData(c *gin.Context) {
	req, err := parseAccountDataImportRequest(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := validateDataHeader(req.Data); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	executeAdminIdempotentJSONFailOpenOnStoreUnavailable(c, "admin.accounts.import_data", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.importData(ctx, req)
	})
}

func (h *AccountHandler) CreateImportTask(c *gin.Context) {
	if h == nil || h.importTaskManager == nil {
		response.Error(c, http.StatusServiceUnavailable, "Import task manager not available")
		return
	}
	var (
		payloadPath string
		filename    string
		groupIDs    []int64
		skipPtr     *bool
		err         error
	)
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.ContentType())), "multipart/form-data") {
		payloadPath, filename, groupIDs, skipPtr, err = persistImportTaskPayloadStreaming(c)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	} else {
		req, parseErr := parseAccountDataImportRequest(c)
		if parseErr != nil {
			response.BadRequest(c, parseErr.Error())
			return
		}
		raw, marshalErr := json.Marshal(req.Data)
		if marshalErr != nil {
			response.BadRequest(c, "Failed to serialize import request")
			return
		}
		payloadPath, err = saveImportPayloadToTempFile(raw)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		filename = "import.json"
		groupIDs = req.GroupIDs
		skipPtr = req.SkipDefaultGroupBind
	}

	task := h.importTaskManager.createTask(filename, accountImportTaskPayload{
		Filepath:             payloadPath,
		Filename:             filename,
		GroupIDs:             groupIDs,
		SkipDefaultGroupBind: skipPtr,
	})
	if task == nil {
		_ = os.Remove(payloadPath)
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
		_ = os.Remove(payloadPath)
		response.Error(c, http.StatusTooManyRequests, err.Error())
		return
	}
	response.Accepted(c, task.state)
}

func (h *AccountHandler) GetImportTask(c *gin.Context) {
	if h == nil || h.importTaskManager == nil {
		response.Error(c, http.StatusServiceUnavailable, "Import task manager not available")
		return
	}
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		response.BadRequest(c, "task_id is required")
		return
	}
	task, ok := h.importTaskManager.getTask(taskID)
	if !ok || task == nil {
		response.NotFound(c, "Import task not found")
		return
	}
	response.Success(c, task)
}

func parseAccountDataImportRequest(c *gin.Context) (DataImportRequest, error) {
	var req DataImportRequest
	if c == nil || c.Request == nil {
		return req, errors.New("invalid request")
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.ContentType())), "multipart/form-data") {
		payload, err := parseImportPayloadFromMultipart(c)
		if err != nil {
			return req, err
		}
		req.Data = payload
		groupIDs, err := parseImportGroupIDs(c)
		if err != nil {
			return req, err
		}
		req.GroupIDs = groupIDs
		if skip, ok, err := parseOptionalBoolFormValue(c, "skip_default_group_bind"); err != nil {
			return req, err
		} else if ok {
			req.SkipDefaultGroupBind = &skip
		}
		return req, nil
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, errors.New("Invalid request: " + err.Error())
	}
	return req, nil
}

func persistImportTaskPayloadStreaming(c *gin.Context) (path string, filename string, groupIDs []int64, skipPtr *bool, err error) {
	if c == nil || c.Request == nil {
		return "", "", nil, nil, errors.New("invalid request")
	}
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return "", "", nil, nil, errors.New("failed to read multipart body")
	}

	var (
		tmpFile   *os.File
		fileFound bool
	)
	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
			if err != nil {
				_ = os.Remove(tmpFile.Name())
			}
		}
	}()

	for {
		part, partErr := reader.NextPart()
		if errors.Is(partErr, io.EOF) {
			break
		}
		if partErr != nil {
			return "", "", nil, nil, fmt.Errorf("failed to read multipart part: %w", partErr)
		}

		formName := strings.TrimSpace(part.FormName())
		switch formName {
		case "file":
			if fileFound {
				_ = part.Close()
				return "", "", nil, nil, errors.New("multiple import files are not allowed")
			}
			tmpFile, err = os.CreateTemp("", "sub2api-account-import-*.json")
			if err != nil {
				_ = part.Close()
				return "", "", nil, nil, err
			}
			filename = part.FileName()
			if filename == "" {
				filename = "import.json"
			}
			if _, err = io.Copy(tmpFile, part); err != nil {
				_ = part.Close()
				return "", "", nil, nil, err
			}
			fileFound = true
		case "group_ids":
			var raw []byte
			raw, err = io.ReadAll(io.LimitReader(part, 32*1024))
			if err != nil {
				_ = part.Close()
				return "", "", nil, nil, err
			}
			groupIDs, err = appendImportGroupIDs(groupIDs, string(raw))
			if err != nil {
				_ = part.Close()
				return "", "", nil, nil, err
			}
		case "skip_default_group_bind":
			var raw []byte
			raw, err = io.ReadAll(io.LimitReader(part, 1024))
			if err != nil {
				_ = part.Close()
				return "", "", nil, nil, err
			}
			value := strings.TrimSpace(string(raw))
			if value != "" {
				parsed, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					_ = part.Close()
					return "", "", nil, nil, fmt.Errorf("invalid skip_default_group_bind value: %s", value)
				}
				skipPtr = &parsed
			}
		default:
			_, _ = io.Copy(io.Discard, part)
		}
		_ = part.Close()
	}

	if !fileFound || tmpFile == nil {
		return "", "", nil, nil, errors.New("import file is required")
	}
	if stat, statErr := tmpFile.Stat(); statErr != nil {
		return "", "", nil, nil, statErr
	} else if stat.Size() <= 0 {
		return "", "", nil, nil, errors.New("import file is empty")
	}

	return tmpFile.Name(), filename, normalizeInt64IDList(groupIDs), skipPtr, nil
}

func persistImportTaskPayload(c *gin.Context) (path string, filename string, err error) {
	if c == nil || c.Request == nil {
		return "", "", errors.New("invalid request")
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.ContentType())), "multipart/form-data") {
		file, header, formErr := c.Request.FormFile("file")
		if formErr != nil {
			return "", "", errors.New("import file is required")
		}
		defer func() { _ = file.Close() }()
		tmp, createErr := os.CreateTemp("", "sub2api-account-import-*.json")
		if createErr != nil {
			return "", "", createErr
		}
		defer func() {
			_ = tmp.Close()
			if err != nil {
				_ = os.Remove(tmp.Name())
			}
		}()
		if _, err = io.Copy(tmp, file); err != nil {
			return "", "", err
		}
		return tmp.Name(), header.Filename, nil
	}
	raw, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil {
		return "", "", errors.New("failed to read request body")
	}
	if len(raw) == 0 {
		return "", "", errors.New("request body is empty")
	}
	path, err = saveImportPayloadToTempFile(raw)
	if err != nil {
		return "", "", err
	}
	return path, "import.json", nil
}

func (h *AccountHandler) importData(ctx context.Context, req DataImportRequest) (DataImportResult, error) {
	return h.importDataWithProgress(ctx, req, nil)
}

func (h *AccountHandler) importDataWithProgress(
	ctx context.Context,
	req DataImportRequest,
	progress func(stage string, current, total int, message string),
) (DataImportResult, error) {
	skipDefaultGroupBind := true
	if req.SkipDefaultGroupBind != nil {
		skipDefaultGroupBind = *req.SkipDefaultGroupBind
	}
	importGroupIDs := normalizeInt64IDList(req.GroupIDs)

	dataPayload := req.Data
	result := DataImportResult{}
	total := len(dataPayload.Proxies) + len(dataPayload.Accounts)
	current := 0
	reportProgress := func(stage, message string) {
		if progress != nil {
			progress(stage, current, total, message)
		}
	}
	reportProgress("preparing", "Preparing import task")

	existingProxies, err := h.listAllProxies(ctx)
	if err != nil {
		return result, err
	}
	existingAccounts, err := h.listAllAccounts(ctx)
	if err != nil {
		return result, err
	}

	proxyKeyToID := make(map[string]int64, len(existingProxies))
	for i := range existingProxies {
		p := existingProxies[i]
		key := buildProxyKey(p.Protocol, p.Host, p.Port, p.Username, p.Password)
		proxyKeyToID[key] = p.ID
	}
	accountDedupKeys := make(map[string]int64, len(existingAccounts))
	for i := range existingAccounts {
		if dedupKey := buildExistingAccountDedupKey(&existingAccounts[i]); dedupKey != "" {
			accountDedupKeys[dedupKey] = existingAccounts[i].ID
		}
	}

	for i := range dataPayload.Proxies {
		reportProgress("importing_proxies", "Importing proxies")
		item := dataPayload.Proxies[i]
		key := item.ProxyKey
		if key == "" {
			key = buildProxyKey(item.Protocol, item.Host, item.Port, item.Username, item.Password)
		}
		if err := validateDataProxy(item); err != nil {
			result.ProxyFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:     "proxy",
				Name:     item.Name,
				ProxyKey: key,
				Message:  err.Error(),
			})
			current++
			continue
		}
		normalizedStatus := normalizeProxyStatus(item.Status)
		if existingID, ok := proxyKeyToID[key]; ok {
			proxyKeyToID[key] = existingID
			result.ProxyReused++
			if normalizedStatus != "" {
				if proxy, getErr := h.adminService.GetProxy(ctx, existingID); getErr == nil && proxy != nil && proxy.Status != normalizedStatus {
					_, _ = h.adminService.UpdateProxy(ctx, existingID, &service.UpdateProxyInput{
						Status: normalizedStatus,
					})
				}
			}
			current++
			continue
		}

		created, createErr := h.adminService.CreateProxy(ctx, &service.CreateProxyInput{
			Name:     defaultProxyName(item.Name),
			Protocol: item.Protocol,
			Host:     item.Host,
			Port:     item.Port,
			Username: item.Username,
			Password: item.Password,
		})
		if createErr != nil {
			result.ProxyFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:     "proxy",
				Name:     item.Name,
				ProxyKey: key,
				Message:  createErr.Error(),
			})
			current++
			continue
		}
		proxyKeyToID[key] = created.ID
		result.ProxyCreated++

		if normalizedStatus != "" && normalizedStatus != created.Status {
			_, _ = h.adminService.UpdateProxy(ctx, created.ID, &service.UpdateProxyInput{
				Status: normalizedStatus,
			})
		}
		current++
	}

	for i := range dataPayload.Accounts {
		reportProgress("importing_accounts", "Importing accounts")
		item := dataPayload.Accounts[i]
		if err := validateDataAccount(item); err != nil {
			result.AccountFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:    "account",
				Name:    item.Name,
				Message: err.Error(),
			})
			current++
			continue
		}

		var proxyID *int64
		if item.ProxyKey != nil && *item.ProxyKey != "" {
			if id, ok := proxyKeyToID[*item.ProxyKey]; ok {
				proxyID = &id
			} else {
				result.AccountFailed++
				result.Errors = append(result.Errors, DataImportError{
					Kind:     "account",
					Name:     item.Name,
					ProxyKey: *item.ProxyKey,
					Message:  "proxy_key not found",
				})
				current++
				continue
			}
		}

		enrichCredentialsFromIDToken(&item)
		dedupKey := buildImportedAccountDedupKey(&item)
		if dedupKey != "" {
			if existingID, ok := accountDedupKeys[dedupKey]; ok && existingID > 0 {
				result.AccountSkipped++
				current++
				continue
			}
		}
		groupIDs := normalizeInt64IDList(item.GroupIDs)
		if len(groupIDs) == 0 {
			groupIDs = importGroupIDs
		}

		accountInput := &service.CreateAccountInput{
			Name:                 item.Name,
			Notes:                item.Notes,
			Platform:             item.Platform,
			Type:                 item.Type,
			Credentials:          item.Credentials,
			Extra:                item.Extra,
			ProxyID:              proxyID,
			Concurrency:          item.Concurrency,
			Priority:             item.Priority,
			RateMultiplier:       item.RateMultiplier,
			GroupIDs:             groupIDs,
			ExpiresAt:            item.ExpiresAt,
			AutoPauseOnExpired:   item.AutoPauseOnExpired,
			SkipDefaultGroupBind: skipDefaultGroupBind,
		}

		if _, err := h.adminService.CreateAccount(ctx, accountInput); err != nil {
			result.AccountFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:    "account",
				Name:    item.Name,
				Message: err.Error(),
			})
			current++
			continue
		}
		if dedupKey != "" {
			accountDedupKeys[dedupKey] = int64(result.AccountCreated + result.AccountSkipped + result.AccountFailed + 1)
		}
		result.AccountCreated++
		current++
	}

	reportProgress("completed", "Import completed")
	return result, nil
}

func (h *AccountHandler) listAllProxies(ctx context.Context) ([]service.Proxy, error) {
	page := 1
	pageSize := dataPageLimit()
	var out []service.Proxy
	for {
		items, total, err := h.adminService.ListProxies(ctx, page, pageSize, "", "", "")
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(out) >= int(total) || len(items) == 0 {
			break
		}
		page++
	}
	return out, nil
}

func (h *AccountHandler) listAllAccounts(ctx context.Context) ([]service.Account, error) {
	page := 1
	pageSize := dataPageLimit()
	var out []service.Account
	for {
		items, total, err := h.adminService.ListAccounts(ctx, page, pageSize, "", "", "", "", "", "", "", 0)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(out) >= int(total) || len(items) == 0 {
			break
		}
		page++
	}
	return out, nil
}

func buildExistingAccountDedupKey(account *service.Account) string {
	if account == nil {
		return ""
	}
	return buildAccountDedupKey(
		account.Platform,
		account.Type,
		account.Name,
		account.GetCredential("api_key"),
		account.GetCredential("access_token"),
		account.GetCredential("refresh_token"),
		account.GetCredential("chatgpt_account_id"),
		account.GetCredential("chatgpt_user_id"),
		account.GetCredential("project_id"),
		account.GetCredential("email"),
		account.GetCredential("base_url"),
	)
}

func buildImportedAccountDedupKey(item *DataAccount) string {
	if item == nil {
		return ""
	}
	get := func(key string) string {
		if item.Credentials == nil {
			return ""
		}
		if raw, ok := item.Credentials[key]; ok && raw != nil {
			return strings.TrimSpace(fmt.Sprintf("%v", raw))
		}
		return ""
	}
	return buildAccountDedupKey(
		item.Platform,
		item.Type,
		item.Name,
		get("api_key"),
		get("access_token"),
		get("refresh_token"),
		get("chatgpt_account_id"),
		get("chatgpt_user_id"),
		get("project_id"),
		get("email"),
		get("base_url"),
	)
}

func buildAccountDedupKey(platform, accountType, name, apiKey, accessToken, refreshToken, chatgptAccountID, chatgptUserID, projectID, email, baseURL string) string {
	normalizedPlatform := strings.ToLower(strings.TrimSpace(platform))
	normalizedType := strings.ToLower(strings.TrimSpace(accountType))
	normalizedBaseURL := strings.ToLower(strings.TrimSpace(baseURL))
	normalizedName := strings.ToLower(strings.TrimSpace(name))

	switch {
	case strings.TrimSpace(apiKey) != "":
		return fmt.Sprintf("%s|%s|api_key|%s|%s", normalizedPlatform, normalizedType, strings.TrimSpace(apiKey), normalizedBaseURL)
	case strings.TrimSpace(chatgptAccountID) != "":
		return fmt.Sprintf("%s|%s|chatgpt_account_id|%s", normalizedPlatform, normalizedType, strings.TrimSpace(chatgptAccountID))
	case strings.TrimSpace(chatgptUserID) != "":
		return fmt.Sprintf("%s|%s|chatgpt_user_id|%s", normalizedPlatform, normalizedType, strings.TrimSpace(chatgptUserID))
	case strings.TrimSpace(projectID) != "":
		return fmt.Sprintf("%s|%s|project_id|%s", normalizedPlatform, normalizedType, strings.TrimSpace(projectID))
	case strings.TrimSpace(refreshToken) != "":
		return fmt.Sprintf("%s|%s|refresh_token|%s", normalizedPlatform, normalizedType, strings.TrimSpace(refreshToken))
	case strings.TrimSpace(accessToken) != "":
		return fmt.Sprintf("%s|%s|access_token|%s", normalizedPlatform, normalizedType, strings.TrimSpace(accessToken))
	case strings.TrimSpace(email) != "":
		return fmt.Sprintf("%s|%s|email|%s", normalizedPlatform, normalizedType, strings.ToLower(strings.TrimSpace(email)))
	case normalizedName != "":
		return fmt.Sprintf("%s|%s|name|%s", normalizedPlatform, normalizedType, normalizedName)
	default:
		return ""
	}
}

func (h *AccountHandler) listAccountsFiltered(ctx context.Context, platform, accountType, status, search, group, plan, oauthType, tierID string) ([]service.Account, error) {
	var groupID int64
	if strings.TrimSpace(group) != "" {
		if strings.TrimSpace(group) == accountListGroupUngroupedQueryValue {
			groupID = service.AccountListGroupUngrouped
		} else {
			parsedGroupID, err := strconv.ParseInt(strings.TrimSpace(group), 10, 64)
			if err != nil || parsedGroupID < 0 {
				return nil, fmt.Errorf("invalid group filter: %s", group)
			}
			groupID = parsedGroupID
		}
	}
	page := 1
	pageSize := dataPageLimit()
	var out []service.Account
	for {
		items, total, err := h.adminService.ListAccounts(ctx, page, pageSize, platform, accountType, status, search, plan, oauthType, tierID, groupID)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(out) >= int(total) || len(items) == 0 {
			break
		}
		page++
	}
	return out, nil
}

func dataPageLimit() int {
	return pagination.PaginationParams{Page: 1, PageSize: dataPageCap}.Limit()
}

func (h *AccountHandler) resolveExportAccounts(ctx context.Context, ids []int64, c *gin.Context) ([]service.Account, error) {
	if len(ids) > 0 {
		accounts, err := h.adminService.GetAccountsByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		out := make([]service.Account, 0, len(accounts))
		for _, acc := range accounts {
			if acc == nil {
				continue
			}
			out = append(out, *acc)
		}
		return out, nil
	}

	platform := c.Query("platform")
	accountType := c.Query("type")
	status := c.Query("status")
	search := strings.TrimSpace(c.Query("search"))
	group := strings.TrimSpace(c.Query("group"))
	plan := strings.TrimSpace(c.Query("plan"))
	oauthType := strings.TrimSpace(c.Query("oauth_type"))
	tierID := strings.TrimSpace(c.Query("tier_id"))
	if len(search) > 100 {
		search = search[:100]
	}
	return h.listAccountsFiltered(ctx, platform, accountType, status, search, group, plan, oauthType, tierID)
}

func (h *AccountHandler) resolveExportProxies(ctx context.Context, accounts []service.Account) ([]service.Proxy, error) {
	if len(accounts) == 0 {
		return []service.Proxy{}, nil
	}

	seen := make(map[int64]struct{})
	ids := make([]int64, 0)
	for i := range accounts {
		if accounts[i].ProxyID == nil {
			continue
		}
		id := *accounts[i].ProxyID
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return []service.Proxy{}, nil
	}

	return h.adminService.GetProxiesByIDs(ctx, ids)
}

func parseAccountIDs(c *gin.Context) ([]int64, error) {
	values := c.QueryArray("ids")
	if len(values) == 0 {
		raw := strings.TrimSpace(c.Query("ids"))
		if raw != "" {
			values = []string{raw}
		}
	}
	if len(values) == 0 {
		return nil, nil
	}

	ids := make([]int64, 0, len(values))
	for _, item := range values {
		for _, part := range strings.Split(item, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("invalid account id: %s", part)
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func parseIncludeProxies(c *gin.Context) (bool, error) {
	raw := strings.TrimSpace(strings.ToLower(c.Query("include_proxies")))
	if raw == "" {
		return true, nil
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return true, fmt.Errorf("invalid include_proxies value: %s", raw)
	}
}

func validateDataHeader(payload DataPayload) error {
	if payload.Type != "" && payload.Type != dataType && payload.Type != legacyDataType {
		return fmt.Errorf("unsupported data type: %s", payload.Type)
	}
	if payload.Version != 0 && payload.Version != dataVersion {
		return fmt.Errorf("unsupported data version: %d", payload.Version)
	}
	if payload.Proxies == nil {
		return errors.New("proxies is required")
	}
	if payload.Accounts == nil {
		return errors.New("accounts is required")
	}
	return nil
}

func validateDataProxy(item DataProxy) error {
	if strings.TrimSpace(item.Protocol) == "" {
		return errors.New("proxy protocol is required")
	}
	if strings.TrimSpace(item.Host) == "" {
		return errors.New("proxy host is required")
	}
	if item.Port <= 0 || item.Port > 65535 {
		return errors.New("proxy port is invalid")
	}
	switch item.Protocol {
	case "http", "https", "socks4", "socks5", "socks5h":
	default:
		return fmt.Errorf("proxy protocol is invalid: %s", item.Protocol)
	}
	if item.Status != "" {
		normalizedStatus := normalizeProxyStatus(item.Status)
		if normalizedStatus != service.StatusActive && normalizedStatus != "inactive" {
			return fmt.Errorf("proxy status is invalid: %s", item.Status)
		}
	}
	return nil
}

func validateDataAccount(item DataAccount) error {
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("account name is required")
	}
	if strings.TrimSpace(item.Platform) == "" {
		return errors.New("account platform is required")
	}
	if strings.TrimSpace(item.Type) == "" {
		return errors.New("account type is required")
	}
	if len(item.Credentials) == 0 {
		return errors.New("account credentials is required")
	}
	switch item.Type {
	case service.AccountTypeOAuth, service.AccountTypeSetupToken, service.AccountTypeAPIKey, service.AccountTypeUpstream:
	default:
		return fmt.Errorf("account type is invalid: %s", item.Type)
	}
	if item.RateMultiplier != nil && *item.RateMultiplier < 0 {
		return errors.New("rate_multiplier must be >= 0")
	}
	if item.Concurrency < 0 {
		return errors.New("concurrency must be >= 0")
	}
	if item.Priority < 0 {
		return errors.New("priority must be >= 0")
	}
	return nil
}

func defaultProxyName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "imported-proxy"
	}
	return name
}

// enrichCredentialsFromIDToken performs best-effort extraction of user info fields
// (email, plan_type, chatgpt_account_id, etc.) from id_token in credentials.
// Only applies to OpenAI/Sora OAuth accounts. Skips expired token errors silently.
// Existing credential values are never overwritten — only missing fields are filled.
func enrichCredentialsFromIDToken(item *DataAccount) {
	if item.Credentials == nil {
		return
	}
	// Only enrich OpenAI/Sora OAuth accounts
	platform := strings.ToLower(strings.TrimSpace(item.Platform))
	if platform != service.PlatformOpenAI && platform != service.PlatformSora {
		return
	}
	if strings.ToLower(strings.TrimSpace(item.Type)) != service.AccountTypeOAuth {
		return
	}

	idToken, _ := item.Credentials["id_token"].(string)
	if strings.TrimSpace(idToken) == "" {
		return
	}

	// DecodeIDToken skips expiry validation — safe for imported data
	claims, err := openai.DecodeIDToken(idToken)
	if err != nil {
		slog.Debug("import_enrich_id_token_decode_failed", "account", item.Name, "error", err)
		return
	}

	userInfo := claims.GetUserInfo()
	if userInfo == nil {
		return
	}

	// Fill missing fields only (never overwrite existing values)
	setIfMissing := func(key, value string) {
		if value == "" {
			return
		}
		if existing, _ := item.Credentials[key].(string); existing == "" {
			item.Credentials[key] = value
		}
	}

	setIfMissing("email", userInfo.Email)
	setIfMissing("plan_type", userInfo.PlanType)
	setIfMissing("chatgpt_account_id", userInfo.ChatGPTAccountID)
	setIfMissing("chatgpt_user_id", userInfo.ChatGPTUserID)
	setIfMissing("organization_id", userInfo.OrganizationID)
}

func normalizeProxyStatus(status string) string {
	normalized := strings.TrimSpace(strings.ToLower(status))
	switch normalized {
	case "":
		return ""
	case service.StatusActive:
		return service.StatusActive
	case "inactive", service.StatusDisabled:
		return "inactive"
	default:
		return normalized
	}
}

func parseImportPayloadFromMultipart(c *gin.Context) (DataPayload, error) {
	var payload DataPayload
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		return payload, errors.New("import file is required")
	}
	defer func() { _ = file.Close() }()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return payload, errors.New("import file is empty")
		}
		return payload, fmt.Errorf("invalid import file: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return payload, errors.New("invalid import file: multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return payload, fmt.Errorf("invalid import file: %w", err)
	}
	return payload, nil
}

func parseImportGroupIDs(c *gin.Context) ([]int64, error) {
	values := c.PostFormArray("group_ids")
	if len(values) == 0 {
		if raw := strings.TrimSpace(c.PostForm("group_ids")); raw != "" {
			values = []string{raw}
		}
	}
	if len(values) == 0 {
		return nil, nil
	}
	ids := make([]int64, 0, len(values))
	for _, item := range values {
		for _, part := range strings.Split(item, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("invalid group_ids value: %s", part)
			}
			ids = append(ids, id)
		}
	}
	return normalizeInt64IDList(ids), nil
}

func appendImportGroupIDs(ids []int64, raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ids, nil
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid group_ids value: %s", part)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func parseOptionalBoolFormValue(c *gin.Context, key string) (bool, bool, error) {
	raw := strings.TrimSpace(c.PostForm(key))
	if raw == "" {
		return false, false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false, fmt.Errorf("invalid %s value: %s", key, raw)
	}
	return value, true, nil
}

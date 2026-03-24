package service

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

type AccountExportService struct {
	db *sql.DB
}

type AccountExportFilters struct {
	Platform  string
	Type      string
	Status    string
	Search    string
	Group     string
	Plan      string
	OAuthType string
	TierID    string
}

type AccountExportQuery struct {
	IDs     []int64
	Filters AccountExportFilters
	IncludeProxies bool
}

type AccountExportStats struct {
	AccountCount int
	ProxyCount   int
}

func NewAccountExportService(db *sql.DB) *AccountExportService {
	return &AccountExportService{db: db}
}

func (s *AccountExportService) StreamJSON(ctx context.Context, writer io.Writer, query AccountExportQuery) (AccountExportStats, error) {
	if s == nil || s.db == nil {
		return AccountExportStats{}, fmt.Errorf("account export service unavailable")
	}
	includeProxies := query.IncludeProxies
	buffered := bufio.NewWriterSize(writer, 256*1024)
	defer func() { _ = buffered.Flush() }()

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := io.WriteString(buffered, `{"type":"sub2api-data","version":1,"exported_at":`+strconv.Quote(now)+`,"accounts":[`); err != nil {
		return AccountExportStats{}, err
	}

	proxyIDs := make(map[int64]struct{})
	accountCount, err := s.streamAccounts(ctx, buffered, query, proxyIDs)
	if err != nil {
		return AccountExportStats{}, err
	}

	if _, err := io.WriteString(buffered, `],"proxies":[`); err != nil {
		return AccountExportStats{}, err
	}

	proxyCount := 0
	if includeProxies {
		count, err := s.streamProxies(ctx, buffered, proxyIDs)
		if err != nil {
			return AccountExportStats{}, err
		}
		proxyCount = count
	}

	if _, err := io.WriteString(buffered, `]}`); err != nil {
		return AccountExportStats{}, err
	}
	if err := buffered.Flush(); err != nil {
		return AccountExportStats{}, err
	}
	return AccountExportStats{AccountCount: accountCount, ProxyCount: proxyCount}, nil
}

func (s *AccountExportService) streamAccounts(ctx context.Context, writer io.Writer, query AccountExportQuery, proxyIDs map[int64]struct{}) (int, error) {
	if len(query.IDs) > 0 {
		return s.streamSelectedAccounts(ctx, writer, normalizeExportIDs(query.IDs), proxyIDs)
	}
	return s.streamFilteredAccounts(ctx, writer, query, proxyIDs)
}

func (s *AccountExportService) streamSelectedAccounts(ctx context.Context, writer io.Writer, ids []int64, proxyIDs map[int64]struct{}) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	wroteAny := false
	totalCount := 0
	for start := 0; start < len(ids); start += 1000 {
		end := start + 1000
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]
		rows, err := s.db.QueryContext(ctx, `
			SELECT id, name, notes, platform, type, credentials, extra, proxy_id, concurrency, priority, rate_multiplier, expires_at, auto_pause_on_expired
			FROM accounts
			WHERE deleted_at IS NULL AND id = ANY($1)
			ORDER BY id DESC
		`, pq.Array(chunk))
		if err != nil {
			return totalCount, err
		}
		count, wrote, err := s.writeAccountRows(ctx, rows, writer, wroteAny, proxyIDs)
		_ = rows.Close()
		if err != nil {
			return totalCount, err
		}
		totalCount += count
		wroteAny = wrote
	}
	return totalCount, nil
}

func (s *AccountExportService) streamFilteredAccounts(ctx context.Context, writer io.Writer, query AccountExportQuery, proxyIDs map[int64]struct{}) (int, error) {
	lastID := int64(0)
	wroteAny := false
	totalCount := 0
	for {
		sqlText, args, err := buildAccountExportQuery(query.Filters, lastID, 1000)
		if err != nil {
			return totalCount, err
		}
		rows, err := s.db.QueryContext(ctx, sqlText, args...)
		if err != nil {
			return totalCount, err
		}
		count, wrote, nextLastID, err := s.writeAccountRowsKeyset(ctx, rows, writer, wroteAny, proxyIDs)
		_ = rows.Close()
		if err != nil {
			return totalCount, err
		}
		totalCount += count
		wroteAny = wrote
		if count == 0 {
			break
		}
		lastID = nextLastID
	}
	return totalCount, nil
}

func buildAccountExportQuery(filters AccountExportFilters, lastID int64, limit int) (string, []any, error) {
	conditions := []string{"deleted_at IS NULL"}
	args := make([]any, 0, 10)
	argIndex := 1
	addArg := func(value any) string {
		args = append(args, value)
		arg := fmt.Sprintf("$%d", argIndex)
		argIndex++
		return arg
	}

	if trimmed := strings.TrimSpace(filters.Platform); trimmed != "" {
		conditions = append(conditions, "platform = "+addArg(trimmed))
	}
	if trimmed := strings.TrimSpace(filters.Type); trimmed != "" {
		conditions = append(conditions, "type = "+addArg(trimmed))
	}
	if trimmed := strings.TrimSpace(filters.Status); trimmed != "" {
		switch trimmed {
		case "rate_limited":
			conditions = append(conditions, "rate_limit_reset_at > NOW()")
		case "temp_unschedulable":
			conditions = append(conditions, "temp_unschedulable_until IS NOT NULL AND temp_unschedulable_until > NOW()")
		default:
			conditions = append(conditions, "status = "+addArg(trimmed))
		}
	}
	if trimmed := strings.TrimSpace(filters.Search); trimmed != "" {
		conditions = append(conditions, "name ILIKE '%' || "+addArg(trimmed)+" || '%'")
	}
	if trimmed := strings.ToLower(strings.TrimSpace(filters.Plan)); trimmed != "" {
		conditions = append(conditions, "LOWER(COALESCE(credentials->>'plan_type','')) = "+addArg(trimmed))
	}
	if trimmed := strings.ToLower(strings.TrimSpace(filters.OAuthType)); trimmed != "" {
		conditions = append(conditions, "LOWER(COALESCE(credentials->>'oauth_type','')) = "+addArg(trimmed))
	}
	if trimmed := strings.ToLower(strings.TrimSpace(filters.TierID)); trimmed != "" {
		conditions = append(conditions, "LOWER(COALESCE(credentials->>'tier_id','')) = "+addArg(trimmed))
	}
	if trimmed := strings.TrimSpace(filters.Group); trimmed != "" {
		if trimmed == "ungrouped" {
			conditions = append(conditions, "NOT EXISTS (SELECT 1 FROM account_groups ag WHERE ag.account_id = accounts.id)")
		} else {
			groupID, err := strconv.ParseInt(trimmed, 10, 64)
			if err != nil || groupID <= 0 {
				return "", nil, fmt.Errorf("invalid group filter: %s", trimmed)
			}
			conditions = append(conditions, "EXISTS (SELECT 1 FROM account_groups ag WHERE ag.account_id = accounts.id AND ag.group_id = "+addArg(groupID)+")")
		}
	}
	if lastID > 0 {
		conditions = append(conditions, "id < "+addArg(lastID))
	}

	limitArg := addArg(limit)
	sqlText := `
		SELECT id, name, notes, platform, type, credentials, extra, proxy_id, concurrency, priority, rate_multiplier, expires_at, auto_pause_on_expired
		FROM accounts
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY id DESC
		LIMIT ` + limitArg
	return sqlText, args, nil
}

func (s *AccountExportService) writeAccountRows(ctx context.Context, rows *sql.Rows, writer io.Writer, wroteAny bool, proxyIDs map[int64]struct{}) (count int, finalWroteAny bool, err error) {
	finalWroteAny = wroteAny
	var lastID int64
	count, finalWroteAny, lastID, err = s.writeAccountRowsKeyset(ctx, rows, writer, wroteAny, proxyIDs)
	_ = lastID
	return count, finalWroteAny, err
}

func (s *AccountExportService) writeAccountRowsKeyset(ctx context.Context, rows *sql.Rows, writer io.Writer, wroteAny bool, proxyIDs map[int64]struct{}) (count int, finalWroteAny bool, lastID int64, err error) {
	var ids []int64
	var batchProxyIDs []int64
	entries := make([]struct {
		ID                  int64
		Name                string
		Notes               *string
		Platform            string
		Type                string
		Credentials         map[string]any
		Extra               map[string]any
		ProxyID             *int64
		Concurrency         int
		Priority            int
		RateMultiplier      *float64
		ExpiresAt           *int64
		AutoPauseOnExpired  bool
	}, 0, 256)

	for rows.Next() {
		var (
			id                 int64
			name               string
			notes              sql.NullString
			platform           string
			accountType        string
			credentialsRaw     []byte
			extraRaw           []byte
			proxyID            sql.NullInt64
			concurrency        int
			priority           int
			rateMultiplier     sql.NullFloat64
			expiresAt          sql.NullTime
			autoPauseOnExpired bool
		)
		if err = rows.Scan(&id, &name, &notes, &platform, &accountType, &credentialsRaw, &extraRaw, &proxyID, &concurrency, &priority, &rateMultiplier, &expiresAt, &autoPauseOnExpired); err != nil {
			return count, wroteAny, lastID, err
		}
		lastID = id
		ids = append(ids, id)
		var credentials map[string]any
		if len(credentialsRaw) > 0 {
			_ = json.Unmarshal(credentialsRaw, &credentials)
		}
		var extra map[string]any
		if len(extraRaw) > 0 {
			_ = json.Unmarshal(extraRaw, &extra)
		}
		entry := struct {
			ID                 int64
			Name               string
			Notes              *string
			Platform           string
			Type               string
			Credentials        map[string]any
			Extra              map[string]any
			ProxyID            *int64
			Concurrency        int
			Priority           int
			RateMultiplier     *float64
			ExpiresAt          *int64
			AutoPauseOnExpired bool
		}{
			ID:                 id,
			Name:               name,
			Platform:           platform,
			Type:               accountType,
			Credentials:        credentials,
			Extra:              extra,
			Concurrency:        concurrency,
			Priority:           priority,
			AutoPauseOnExpired: autoPauseOnExpired,
		}
		if notes.Valid {
			value := notes.String
			entry.Notes = &value
		}
		if proxyID.Valid {
			value := proxyID.Int64
			entry.ProxyID = &value
			if proxyIDs != nil && value > 0 {
				proxyIDs[value] = struct{}{}
			}
			batchProxyIDs = append(batchProxyIDs, value)
		}
		if rateMultiplier.Valid {
			value := rateMultiplier.Float64
			entry.RateMultiplier = &value
		}
		if expiresAt.Valid {
			value := expiresAt.Time.Unix()
			entry.ExpiresAt = &value
		}
		entries = append(entries, entry)
	}
	if err = rows.Err(); err != nil {
		return count, wroteAny, lastID, err
	}
	groupIDsByAccount, err := s.loadGroupIDsByAccount(ctx, ids)
	if err != nil {
		return count, wroteAny, lastID, err
	}
	proxyKeyByID, err := s.loadProxyKeyByID(ctx, batchProxyIDs)
	if err != nil {
		return count, wroteAny, lastID, err
	}
	for _, entry := range entries {
		payload := map[string]any{
			"name":                  entry.Name,
			"platform":              entry.Platform,
			"type":                  entry.Type,
			"credentials":           entry.Credentials,
			"concurrency":           entry.Concurrency,
			"priority":              entry.Priority,
			"auto_pause_on_expired": entry.AutoPauseOnExpired,
		}
		if entry.Notes != nil {
			payload["notes"] = *entry.Notes
		}
		if len(entry.Extra) > 0 {
			payload["extra"] = entry.Extra
		}
		if entry.ProxyID != nil {
			if proxyKey, ok := proxyKeyByID[*entry.ProxyID]; ok && proxyKey != "" {
				payload["proxy_key"] = proxyKey
			}
		}
		if entry.RateMultiplier != nil {
			payload["rate_multiplier"] = *entry.RateMultiplier
		}
		if entry.ExpiresAt != nil {
			payload["expires_at"] = *entry.ExpiresAt
		}
		if groupIDs := groupIDsByAccount[entry.ID]; len(groupIDs) > 0 {
			payload["group_ids"] = groupIDs
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return count, wroteAny, lastID, err
		}
		if wroteAny {
			if _, err := io.WriteString(writer, ","); err != nil {
				return count, wroteAny, lastID, err
			}
		}
		if _, err := writer.Write(raw); err != nil {
			return count, wroteAny, lastID, err
		}
		wroteAny = true
		count++
	}
	return count, wroteAny, lastID, nil
}

func (s *AccountExportService) loadGroupIDsByAccount(ctx context.Context, accountIDs []int64) (map[int64][]int64, error) {
	result := make(map[int64][]int64)
	if len(accountIDs) == 0 {
		return result, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id, group_id
		FROM account_groups
		WHERE account_id = ANY($1)
		ORDER BY account_id ASC, group_id ASC
	`, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var accountID int64
		var groupID int64
		if err := rows.Scan(&accountID, &groupID); err != nil {
			return nil, err
		}
		result[accountID] = append(result[accountID], groupID)
	}
	return result, rows.Err()
}

func (s *AccountExportService) streamProxies(ctx context.Context, writer io.Writer, proxyIDs map[int64]struct{}) (int, error) {
	if len(proxyIDs) == 0 {
		return 0, nil
	}
	ids := make([]int64, 0, len(proxyIDs))
	for id := range proxyIDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	wroteAny := false
	totalCount := 0
	for start := 0; start < len(ids); start += 1000 {
		end := start + 1000
		if end > len(ids) {
			end = len(ids)
		}
		rows, err := s.db.QueryContext(ctx, `
			SELECT id, name, protocol, host, port, username, password, status
			FROM proxies
			WHERE deleted_at IS NULL AND id = ANY($1)
			ORDER BY id DESC
		`, pq.Array(ids[start:end]))
		if err != nil {
			return totalCount, err
		}
		for rows.Next() {
			var (
				id       int64
				name     string
				protocol string
				host     string
				port     int
				username sql.NullString
				password sql.NullString
				status   string
			)
			if err := rows.Scan(&id, &name, &protocol, &host, &port, &username, &password, &status); err != nil {
				_ = rows.Close()
				return totalCount, err
			}
			payload := map[string]any{
				"proxy_key": buildAccountExportProxyKey(protocol, host, port, username.String, password.String),
				"name":      name,
				"protocol":  protocol,
				"host":      host,
				"port":      port,
				"status":    status,
			}
			if username.Valid && username.String != "" {
				payload["username"] = username.String
			}
			if password.Valid && password.String != "" {
				payload["password"] = password.String
			}
			raw, err := json.Marshal(payload)
			if err != nil {
				_ = rows.Close()
				return totalCount, err
			}
			if wroteAny {
				if _, err := io.WriteString(writer, ","); err != nil {
					_ = rows.Close()
					return totalCount, err
				}
			}
			if _, err := writer.Write(raw); err != nil {
				_ = rows.Close()
				return totalCount, err
			}
			wroteAny = true
			totalCount++
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return totalCount, err
		}
		_ = rows.Close()
	}
	return totalCount, nil
}

func (s *AccountExportService) loadProxyKeyByID(ctx context.Context, proxyIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	ids := normalizeExportIDs(proxyIDs)
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, protocol, host, port, username, password
		FROM proxies
		WHERE deleted_at IS NULL AND id = ANY($1)
	`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			id       int64
			protocol string
			host     string
			port     int
			username sql.NullString
			password sql.NullString
		)
		if err := rows.Scan(&id, &protocol, &host, &port, &username, &password); err != nil {
			return nil, err
		}
		result[id] = buildAccountExportProxyKey(protocol, host, port, username.String, password.String)
	}
	return result, rows.Err()
}

func normalizeExportIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] > out[j] })
	return out
}

func buildAccountExportProxyKey(protocol, host string, port int, username, password string) string {
	return fmt.Sprintf("%s|%s|%d|%s|%s", strings.TrimSpace(protocol), strings.TrimSpace(host), port, strings.TrimSpace(username), strings.TrimSpace(password))
}

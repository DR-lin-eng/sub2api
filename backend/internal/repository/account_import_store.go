package repository

import (
	"context"
	"database/sql"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func ProvideAccountImportAccountStore(client *dbent.Client, sqlDB *sql.DB, schedulerCache service.SchedulerCache) service.AccountImportAccountStore {
	return newAccountRepositoryWithSQL(client, sqlDB, schedulerCache)
}

func (r *accountRepository) CreateImportPlaceholders(ctx context.Context, accounts []*service.Account) error {
	if len(accounts) == 0 {
		return nil
	}
	builders := make([]*dbent.AccountCreate, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		builder := r.client.Account.Create().
			SetName(account.Name).
			SetNillableNotes(account.Notes).
			SetPlatform(account.Platform).
			SetType(account.Type).
			SetCredentials(normalizeJSONMap(account.Credentials)).
			SetExtra(normalizeJSONMap(account.Extra)).
			SetConcurrency(account.Concurrency).
			SetPriority(account.Priority).
			SetStatus(account.Status).
			SetErrorMessage(account.ErrorMessage).
			SetSchedulable(account.Schedulable).
			SetAutoPauseOnExpired(account.AutoPauseOnExpired)
		if account.RateMultiplier != nil {
			builder.SetRateMultiplier(*account.RateMultiplier)
		}
		if account.LoadFactor != nil {
			builder.SetLoadFactor(*account.LoadFactor)
		}
		if account.ProxyID != nil {
			builder.SetProxyID(*account.ProxyID)
		}
		if account.ExpiresAt != nil {
			builder.SetExpiresAt(*account.ExpiresAt)
		}
		builders = append(builders, builder)
	}
	created, err := r.client.Account.CreateBulk(builders...).Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrAccountNotFound, nil)
	}
	createdByIndex := 0
	for _, account := range accounts {
		if account == nil {
			continue
		}
		entity := created[createdByIndex]
		createdByIndex++
		account.ID = entity.ID
		account.CreatedAt = entity.CreatedAt
		account.UpdatedAt = entity.UpdatedAt
	}
	return nil
}

func (r *accountRepository) LookupMinAccountIDsByDedupFingerprint(ctx context.Context, fingerprints []string) (map[string]int64, error) {
	result := make(map[string]int64, len(fingerprints))
	unique := make([]string, 0, len(fingerprints))
	seen := make(map[string]struct{}, len(fingerprints))
	for _, fingerprint := range fingerprints {
		fingerprint = strings.TrimSpace(fingerprint)
		if fingerprint == "" {
			continue
		}
		if _, exists := seen[fingerprint]; exists {
			continue
		}
		seen[fingerprint] = struct{}{}
		unique = append(unique, fingerprint)
	}
	if len(unique) == 0 {
		return result, nil
	}

	rows, err := r.sql.QueryContext(ctx, `
		SELECT extra->>$1 AS fingerprint, MIN(id) AS min_id
		FROM accounts
		WHERE deleted_at IS NULL
			AND extra->>$1 = ANY($2)
		GROUP BY 1
	`, service.AccountExtraDedupFingerprintKey, pq.Array(unique))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var fingerprint string
		var minID int64
		if err := rows.Scan(&fingerprint, &minID); err != nil {
			return nil, err
		}
		result[fingerprint] = minID
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

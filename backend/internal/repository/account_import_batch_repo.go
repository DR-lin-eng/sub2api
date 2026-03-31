package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	accountImportQueueKey      = "account_import:queue"
	accountImportProcessingKey = "account_import:processing"
	accountImportBatchPrefix   = "account_import:batch:"
	accountImportLeasePrefix   = "account_import:lease:"
)

type accountImportBatchRepository struct {
	rdb *redis.Client
}

type accountImportChunkToken struct {
	BatchID    string  `json:"batch_id"`
	AccountIDs []int64 `json:"account_ids,omitempty"`
	Attempt    int     `json:"attempt"`
}

func NewAccountImportBatchRepository(rdb *redis.Client) service.AccountImportBatchRepository {
	if rdb == nil {
		return nil
	}
	return &accountImportBatchRepository{rdb: rdb}
}

func accountImportBatchKey(batchID string) string {
	return accountImportBatchPrefix + strings.TrimSpace(batchID)
}

func accountImportLeaseKey(token string) string {
	return accountImportLeasePrefix + token
}

func (r *accountImportBatchRepository) EnqueueBatch(ctx context.Context, batch service.AccountImportBatch, chunkSize int, ttl time.Duration) error {
	if r == nil || r.rdb == nil {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = 100
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	rawBatch, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	pipe := r.rdb.Pipeline()
	pipe.Set(ctx, accountImportBatchKey(batch.BatchID), rawBatch, ttl)
	for start := 0; start < len(batch.AccountIDs); start += chunkSize {
		end := start + chunkSize
		if end > len(batch.AccountIDs) {
			end = len(batch.AccountIDs)
		}
		token := accountImportChunkToken{
			BatchID:    batch.BatchID,
			AccountIDs: append([]int64(nil), batch.AccountIDs[start:end]...),
			Attempt:    0,
		}
		rawToken, err := json.Marshal(token)
		if err != nil {
			return err
		}
		pipe.LPush(ctx, accountImportQueueKey, rawToken)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (r *accountImportBatchRepository) ClaimNextChunk(ctx context.Context, workerID string, blockTimeout time.Duration, leaseTTL time.Duration) (*service.AccountImportChunkClaim, error) {
	if r == nil || r.rdb == nil {
		return nil, nil
	}
	if blockTimeout <= 0 {
		blockTimeout = time.Second
	}
	if leaseTTL <= 0 {
		leaseTTL = 2 * time.Minute
	}
	raw, err := r.rdb.BRPopLPush(ctx, accountImportQueueKey, accountImportProcessingKey, blockTimeout).Result()
	if err == redis.Nil || raw == "" {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var token accountImportChunkToken
	if err := json.Unmarshal([]byte(raw), &token); err != nil {
		_ = r.rdb.LRem(ctx, accountImportProcessingKey, 1, raw).Err()
		return nil, err
	}
	if err := r.rdb.Set(ctx, accountImportLeaseKey(raw), strings.TrimSpace(workerID), leaseTTL).Err(); err != nil {
		_ = r.rdb.LRem(ctx, accountImportProcessingKey, 1, raw).Err()
		return nil, err
	}
	return &service.AccountImportChunkClaim{
		Token:      raw,
		BatchID:    token.BatchID,
		AccountIDs: token.AccountIDs,
		Attempt:    token.Attempt,
	}, nil
}

func (r *accountImportBatchRepository) CompleteChunk(ctx context.Context, claim *service.AccountImportChunkClaim, progress service.AccountImportBatchProgress) error {
	return r.finishChunk(ctx, claim, progress, "")
}

func (r *accountImportBatchRepository) RetryChunk(ctx context.Context, claim *service.AccountImportChunkClaim, progress service.AccountImportBatchProgress, maxAttempts int, failureMessage string) error {
	if claim == nil {
		return nil
	}
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	var token accountImportChunkToken
	if err := json.Unmarshal([]byte(claim.Token), &token); err != nil {
		return err
	}
	if token.Attempt+1 >= maxAttempts {
		progress.FailedAccounts += len(token.AccountIDs)
		return r.finishChunk(ctx, claim, progress, failureMessage)
	}
	token.Attempt++
	rawToken, err := json.Marshal(token)
	if err != nil {
		return err
	}

	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, accountImportLeaseKey(claim.Token))
	pipe.LRem(ctx, accountImportProcessingKey, 1, claim.Token)
	pipe.LPush(ctx, accountImportQueueKey, rawToken)
	if strings.TrimSpace(claim.BatchID) != "" {
		if batch, ok := r.loadBatch(ctx, claim.BatchID); ok && batch != nil {
			batch.Progress.LastError = strings.TrimSpace(failureMessage)
			pipe.Set(ctx, accountImportBatchKey(claim.BatchID), mustMarshalJSON(batch), 24*time.Hour)
		}
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (r *accountImportBatchRepository) RequeueExpiredClaims(ctx context.Context, limit int, leaseTTL time.Duration) (int, error) {
	if r == nil || r.rdb == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 100
	}
	values, err := r.rdb.LRange(ctx, accountImportProcessingKey, 0, int64(limit-1)).Result()
	if err != nil && err != redis.Nil {
		return 0, err
	}
	requeued := 0
	for _, raw := range values {
		exists, err := r.rdb.Exists(ctx, accountImportLeaseKey(raw)).Result()
		if err != nil {
			return requeued, err
		}
		if exists > 0 {
			continue
		}
		pipe := r.rdb.Pipeline()
		pipe.LRem(ctx, accountImportProcessingKey, 1, raw)
		pipe.LPush(ctx, accountImportQueueKey, raw)
		if _, err := pipe.Exec(ctx); err != nil {
			return requeued, err
		}
		requeued++
	}
	return requeued, nil
}

func (r *accountImportBatchRepository) finishChunk(ctx context.Context, claim *service.AccountImportChunkClaim, progress service.AccountImportBatchProgress, failureMessage string) error {
	if r == nil || r.rdb == nil || claim == nil {
		return nil
	}
	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, accountImportLeaseKey(claim.Token))
	pipe.LRem(ctx, accountImportProcessingKey, 1, claim.Token)
	if batch, ok := r.loadBatch(ctx, claim.BatchID); ok && batch != nil {
		batch.Progress.CompletedChunks++
		batch.Progress.CompletedAccounts += progress.CompletedAccounts
		batch.Progress.DuplicateAccounts += progress.DuplicateAccounts
		batch.Progress.FailedAccounts += progress.FailedAccounts
		if strings.TrimSpace(failureMessage) != "" {
			batch.Progress.LastError = strings.TrimSpace(failureMessage)
		} else if strings.TrimSpace(progress.LastError) != "" {
			batch.Progress.LastError = strings.TrimSpace(progress.LastError)
		}
		pipe.Set(ctx, accountImportBatchKey(claim.BatchID), mustMarshalJSON(batch), 24*time.Hour)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *accountImportBatchRepository) loadBatch(ctx context.Context, batchID string) (*service.AccountImportBatch, bool) {
	if strings.TrimSpace(batchID) == "" {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, accountImportBatchKey(batchID)).Result()
	if err != nil || raw == "" {
		return nil, false
	}
	var batch service.AccountImportBatch
	if err := json.Unmarshal([]byte(raw), &batch); err != nil {
		return nil, false
	}
	return &batch, true
}

func mustMarshalJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func ProvideAccountImportBatchRepository(rdb *redis.Client) service.AccountImportBatchRepository {
	return NewAccountImportBatchRepository(rdb)
}

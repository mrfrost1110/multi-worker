package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type CacheRepository struct {
	db *Database
}

func NewCacheRepository(db *Database) *CacheRepository {
	return &CacheRepository{db: db}
}

// HashContent creates a SHA256 hash of the content
func (r *CacheRepository) HashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// Exists checks if a content hash exists in the cache
func (r *CacheRepository) Exists(ctx context.Context, contentHash string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM content_cache WHERE content_hash = $1`
	err := r.db.GetContext(ctx, &count, query, contentHash)
	if err != nil {
		return false, fmt.Errorf("failed to check cache: %w", err)
	}
	return count > 0, nil
}

// ExistsForTask checks if content exists for a specific task
func (r *CacheRepository) ExistsForTask(ctx context.Context, contentHash, taskID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM content_cache WHERE content_hash = $1 AND task_id = $2`
	err := r.db.GetContext(ctx, &count, query, contentHash, taskID)
	if err != nil {
		return false, fmt.Errorf("failed to check task cache: %w", err)
	}
	return count > 0, nil
}

// Add adds a content hash to the cache
func (r *CacheRepository) Add(ctx context.Context, contentHash, source, taskID string) error {
	query := `
		INSERT INTO content_cache (content_hash, source, task_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (content_hash, task_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, contentHash, source, taskID)
	if err != nil {
		return fmt.Errorf("failed to add to cache: %w", err)
	}
	return nil
}

// AddBatch adds multiple content hashes to the cache
func (r *CacheRepository) AddBatch(ctx context.Context, hashes []string, source, taskID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO content_cache (content_hash, source, task_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (content_hash, task_id) DO NOTHING
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, hash := range hashes {
		if _, err := stmt.ExecContext(ctx, hash, source, taskID); err != nil {
			return fmt.Errorf("failed to insert hash: %w", err)
		}
	}

	return tx.Commit()
}

// FilterNew returns only the content hashes that don't exist in the cache
func (r *CacheRepository) FilterNew(ctx context.Context, contentHashes []string, taskID string) ([]string, error) {
	if len(contentHashes) == 0 {
		return nil, nil
	}

	var newHashes []string
	for _, hash := range contentHashes {
		exists, err := r.ExistsForTask(ctx, hash, taskID)
		if err != nil {
			return nil, err
		}
		if !exists {
			newHashes = append(newHashes, hash)
		}
	}

	return newHashes, nil
}

// CleanOld removes cache entries older than the specified duration
func (r *CacheRepository) CleanOld(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM content_cache WHERE created_at < $1`
	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to clean old cache: %w", err)
	}
	return result.RowsAffected()
}

// CleanByTask removes all cache entries for a specific task
func (r *CacheRepository) CleanByTask(ctx context.Context, taskID string) error {
	query := `DELETE FROM content_cache WHERE task_id = $1`
	_, err := r.db.ExecContext(ctx, query, taskID)
	return err
}

// Count returns the total number of cached entries
func (r *CacheRepository) Count(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM content_cache`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

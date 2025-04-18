package cache

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// SQLiteCache is a SQLite implementation of the CacheRepository interface
type SQLiteCache struct {
	db          *sql.DB
	logger      *zap.Logger
	cleanupFreq time.Duration
	stopCh      chan struct{}
}

// NewSQLiteCache creates a new SQLite cache
func NewSQLiteCache(dbPath string, logger *zap.Logger, cleanupFreq time.Duration) (*SQLiteCache, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}
	
	// Create table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS spam_cache (
			sender_email TEXT PRIMARY KEY,
			is_spam BOOLEAN,
			score REAL,
			last_seen TIMESTAMP,
			expires_at TIMESTAMP
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}
	
	// Create index on expires_at for faster cleanup
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_expires_at ON spam_cache(expires_at)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create index: %w", err)
	}
	
	cache := &SQLiteCache{
		db:          db,
		logger:      logger,
		cleanupFreq: cleanupFreq,
		stopCh:      make(chan struct{}),
	}
	
	// Start background cleanup
	go cache.startCleanupTask()
	
	return cache, nil
}

// Get retrieves a cached entry for a sender
func (c *SQLiteCache) Get(senderEmail string) (*core.SpamAnalysisResult, bool) {
	var isSpam bool
	var score float32
	var lastSeen, expiresAt string
	
	err := c.db.QueryRow(`
		SELECT is_spam, score, last_seen, expires_at
		FROM spam_cache
		WHERE sender_email = ? AND expires_at > datetime('now')
	`, senderEmail).Scan(&isSpam, &score, &lastSeen, &expiresAt)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false
		}
		c.logger.Error("Failed to query cache", zap.Error(err), zap.String("sender", senderEmail))
		return nil, false
	}
	
	// Parse timestamp
	analyzedAt, err := time.Parse(time.RFC3339, lastSeen)
	if err != nil {
		c.logger.Error("Failed to parse last_seen timestamp", zap.Error(err))
		return nil, false
	}
	
	// Convert to SpamAnalysisResult
	result := &core.SpamAnalysisResult{
		IsSpam:     isSpam,
		Score:      float64(score),
		AnalyzedAt: analyzedAt,
	}
	
	return result, true
}

// Set stores a cache entry
func (c *SQLiteCache) Set(key string, result *core.SpamAnalysisResult, ttl time.Duration) {
	expiresAt := time.Now().Add(ttl)
	
	_, err := c.db.Exec(`
		INSERT OR REPLACE INTO spam_cache (sender_email, is_spam, score, last_seen, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, key, result.IsSpam, float32(result.Score), result.AnalyzedAt.Format(time.RFC3339), expiresAt.Format(time.RFC3339))
	
	if err != nil {
		c.logger.Error("Failed to insert cache entry", zap.Error(err), zap.String("sender", key))
	}
}

// Delete removes a cache entry
func (c *SQLiteCache) Delete(ctx context.Context, senderEmail string) error {
	_, err := c.db.ExecContext(ctx, `
		DELETE FROM spam_cache
		WHERE sender_email = ?
	`, senderEmail)
	
	if err != nil {
		return fmt.Errorf("failed to delete cache entry: %w", err)
	}
	
	return nil
}

// Cleanup removes expired entries
func (c *SQLiteCache) Cleanup(ctx context.Context) error {
	result, err := c.db.ExecContext(ctx, `
		DELETE FROM spam_cache
		WHERE expires_at <= datetime('now')
	`)
	
	if err != nil {
		return fmt.Errorf("failed to clean up expired entries: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.logger.Warn("Failed to get rows affected during cleanup", zap.Error(err))
	} else {
		c.logger.Debug("Cleaned up expired cache entries", zap.Int64("expired_count", rowsAffected))
	}
	
	return nil
}

// startCleanupTask starts a background task to clean up expired entries
func (c *SQLiteCache) startCleanupTask() {
	ticker := time.NewTicker(c.cleanupFreq)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := c.Cleanup(context.Background()); err != nil {
				c.logger.Error("Failed to clean up cache", zap.Error(err))
			}
		case <-c.stopCh:
			return
		}
	}
}

// Stop stops the background cleanup task and closes the database connection
func (c *SQLiteCache) Stop() {
	close(c.stopCh)
	if err := c.db.Close(); err != nil {
		c.logger.Error("Failed to close SQLite database", zap.Error(err))
	}
}

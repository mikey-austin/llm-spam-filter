package cache

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// MySQLCache is a MySQL implementation of the CacheRepository interface
type MySQLCache struct {
	db          *sql.DB
	logger      *zap.Logger
	cleanupFreq time.Duration
	stopCh      chan struct{}
}

// NewMySQLCache creates a new MySQL cache
func NewMySQLCache(dsn string, logger *zap.Logger, cleanupFreq time.Duration) (*MySQLCache, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to MySQL database: %w", err)
	}

	// Create table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS spam_cache (
			sender_email VARCHAR(255) PRIMARY KEY,
			is_spam BOOLEAN,
			score FLOAT,
			last_seen TIMESTAMP,
			expires_at TIMESTAMP,
			INDEX idx_expires_at (expires_at)
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	cache := &MySQLCache{
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
func (c *MySQLCache) Get(ctx context.Context, senderEmail string) (*core.CacheEntry, error) {
	var entry core.CacheEntry
	var lastSeen, expiresAt string

	err := c.db.QueryRowContext(ctx, `
		SELECT sender_email, is_spam, score, last_seen, expires_at
		FROM spam_cache
		WHERE sender_email = ? AND expires_at > NOW()
	`, senderEmail).Scan(&entry.SenderEmail, &entry.IsSpam, &entry.Score, &lastSeen, &expiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query cache: %w", err)
	}

	// Parse timestamps
	entry.LastSeen, err = time.Parse("2006-01-02 15:04:05", lastSeen)
	if err != nil {
		return nil, fmt.Errorf("failed to parse last_seen timestamp: %w", err)
	}

	entry.ExpiresAt, err = time.Parse("2006-01-02 15:04:05", expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expires_at timestamp: %w", err)
	}

	return &entry, nil
}

// Set stores a cache entry
func (c *MySQLCache) Set(ctx context.Context, entry *core.CacheEntry) error {
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO spam_cache (sender_email, is_spam, score, last_seen, expires_at)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			is_spam = VALUES(is_spam),
			score = VALUES(score),
			last_seen = VALUES(last_seen),
			expires_at = VALUES(expires_at)
	`, entry.SenderEmail, entry.IsSpam, entry.Score, entry.LastSeen.Format("2006-01-02 15:04:05"), entry.ExpiresAt.Format("2006-01-02 15:04:05"))

	if err != nil {
		return fmt.Errorf("failed to insert cache entry: %w", err)
	}

	return nil
}

// Delete removes a cache entry
func (c *MySQLCache) Delete(ctx context.Context, senderEmail string) error {
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
func (c *MySQLCache) Cleanup(ctx context.Context) error {
	result, err := c.db.ExecContext(ctx, `
		DELETE FROM spam_cache
		WHERE expires_at <= NOW()
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
func (c *MySQLCache) startCleanupTask() {
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
func (c *MySQLCache) Stop() {
	close(c.stopCh)
	if err := c.db.Close(); err != nil {
		c.logger.Error("Failed to close MySQL database", zap.Error(err))
	}
}

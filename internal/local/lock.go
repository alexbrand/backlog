package local

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DefaultLockTTL is the default time-to-live for a lock.
	DefaultLockTTL = 30 * time.Minute

	// locksDir is the directory name for lock files.
	locksDir = ".locks"
)

// LockFile represents a file-based lock for a task.
type LockFile struct {
	Agent     string
	ClaimedAt time.Time
	ExpiresAt time.Time
}

// lockFilePath returns the path to the lock file for a task.
func (l *Local) lockFilePath(taskID string) string {
	return filepath.Join(l.path, locksDir, taskID+".lock")
}

// readLock reads the lock file for a task if it exists.
// Returns nil if the lock file doesn't exist.
func (l *Local) readLock(taskID string) (*LockFile, error) {
	lockPath := l.lockFilePath(taskID)
	content, err := os.ReadFile(lockPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	return parseLockFile(content)
}

// writeLock writes a lock file for a task.
func (l *Local) writeLock(taskID string, lock *LockFile) error {
	lockPath := l.lockFilePath(taskID)

	// Ensure locks directory exists
	locksPath := filepath.Join(l.path, locksDir)
	if err := os.MkdirAll(locksPath, 0755); err != nil {
		return fmt.Errorf("failed to create locks directory: %w", err)
	}

	content := formatLockFile(lock)
	if err := os.WriteFile(lockPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

// removeLock removes the lock file for a task.
func (l *Local) removeLock(taskID string) error {
	lockPath := l.lockFilePath(taskID)
	err := os.Remove(lockPath)
	if os.IsNotExist(err) {
		return nil // Already removed, no error
	}
	return err
}

// isLockActive checks if a lock is currently active (not expired).
func (lock *LockFile) isActive() bool {
	return time.Now().UTC().Before(lock.ExpiresAt)
}

// parseLockFile parses the content of a lock file.
func parseLockFile(content []byte) (*LockFile, error) {
	lock := &LockFile{}

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "agent:") {
			lock.Agent = strings.TrimSpace(strings.TrimPrefix(line, "agent:"))
		} else if strings.HasPrefix(line, "claimed_at:") {
			ts := strings.TrimSpace(strings.TrimPrefix(line, "claimed_at:"))
			t, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return nil, fmt.Errorf("invalid claimed_at timestamp: %w", err)
			}
			lock.ClaimedAt = t
		} else if strings.HasPrefix(line, "expires_at:") {
			ts := strings.TrimSpace(strings.TrimPrefix(line, "expires_at:"))
			t, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return nil, fmt.Errorf("invalid expires_at timestamp: %w", err)
			}
			lock.ExpiresAt = t
		}
	}

	return lock, nil
}

// formatLockFile formats a lock file for writing.
func formatLockFile(lock *LockFile) string {
	return fmt.Sprintf("agent: %s\nclaimed_at: %s\nexpires_at: %s\n",
		lock.Agent,
		lock.ClaimedAt.Format(time.RFC3339),
		lock.ExpiresAt.Format(time.RFC3339))
}

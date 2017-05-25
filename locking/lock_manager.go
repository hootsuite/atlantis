package locking

import "time"

type Run struct {
	RepoOwner  string
	RepoName   string
	Path       string
	Env        string
	PullID     int
	User       string
	Timestamp  time.Time
}

type TryLockResponse struct {
	LockAcquired bool
	LockingRun   Run // what is currently holding the lock
	LockID       string
}

type LockManager interface {
	TryLock(run Run) (TryLockResponse, error)
	Unlock(lockID string) error
	ListLocks() (map[string]Run, error)
}

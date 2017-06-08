package locking

import (
	"github.com/boltdb/bolt"
	"fmt"
	"encoding/json"
	"github.com/pkg/errors"
	"bytes"
	"os"
	"time"
	"path"
)

type BoltDBLockManager struct {
	db          *bolt.DB
	locksBucket []byte
}

func NewBoltDBLockManager(dataDir string, locksBucket string) (*BoltDBLockManager, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, errors.Wrap(err,"creating data dir")
	}
	db, err := bolt.Open(path.Join(dataDir, "atlantis.db"), 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		if err.Error() == "timeout" {
			return nil, errors.New("starting BoltDB: timeout (a possible cause is another Atlantis instance already running)")
		}
		return nil, errors.Wrap(err,"starting BoltDB")
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(locksBucket)); err != nil {
			return errors.Wrapf(err, "creating %q bucket", locksBucket)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err,"starting BoltDB")
	}
	// todo: close BoltDB when server is sigtermed
	return &BoltDBLockManager{db, []byte(locksBucket)}, nil
}

// NewBoltDBLockManagerWithDB is used for testing
func NewBoltDBLockManagerWithDB(db *bolt.DB, locksBucket string) (*BoltDBLockManager, error) {
	return &BoltDBLockManager{db, []byte(locksBucket)}, nil
}

func (b BoltDBLockManager) TryLock(run Run) (TryLockResponse, error) {
	var response TryLockResponse
	newRunSerialized, _ := json.Marshal(run)
	lockID := run.StateKey()
	transactionErr := b.db.Update(func(tx *bolt.Tx) error {
		locksBucket := tx.Bucket(b.locksBucket)

		// if there is no run at that key then we're free to create the lock
		lockingRunSerialized := locksBucket.Get([]byte(lockID))
		if lockingRunSerialized == nil {
			locksBucket.Put([]byte(lockID), newRunSerialized) // not a readonly bucket so okay to ignore error
			response = TryLockResponse{
				LockAcquired: true,
				LockingRun:   run,
				LockID:       lockID,
			}
			return nil
		}

		// otherwise the lock fails, return to caller the run that's holding the lock
		var lockingRun Run
		if err := b.deserialize(lockingRunSerialized, &lockingRun); err != nil {
			return errors.Wrap(err, "failed to deserialize run")
		}
		response = TryLockResponse{
			LockAcquired: false,
			LockingRun: lockingRun,
			LockID: lockID,
		}
		return nil
	})

	if transactionErr != nil {
		return response, errors.Wrap(transactionErr, "DB transaction failed")
	}

	return response, nil
}

func (b BoltDBLockManager) Unlock(lockID string) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		locks := tx.Bucket(b.locksBucket)
		return locks.Delete([]byte(lockID))
	})
	return errors.Wrap(err, "DB transaction failed")
}

func (b BoltDBLockManager) ListLocks() (map[string]Run, error) {
	m := make(map[string]Run)
	bytes := make(map[string][]byte)

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.locksBucket)
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			bytes[string(k)] = v
		}
		return nil
	})
	if err != nil {
		return m, errors.Wrap(err, "DB transaction failed")
	}

	// deserialize bytes into the proper objects
	for k, v := range bytes {
		var run Run
		if err := b.deserialize(v, &run); err != nil {
			return m, errors.Wrap(err, fmt.Sprintf("failed to deserialize run at key %q", string(k)))
		}
		m[k] = run
	}

	return m, nil
}

func (b BoltDBLockManager) FindLocksForPull(repoFullName string, pullNum int) ([]string, error) {
	var ids []string
	err := b.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(b.locksBucket).Cursor()

		// the key for each lock is repoFullName/path/env so we can scan through all entries
		// and get the locks for that repo. Then we can check if the lock is for the right pull
		prefix := []byte(repoFullName + "/")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var run Run
			if err := b.deserialize(v, &run); err != nil {
				return errors.Wrapf(err, "failed to deserialize run at key %q", string(k))
			}
			if run.PullNum == pullNum {
				ids = append(ids, string(k))
			}
		}

		return nil
	})
	return ids, err
}

func (b BoltDBLockManager) deserialize(bs []byte, run *Run) error {
	return json.Unmarshal(bs, run)
}

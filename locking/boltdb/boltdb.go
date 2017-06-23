package boltdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/hootsuite/atlantis/models"
	"github.com/pkg/errors"
	"os"
	"path"
	"time"
)

type Backend struct {
	db     *bolt.DB
	bucket []byte
}

const bucketName = "runLocks"

func New(dataDir string) (*Backend, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, errors.Wrap(err, "creating data dir")
	}
	db, err := bolt.Open(path.Join(dataDir, "atlantis.db"), 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		if err.Error() == "timeout" {
			return nil, errors.New("starting BoltDB: timeout (a possible cause is another Atlantis instance already running)")
		}
		return nil, errors.Wrap(err, "starting BoltDB")
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketName)); err != nil {
			return errors.Wrapf(err, "creating %q bucketName", bucketName)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "starting BoltDB")
	}
	// todo: close BoltDB when server is sigtermed
	return &Backend{db, []byte(bucketName)}, nil
}

// NewWithDB is used for testing
func NewWithDB(db *bolt.DB, bucket string) (*Backend, error) {
	return &Backend{db, []byte(bucket)}, nil
}

func (b *Backend) TryLock(newLock models.ProjectLock) (bool, models.ProjectLock, error) {
	// return variables
	var lockAcquired bool
	var currLock models.ProjectLock
	key := b.key(newLock.Project, newLock.Env)
	newLockSerialized, _ := json.Marshal(newLock)
	transactionErr := b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)

		// if there is no run at that key then we're free to create the lock
		currLockSerialized := bucket.Get([]byte(key))
		if currLockSerialized == nil {
			bucket.Put([]byte(key), newLockSerialized) // not a readonly bucketName so okay to ignore error
			lockAcquired = true
			currLock = newLock
			return nil
		}

		// otherwise the lock fails, return to caller the run that's holding the lock
		if err := json.Unmarshal(currLockSerialized, &currLock); err != nil {
			return errors.Wrap(err, "failed to deserialize current lock")
		}
		lockAcquired = false
		return nil
	})

	if transactionErr != nil {
		return false, currLock, errors.Wrap(transactionErr, "DB transaction failed")
	}

	return lockAcquired, currLock, nil
}

func (b Backend) Unlock(p models.Project, env string) (*models.ProjectLock, error) {
	var lock models.ProjectLock
	foundLock := false
	key := b.key(p, env)
	err := b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		serialized := bucket.Get([]byte(key))
		if serialized != nil {
			if err := json.Unmarshal(serialized, &lock); err != nil {
				return errors.Wrap(err, "failed to deserialize lock")
			}
			foundLock = true
		}
		return bucket.Delete([]byte(key))
	})
	err = errors.Wrap(err, "DB transaction failed")
	if foundLock {
		return &lock, err
	}
	return nil, err
}

func (b Backend) List() ([]models.ProjectLock, error) {
	var locks []models.ProjectLock
	var locksBytes [][]byte
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			locksBytes = append(locksBytes, v)
		}
		return nil
	})
	if err != nil {
		return locks, errors.Wrap(err, "DB transaction failed")
	}

	// deserialize bytes into the proper objects
	for k, v := range locksBytes {
		var lock models.ProjectLock
		if err := json.Unmarshal(v, &lock); err != nil {
			return locks, errors.Wrap(err, fmt.Sprintf("failed to deserialize lock at key %q", string(k)))
		}
		locks = append(locks, lock)
	}

	return locks, nil
}

func (b Backend) UnlockByPull(repoFullName string, pullNum int) ([]models.ProjectLock, error) {
	var locks []models.ProjectLock
	err := b.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(b.bucket).Cursor()

		// we can use the repoFullName as a prefix search since that's the first part of the key
		for k, v := c.Seek([]byte(repoFullName)); k != nil && bytes.HasPrefix(k, []byte(repoFullName)); k, v = c.Next() {
			var lock models.ProjectLock
			if err := json.Unmarshal(v, &lock); err != nil {
				return errors.Wrapf(err, "failed to deserialize lock at key %q", string(k))
			}
			if lock.Pull.Num == pullNum {
				locks = append(locks, lock)
			}
		}
		return nil
	})

	// delete the locks
	for _, lock := range locks {
		if _, err = b.Unlock(lock.Project, lock.Env); err != nil {
			return locks, errors.Wrapf(err, "unlocking repo %s, path %s, env %s", lock.Project.RepoFullName, lock.Project.Path, lock.Env)
		}
	}
	return locks, nil
}

func (b Backend) key(p models.Project, env string) string {
	return fmt.Sprintf("%s/%s/%s", p.RepoFullName, p.Path, env)
}

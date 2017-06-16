package boltdb_test

import (
	"github.com/hootsuite/atlantis/locking"
	. "github.com/hootsuite/atlantis/testing_util"

	"github.com/boltdb/bolt"
	"os"
	"testing"
	"io/ioutil"
	"github.com/pkg/errors"

	"github.com/hootsuite/atlantis/locking/boltdb"
	"github.com/hootsuite/atlantis/models"
)

var lockBucket = "bucket"
var repo = "owner/repo"
var path = "/parent/child"
var env = "default"
var pullNum = 1

func TestListNoLocks(t *testing.T) {
	t.Log("listing locks when there are none should return an empty list")
	db, b := newTestDB()
	defer cleanupDB(db)
	ls, err := b.List()
	Ok(t, err)
	Equals(t, 0, len(ls))
}

func TestListOneLock(t *testing.T) {
	t.Log("listing locks when there is one should return it")
	db, b := newTestDB()
	defer cleanupDB(db)
	_, _, err := b.TryLock(repo, path, env, pullNum)
	Ok(t, err)
	ls, err := b.List()
	Ok(t, err)
	Equals(t, 1, len(ls))
}

func TestListMultipleLocks(t *testing.T) {
	t.Log("listing locks when there are multiple should return them")
	db, b := newTestDB()
	defer cleanupDB(db)

	// add multiple locks
	repos := []string{
		"owner/repo1",
		"owner/repo2",
		"owner/repo3",
		"owner/repo4",
	}

	for _, r := range repos {
		_, _, err := b.TryLock(r, path, env, pullNum)
		Ok(t, err)
	}
	ls, err := b.List()
	Ok(t, err)
	Equals(t, 4, len(ls))
	for _, r := range repos {
		found := false
		for _, l := range ls {
			if l.Project.RepoFullName == r {
				found = true
			}
		}
		Assert(t, found == true, "expected %s in %v", r, ls)
	}
}

func TestListAddRemove(t *testing.T) {
	t.Log("listing after adding and removing should return none")
	db, b := newTestDB()
	defer cleanupDB(db)
	_, _, err := b.TryLock(repo, path, env, pullNum)
	Ok(t, err)
	b.Unlock(repo, path, env)

	ls, err := b.List()
	Ok(t, err)
	Equals(t, 0, len(ls))
}

func TestLockingNoLocks(t *testing.T) {
	t.Log("with no locks yet, lock should succeed")
	db, b := newTestDB()
	defer cleanupDB(db)
	acquired, curr, err := b.TryLock(repo, path, env, pullNum)
	Ok(t, err)
	Equals(t, true, acquired)
	Equals(t, project, r.CurrentLock)
	Equals(t, project.StateKey(), r.LockKey)
}

func TestLockingExistingLock(t *testing.T) {
	t.Log("if there is an existing lock, lock should...")
	db, b := newTestDB()
	defer cleanupDB(db)
	_, _, err := b.TryLock(repo, path, env, pullNum)
	Ok(t, err)

	t.Log("...succeed if the new project has a different path")
	{
		new := project
		new.Path = "different/path"
		r, err := b.TryLock(repo, path, env, pullNum)
		Ok(t, err)
		Equals(t, true, r.LockAcquired)
		Equals(t, new, r.CurrentLock)
		Equals(t, new.StateKey(), r.LockKey)
	}

	t.Log("...succeed if the new project has a different environment")
	{
		new := project
		new.Env = "different-env"
		r, err := b.TryLock(repo, path, env, pullNum)
		Ok(t, err)
		Equals(t, true, r.LockAcquired)
		Equals(t, new, r.CurrentLock)
		Equals(t, new.StateKey(), r.LockKey)
	}

	t.Log("...succeed if the new project has a different repoName")
	{
		new := project
		new.RepoFullName = "new/repo"
		r, err := b.TryLock(repo, path, env, pullNum)
		Ok(t, err)
		Equals(t, true, r.LockAcquired)
		Equals(t, new, r.CurrentLock)
		Equals(t, new.StateKey(), r.LockKey)
	}

	t.Log("...not succeed if the new project only has a different pullNum")
	{
		new := project
		new.PullNum = project.PullNum + 1
		r, err := b.TryLock(repo, path, env, pullNum)
		Ok(t, err)
		Equals(t, false, r.LockAcquired)
		Equals(t, project, r.CurrentLock)
		Equals(t, project.StateKey(), r.LockKey)
	}
}

func TestUnlockingNoLocks(t *testing.T) {
	t.Log("unlocking with no locks should succeed")
	db, b := newTestDB()
	defer cleanupDB(db)

	Ok(t, b.Unlock("any-lock-id"))
}

func TestUnlocking(t *testing.T) {
	t.Log("unlocking with an existing lock should succeed")
	db, b := newTestDB()
	defer cleanupDB(db)

	b.TryLock(repo, path, env, pullNum)
	Ok(t, b.Unlock(project.StateKey()))

	// should be no locks listed
	ls, err := b.List()
	Ok(t, err)
	Equals(t, 0, len(ls))

	// should be able to re-lock that repo with a new pull num
	new := project
	new.PullNum = project.PullNum + 1
	r, err := b.TryLock(repo, path, env, pullNum)
	Ok(t, err)
	Equals(t, true, r.LockAcquired)
}

func TestUnlockingMultiple(t *testing.T) {
	t.Log("unlocking and locking multiple locks should succeed")
	db, b := newTestDB()
	defer cleanupDB(db)

	b.TryLock(repo, path, env, pullNum)

	new := project
	new.RepoFullName = "new/repo"
	b.TryLock(repo, path, env, pullNum)

	new2 := project
	new2.Path = "new/path"
	b.TryLock(repo, path, env, pullNum)

	new3 := project
	new3.Env = "new/env"
	b.TryLock(repo, path, env, pullNum)

	// now try and unlock them
	Ok(t, b.Unlock(new3.StateKey()))
	Ok(t, b.Unlock(new2.StateKey()))
	Ok(t, b.Unlock(new.StateKey()))
	Ok(t, b.Unlock(project.StateKey()))

	// should be none left
	ls, err := b.List()
	Ok(t, err)
	Equals(t, 0, len(ls))
}

func TestFindLocksNone(t *testing.T) {
	t.Log("find should return no locks when there are none")
	db, b := newTestDB()
	defer cleanupDB(db)

	ls, err := b.FindLocksForPull("any/repo", 1)
	Ok(t, err)
	Equals(t, 0, len(ls))
}

func TestFindLocksOne(t *testing.T) {
	t.Log("with one lock find should...")
	db, b := newTestDB()
	defer cleanupDB(db)
	_, _, err := b.TryLock(repo, path, env, pullNum)
	Ok(t, err)

	t.Log("...return no locks from the same repo but different pull num")
	{
		ls, err := b.FindLocksForPull(project.RepoFullName, project.PullNum + 1)
		Ok(t, err)
		Equals(t, 0, len(ls))
	}
	t.Log("...return no locks from a different repo but the same pull num")
	{
		ls, err := b.FindLocksForPull(project.RepoFullName + "dif", project.PullNum)
		Ok(t, err)
		Equals(t, 0, len(ls))
	}
	t.Log("...return the one lock when called with that repo and pull num")
	{
		ls, err := b.FindLocksForPull(project.RepoFullName, project.PullNum)
		Ok(t, err)
		Equals(t, 1, len(ls))
		Equals(t, project.StateKey(), ls[0])
	}
}

func TestFindLocksAfterUnlock(t *testing.T) {
	t.Log("after locking and unlocking find should return no locks")
	db, b := newTestDB()
	defer cleanupDB(db)
	_, _, err := b.TryLock(repo, path, env, pullNum)
	Ok(t, err)
	Ok(t, b.Unlock(project.StateKey()))

	ls, err := b.FindLocksForPull(project.RepoFullName, project.PullNum)
	Ok(t, err)
	Equals(t, 0, len(ls))
}

func TestFindMultipleMatching(t *testing.T) {
	t.Log("find should return all matching lock ids")
	db, b := newTestDB()
	defer cleanupDB(db)
	_, _, err := b.TryLock(repo, path, env, pullNum)
	Ok(t, err)

	// add additional locks with the same repo and pull num but different paths/envs
	new := project
	new.Path = "dif/path"
	_, err = b.TryLock(repo, path, env, pullNum)
	Ok(t, err)
	new2 := project
	new2.Env = "new-env"
	_, err = b.TryLock(repo, path, env, pullNum)
	Ok(t, err)

	// should get all of them back
	ls, err := b.FindLocksForPull(project.RepoFullName, project.PullNum)
	Ok(t, err)
	Equals(t, 3, len(ls))
	Contains(t, project.StateKey(), ls)
	Contains(t, new.StateKey(), ls)
	Contains(t, new2.StateKey(), ls)
}

// newTestDB returns a TestDB using a temporary path.
func newTestDB() (*bolt.DB, *boltdb.Backend) {
	// Retrieve a temporary path.
	f, err := ioutil.TempFile("", "")
	if err != nil {
		panic(errors.Wrap(err, "failed to create temp file"))
	}
	path := f.Name()
	f.Close()

	// Open the database.
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		panic(errors.Wrap(err, "could not start bolt DB"))
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(lockBucket)); err != nil {
			return errors.Wrap(err, "failed to create bucket")
		}
		return nil
	}); err != nil {
		panic(errors.Wrap(err, "could not create bucket"))
	}
	b, _ := boltdb.NewWithDB(db, lockBucket)
	return db, b
}

func cleanupDB(db *bolt.DB) {
	os.Remove(db.Path())
	db.Close()
}


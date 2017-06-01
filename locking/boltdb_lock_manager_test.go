package locking_test

import (
	. "github.com/hootsuite/atlantis/locking"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/boltdb/bolt"
	"os"
	"time"
)

var _ = Describe("BoltDBLockManager", func() {
	var db *bolt.DB
	var locker *BoltDBLockManager
	var run Run
	lockBucket := "locks"
	BeforeEach(func() {
		db = NewTestDB()
		locker = NewBoltDBLockManager(db, lockBucket)
		run = Run{
			RepoFullName: "owner/repo",
			Path: "parent/child",
			Env: "default",
			PullNum: 1,
			User: "user",
			Timestamp: time.Now(),
		}
	})
	AfterEach(func() {
		os.Remove(db.Path())
		db.Close()
	})
	var addLock = func(locker *BoltDBLockManager, run Run) {
		res, err := locker.TryLock(run)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.LockAcquired).To(BeTrue())
		Expect(res.LockingRun).To(Equal(run))
	}

	Describe("listing locks", func() {
		Context("with no locks", func() {
			It("should return 0", func() {
				Expect(locker.ListLocks()).To(BeEmpty())
			})
		})
		Context("with one lock", func() {
			BeforeEach(func() {
				addLock(locker, run)
			})
			It("should return it", func() {
				Expect(locker.ListLocks()).To(HaveLen(1))
				Expect(locker.ListLocks()).To(Equal(map[string]Run{
					run.StateKey(): run,
				}))
			})
		})
		Context("with two locks", func() {
			var run2 Run
			BeforeEach(func() {
				addLock(locker, run)
				run2 = run
				run2.RepoFullName = "new/repo"
				addLock(locker, run2)
			})
			It("should return them", func() {
				Expect(locker.ListLocks()).To(HaveLen(2))
				Expect(locker.ListLocks()).To(HaveKeyWithValue(run.StateKey(), run))
				Expect(locker.ListLocks()).To(HaveKeyWithValue(run2.StateKey(), run2))
			})
		})
		Context("with one lock added and then unlocked", func() {
			BeforeEach(func() {
				addLock(locker, run)
				err := locker.Unlock(run.StateKey())
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return 0", func() {
				Expect(locker.ListLocks()).To(BeEmpty())
			})
		})
	})

	Describe("locking", func() {
		Context("with no current locks", func() {
			It("should succeed", func() {
				Expect(locker.TryLock(run)).To(Equal(TryLockResponse{
					LockAcquired: true,
					LockingRun: run,
					LockID: run.StateKey(),
				}))
			})
		})
		Context("with an existing lock", func() {
			BeforeEach(func() {
				addLock(locker, run)
			})
			It("should succeed if the new run has a different path", func() {
				newRun := run
				newRun.Path = "different/path"
				Expect(locker.TryLock(newRun)).To(Equal(TryLockResponse{
					LockAcquired: true,
					LockingRun: newRun,
					LockID: newRun.StateKey(),
				}))
			})
			It("should succeed if the new run has a different environment", func() {
				newRun := run
				newRun.Env = "different-env"
				Expect(locker.TryLock(newRun)).To(Equal(TryLockResponse{
					LockAcquired: true,
					LockingRun: newRun,
					LockID: newRun.StateKey(),
				}))
			})
			It("should succeed if the new run has a different repoName", func() {
				newRun := run
				newRun.RepoFullName = "new/repo"
				Expect(locker.TryLock(newRun)).To(Equal(TryLockResponse{
					LockAcquired: true,
					LockingRun: newRun,
					LockID: newRun.StateKey(),
				}))
			})
			It("should not succeed if the new run only has a different pullNum and return the locking run", func() {
				newRun := run
				newRun.PullNum = 2
				Expect(locker.TryLock(newRun)).To(Equal(TryLockResponse{
					LockAcquired: false,
					LockingRun: run,
					LockID: run.StateKey(),
				}))
			})
		})
	})

	Describe("unlocking", func() {
		Context("with no locks", func() {
			It("should unlock a key that doesn't exist", func() {
				Expect(locker.Unlock("any-lock-id")).To(Succeed())
			})
			It("should unlock a key that doesn't exist even if it's an empty string", func() {
				Expect(locker.Unlock("")).To(Succeed())
			})
		})
		Context("with an existing lock", func() {
			BeforeEach(func() {
				addLock(locker, run)
			})
			It("should unlock a key that doesn't exist", func() {
				Expect(locker.Unlock("any-lock-id")).To(Succeed())
			})
			It("should unlock the run successfully", func() {
				Expect(locker.Unlock(run.StateKey())).To(Succeed())
				Expect(locker.ListLocks()).To(BeEmpty())
			})
		})
	})

	Describe("FindLocksForPull", func() {
		Context("with no locks", func() {
			It("should return no locks", func() {
				Expect(locker.FindLocksForPull("owner/repo", 1)).To(BeEmpty())
			})
		})
		Context("with one lock", func() {
			BeforeEach(func() {
				addLock(locker, run)
			})
			It("should return no locks from the same repo but different pull num", func() {
				Expect(locker.FindLocksForPull(run.RepoFullName, run.PullNum + 1)).To(BeEmpty())
			})
			It("should return no locks from a different repo but the same pull num", func() {
				Expect(locker.FindLocksForPull(run.RepoFullName + "dif", run.PullNum)).To(BeEmpty())
			})
			It("should return the one lock when called with that repo and pull num", func () {
				Expect(locker.FindLocksForPull(run.RepoFullName, run.PullNum)).To(Equal([]string{run.StateKey()}))
			})
		})
		Context("with one lock added and then unlocked", func() {
			BeforeEach(func() {
				addLock(locker, run)
				err := locker.Unlock(run.StateKey())
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return 0 locks from that repo and pull num", func () {
				Expect(locker.FindLocksForPull(run.RepoFullName, run.PullNum)).To(BeEmpty())
			})
		})
		Context("with two locks from the different repos but same pull nums", func() {
			var run2 Run
			BeforeEach(func() {
				addLock(locker, run)
				run2 = run
				run2.RepoFullName = run.RepoFullName + "dif"
				addLock(locker, run2)
			})
			It("should return the lock from the matching repo and pull num and nothing else", func() {
				Expect(locker.FindLocksForPull(run.RepoFullName, run.PullNum)).To(Equal([]string{run.StateKey()}))
				Expect(locker.FindLocksForPull(run2.RepoFullName, run2.PullNum)).To(Equal([]string{run2.StateKey()}))
			})
		})
		Context("with two locks from the same repos and pull num but with different paths", func() {
			var run2 Run
			BeforeEach(func() {
				addLock(locker, run)
				run2 = run
				run2.Path = run.Path + "/child"
				addLock(locker, run2)
			})
			It("should return both runs", func() {
				Expect(locker.FindLocksForPull(run.RepoFullName, run.PullNum)).To(HaveLen(2))
				Expect(locker.FindLocksForPull(run.RepoFullName, run.PullNum)).To(ContainElement(run.StateKey()))
				Expect(locker.FindLocksForPull(run.RepoFullName, run.PullNum)).To(ContainElement(run2.StateKey()))
			})
		})
		Context("with two locks from the same repos and pull num but with different envs", func() {
			var run2 Run
			BeforeEach(func() {
				addLock(locker, run)
				run2 = run
				run2.Env = run.Env + "dif"
				addLock(locker, run2)
			})
			It("should return both runs", func() {
				Expect(locker.FindLocksForPull(run.RepoFullName, run.PullNum)).To(HaveLen(2))
				Expect(locker.FindLocksForPull(run.RepoFullName, run.PullNum)).To(ContainElement(run.StateKey()))
				Expect(locker.FindLocksForPull(run.RepoFullName, run.PullNum)).To(ContainElement(run2.StateKey()))
			})
		})
	})
})

// NewTestDB returns a TestDB using a temporary path.
func NewTestDB() *bolt.DB {
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
		if _, err := tx.CreateBucketIfNotExists([]byte(LockBucket)); err != nil {
			return errors.Wrap(err, "failed to create bucket")
		}
		return nil
	}); err != nil {
		panic(errors.Wrap(err, "could not create bucket"))
	}
	return db
}

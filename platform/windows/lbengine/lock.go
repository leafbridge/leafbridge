package lbengine

import (
	"fmt"

	"github.com/gentlemanautomaton/winobj/winmutex"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/internal/reentrantlock"
)

// lockManager is responsible for acquiring locks on system-wide resources.
type lockManager struct {
	locks map[lbdeploy.LockID]Lock
}

// newLockManager prepares a lock manager.
func newLockManager() *lockManager {
	return &lockManager{
		locks: make(map[lbdeploy.LockID]Lock),
	}
}

// Create attempts to create all of the requested locks. If successful, it
// returns the locks as a group.
//
// If any of the requested locks already exist within the lock manager, the
// existing lock will be included in the group membership.
func (lm *lockManager) Create(resources lbdeploy.Resources, locks ...lbdeploy.LockID) (LockGroup, error) {
	var group LockGroup

	for _, id := range locks {
		if lock, exists := lm.locks[id]; exists {
			group.members = append(group.members, lock)
		} else {
			lock, err := createLock(resources, id)
			if err != nil {
				return LockGroup{}, err
			}
			lm.locks[id] = lock
			group.members = append(group.members, lock)
		}
	}

	return group, nil
}

// CloseAll attempts to release and close all locks currently held by the
// lock manager.
func (lm *lockManager) CloseAll() error {
	for id, lock := range lm.locks {
		lock.locker.Close()
		delete(lm.locks, id)
	}
	return nil
}

// createLock creates a reentrant locker for the given lock ID.
func createLock(resources lbdeploy.Resources, lock lbdeploy.LockID) (Lock, error) {
	// Find the lock with the deployment's resources and verify it.
	lockDefinition, found := resources.Locks[lock]
	if !found {
		return Lock{}, fmt.Errorf("the requested lock ID \"%s\" is not declared in the deployment's resources", lock)
	}
	mutex := lockDefinition.Mutex
	if mutex == "" {
		return Lock{}, fmt.Errorf("the \"%s\" lock does not identify a mutex that it locks", lock)
	}

	// Find the mutex with the deployment's resources and verify it.
	mutexDefinition, found := resources.Mutexes[mutex]
	if !found {
		return Lock{}, fmt.Errorf("the requested mutex ID \"%s\" is not declared in the deployment's resources", mutex)
	}
	if mutexDefinition.Name == "" {
		return Lock{}, fmt.Errorf("the \"%s\" mutex is missing mutex name", mutex)
	}

	// Determine the name of the mutex.
	mutexName, err := mutexDefinition.ObjectName()
	if err != nil {
		return Lock{}, err
	}

	// Create or open the mutex.
	m, err := winmutex.New(mutexName)
	if err != nil {
		return Lock{}, err
	}

	// Return a lock that includes a reentrant variant of the mutex.
	return Lock{
		id:     lock,
		def:    lockDefinition,
		locker: reentrantlock.Wrap(m),
	}, nil
}

// Lock is a lockable resource.
type Lock struct {
	id     lbdeploy.LockID
	def    lbdeploy.Lock
	locker reentrantlock.Locker
}

// LockError is an error returned when a lock cannot be acquired.
type LockError struct {
	LockID lbdeploy.LockID
	Lock   lbdeploy.Lock
}

// Error returns a string describing the error.
func (e LockError) Error() string {
	if e.Lock.ConflictRules.Message != "" {
		return fmt.Sprintf("failed to acquire \"%s\" lock: %s", e.LockID, e.Lock.ConflictRules.Message)
	}
	return fmt.Sprintf("failed to acquire \"%s\" lock", e.LockID)
}

// LockGroup facilitates locking and unlocking a group of lockable resources
// together.
type LockGroup struct {
	members []Lock
}

// Lock attempts to lock all entries in the group.
//
// If any member of the group fails to acquire its lock, all locks in the
// group are released and it returns an error of type LockError.
func (group LockGroup) Lock() error {
	for i, member := range group.members {
		if !member.locker.TryLock() {
			for j := i - 1; j >= 0; j-- {
				group.members[j].locker.Unlock()
			}
			return LockError{
				LockID: member.id,
				Lock:   member.def,
			}
		}
	}
	return nil
}

// Unlock unlocks all members of the lock group.
func (group LockGroup) Unlock() {
	for i := len(group.members) - 1; i >= 0; i-- {
		member := group.members[i]
		member.locker.Unlock()
	}
}

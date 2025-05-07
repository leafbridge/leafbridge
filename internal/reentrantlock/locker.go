package reentrantlock

// Locker is a lock interface capable of being turned into a reentry locker.
type Locker interface {
	Lock()
	TryLock() bool
	Unlock()
	Close() error
}

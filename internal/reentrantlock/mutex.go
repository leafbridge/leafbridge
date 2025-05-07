package reentrantlock

// Mutex is a reentrant version of a locker that wraps an underlying
// non-reentrant mutex.
type Mutex[T Locker] struct {
	counter int
	mutex   T
}

// New returns a reentrant version of the given mutex.
func Wrap[T Locker](mutex T) *Mutex[T] {
	return &Mutex[T]{mutex: mutex}
}

func (m *Mutex[T]) Lock() {
	if m.counter == 0 {
		m.mutex.Lock()
	}
	m.counter++
}

func (m *Mutex[T]) TryLock() bool {
	if m.counter == 0 {
		if !m.mutex.TryLock() {
			return false
		}
	}
	m.counter++
	return true
}

func (m *Mutex[T]) Unlock() {
	m.counter--
	if m.counter == 0 {
		m.mutex.Unlock()
	}
}

func (m *Mutex[T]) Close() error {
	return m.mutex.Close()
}

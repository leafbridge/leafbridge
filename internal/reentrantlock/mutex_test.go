package reentrantlock_test

import (
	"testing"

	"github.com/gentlemanautomaton/winobj/winmutex"
	"github.com/leafbridge/leafbridge/internal/reentrantlock"
)

func TestLock(t *testing.T) {
	mutex, err := winmutex.New("LeafBridge-ReentrantMutex-Test")
	if err != nil {
		t.Fatal(err)
	}
	lock := reentrantlock.Wrap(mutex)
	defer lock.Close()

	for range 64 {
		lock.Lock()
		defer lock.Unlock()
	}
}

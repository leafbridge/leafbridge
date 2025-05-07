package lbdeploy

import "fmt"

// MutexMap holds a set of mutex resources mapped by their identifiers.
type MutexMap map[MutexID]Mutex

// MutexID is a unique identifier for a system-wide mutex resource.
type MutexID string

// MutexName is the name of a system-wide mutex on the machine a
// deployment is running on.
type MutexName string

// MutexNamespace is the namespace within a mutex exists.
// It can be "leafbridge", "global" or "system".
type MutexNamespace string

// Mutex namespaces.
const (
	LeafBridgeMutex MutexNamespace = "leafbridge"
	GlobalMutex     MutexNamespace = "global"
	SessionMutex    MutexNamespace = "session"
)

// Mutex is a system-wide mutex that can be evaluated by conditions or used
// by locks.
type Mutex struct {
	Description string         `json:"description,omitempty"`
	Name        MutexName      `json:"name"`
	Namespace   MutexNamespace `json:"namespace"`
}

// ObjectName returns the name of the mutex object in the Windows Object
// Manager.
func (mutex Mutex) ObjectName() (string, error) {
	switch mutex.Namespace {
	case LeafBridgeMutex:
		// TODO: Consider using a private namespace.
		return fmt.Sprintf("Global\\LeafBridge-Deployment-%s", mutex.Name), nil
	case GlobalMutex:
		return fmt.Sprintf("Global\\%s", mutex.Name), nil
	case SessionMutex:
		return fmt.Sprintf("Session\\%s", mutex.Name), nil
	case "":
		return "", fmt.Errorf("the \"%s\" mutex is missing a mutex namespace", mutex)
	default:
		return "", fmt.Errorf("the \"%s\" mutex has an unrcognized namespace: %s", mutex, mutex.Namespace)
	}
}

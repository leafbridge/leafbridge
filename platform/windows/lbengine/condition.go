package lbengine

import (
	"fmt"
	"os"

	"github.com/gentlemanautomaton/winobj/winmutex"
	"github.com/leafbridge/leafbridge/core/idset"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbvalue"
	"github.com/leafbridge/leafbridge/platform/windows/localfs"
	"github.com/leafbridge/leafbridge/platform/windows/localregistry"
)

// conditionSet keeps track of a set of conditions as they are evaluated.
type conditionSet = idset.SetOf[lbdeploy.ConditionID]

// ConditionEngine is responsible for evaluating conditions on the local
// system.
type ConditionEngine struct {
	deployment lbdeploy.Deployment
}

// NewConditionEngine prepares a condition engine for the given deployment.
func NewConditionEngine(dep lbdeploy.Deployment) ConditionEngine {
	return ConditionEngine{
		deployment: dep,
	}
}

// Evaluate returns true if the given condition is currently true.
//
// TODO: Consider returning some sort of evaluation struct that describes
// the condition (or subconditions) that failed.
func (engine ConditionEngine) Evaluate(condition lbdeploy.ConditionID) (bool, error) {
	// Find the condition within the deployment.
	definition, found := engine.deployment.Conditions[condition]
	if !found {
		return false, fmt.Errorf("the condition \"%s\" does not exist within the \"%s\" deployment", condition, engine.deployment.ID)
	}

	return engine.evaluate(condition, definition, make(lbdeploy.ConditionCache), make(conditionSet))
}

func (engine ConditionEngine) evaluate(id lbdeploy.ConditionID, condition lbdeploy.Condition, cache lbdeploy.ConditionCache, seen conditionSet) (bool, error) {
	// Special handling for conditions that are identified.
	if id != "" {
		// If this condition has already been evaluated, return the cached value.
		if value, computed := cache[id]; computed {
			return value, nil
		}

		// Check for recursive calls.
		if seen.Contains(id) {
			return false, fmt.Errorf("the \"%s\" condition is recursive and is already being evaluated", id)
		}

		// Add this condition to evaluation set, then remove it when we're finished.
		seen.Add(id)
		defer seen.Remove(id)
	}

	// Evaluate the condition.
	result, err := func() (bool, error) {
		// Evaluate "any" conditions.
		if len(condition.Any) > 0 {
			for i, candidate := range condition.Any {
				result, err := engine.evaluate("", candidate, cache, seen)
				if err != nil {
					return false, lbdeploy.ConditionError{
						ID:           id,
						Label:        condition.Label,
						Type:         condition.Type,
						Element:      lbdeploy.ConditionElementAny,
						SubCondition: i,
						Err:          err,
					}
				}
				if result {
					return true, nil
				}
			}
			return false, nil
		}

		// Evaluate "all" conditions.
		if len(condition.All) > 0 {
			for i, candidate := range condition.All {
				result, err := engine.evaluate("", candidate, cache, seen)
				if err != nil {
					return false, lbdeploy.ConditionError{
						ID:           id,
						Label:        condition.Label,
						Type:         condition.Type,
						Element:      lbdeploy.ConditionElementAll,
						SubCondition: i,
						Err:          err,
					}
				}
				if !result {
					return false, nil
				}
			}
			return true, nil
		}

		// Evaluate individual conditions.
		switch condition.Type {
		case lbdeploy.ConditionTypeSubcondition:
			candidateID := lbdeploy.ConditionID(condition.Subject)
			candidate, found := engine.deployment.Conditions[candidateID]
			if !found {
				return false, conditionSelfError(id, condition, fmt.Errorf("the \"%s\" condition is not defined in the deployment", condition.Subject))
			}
			return engine.evaluate(candidateID, candidate, cache, seen)
		case lbdeploy.ConditionTypeProcessIsRunning:
			process, found := engine.deployment.Resources.Processes[lbdeploy.ProcessResourceID(condition.Subject)]
			if !found {
				return false, conditionSelfError(id, condition, fmt.Errorf("the \"%s\" process is not defined in the deployment", condition.Subject))
			}
			running, err := NumberOfRunningProcesses(process.Match)
			if err != nil {
				return false, conditionSelfError(id, condition, err)
			}
			return running > 0, nil
		case lbdeploy.ConditionTypeMutexExists:
			mutex, found := engine.deployment.Resources.Mutexes[lbdeploy.MutexID(condition.Subject)]
			if !found {
				return false, conditionSelfError(id, condition, fmt.Errorf("the \"%s\" mutex is not defined in the deployment", condition.Subject))
			}
			name, err := mutex.ObjectName()
			if err != nil {
				return false, conditionSelfError(id, condition, err)
			}
			exists, err := winmutex.Exists(name)
			if err != nil {
				return false, conditionSelfError(id, condition, err)
			}
			return exists, nil
		case lbdeploy.ConditionTypeRegistryKeyExists:
			ref, err := engine.deployment.Resources.Registry.ResolveKey(lbdeploy.RegistryKeyResourceID(condition.Subject))
			if err != nil {
				return false, conditionSelfError(id, condition, err)
			}
			key, err := localregistry.OpenKey(ref)
			if err != nil {
				if os.IsNotExist(err) {
					return false, nil
				}
				return false, conditionSelfError(id, condition, err)
			}
			defer key.Close()
			return true, nil
		case lbdeploy.ConditionTypeRegistryValueExists, lbdeploy.ConditionTypeRegistryValueComparison:
			ref, err := engine.deployment.Resources.Registry.ResolveValue(lbdeploy.RegistryValueResourceID(condition.Subject))
			if err != nil {
				return false, conditionSelfError(id, condition, err)
			}
			key, err := localregistry.OpenKey(ref.Key())
			if err != nil {
				if os.IsNotExist(err) {
					return false, nil
				}
				return false, conditionSelfError(id, condition, err)
			}
			defer key.Close()
			switch condition.Type {
			case lbdeploy.ConditionTypeRegistryValueExists:
				return key.HasValue(ref.Name)
			case lbdeploy.ConditionTypeRegistryValueComparison:
				value, err := key.GetValue(ref.Name, ref.Type)
				if err != nil {
					return false, conditionSelfError(id, condition, err)
				}
				result, err := lbvalue.TryCompare(value, condition.Value)
				if err != nil {
					return false, conditionSelfError(id, condition, err)
				}
				return condition.Comparison.Evaluate(result), nil
			default:
				panic("unhandled condition type")
			}
		case lbdeploy.ConditionTypeDirectoryExists:
			ref, err := engine.deployment.Resources.FileSystem.ResolveDirectory(lbdeploy.DirectoryResourceID(condition.Subject))
			if err != nil {
				return false, conditionSelfError(id, condition, err)
			}
			dir, err := localfs.OpenDir(ref)
			if err != nil {
				if os.IsNotExist(err) {
					return false, nil
				}
				return false, conditionSelfError(id, condition, err)
			}
			defer dir.Close()
			return true, nil
		case lbdeploy.ConditionTypeFileExists:
			ref, err := engine.deployment.Resources.FileSystem.ResolveFile(lbdeploy.FileResourceID(condition.Subject))
			if err != nil {
				return false, conditionSelfError(id, condition, err)
			}
			dir, err := localfs.OpenDir(ref.Dir())
			if err != nil {
				if os.IsNotExist(err) {
					return false, nil
				}
				return false, conditionSelfError(id, condition, err)
			}
			defer dir.Close()
			fi, err := dir.System().Stat(ref.FilePath)
			if err != nil {
				return false, conditionSelfError(id, condition, err)
			}
			if fi.Mode().IsRegular() {
				return true, nil
			}
			path, err := ref.Path()
			if err != nil {
				return false, conditionSelfError(id, condition, fmt.Errorf("file \"%s\": the path exists but it is not a regular file", condition.Subject))
			}
			return false, conditionSelfError(id, condition, fmt.Errorf("file \"%s\": the \"%s\" path exists but it is not a regular file", condition.Subject, path))
		default:
			return false, conditionSelfError(id, condition, fmt.Errorf("unrecognized condition type: %s", condition.Type))
		}
	}()

	// Negate the result if requested.
	if condition.Negated {
		result = !result
	}

	// Record the result in the cache if possible.
	if id != "" && err == nil {
		cache[id] = result
	}

	return result, err
}

func conditionSelfError(id lbdeploy.ConditionID, c lbdeploy.Condition, err error) error {
	return lbdeploy.ConditionError{
		ID:      id,
		Label:   c.Label,
		Type:    c.Type,
		Element: lbdeploy.ConditionElementSelf,
		Err:     err,
	}
}

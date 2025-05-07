package lbdeploy

import (
	"errors"
	"fmt"
	"strings"
)

// DeploymentID is a unique identifier for a deployment.
type DeploymentID string

// Validate returns a non-nil error if the deployment ID is invalid.
func (id DeploymentID) Validate() error {
	if id == "" {
		return errors.New("a deployment ID is missing")
	}
	return nil
}

// Deployment defines a deployment package.
type Deployment struct {
	ID         DeploymentID `json:"id,omitempty"`
	Name       string       `json:"name,omitempty"`
	Behavior   Behavior     `json:"behavior,omitzero"`
	Apps       AppMap       `json:"apps,omitzero"`
	Conditions ConditionMap `json:"conditions,omitzero"`
	Commands   CommandMap   `json:"commands,omitzero"`
	Resources  Resources    `json:"resources,omitzero"`
	Flows      FlowMap      `json:"flows,omitzero"`
}

// Validate returns an error if the deployment contains invalid configuration.
func (dep Deployment) Validate() error {
	if err := dep.ID.Validate(); err != nil {
		return err
	}

	for id := range dep.Conditions {
		if err := dep.ValidateCondition(id); err != nil {
			return err
		}
	}

	return nil
}

// ValidateCondition returns an error if the given condition is not valid.
func (dep Deployment) ValidateCondition(condition ConditionID) error {
	definition, found := dep.Conditions[condition]
	if !found {
		return fmt.Errorf("the condition \"%s\" does not exist within the \"%s\" deployment", condition, dep.ID)
	}

	if err := dep.validateCondition(definition); err != nil {
		return fmt.Errorf("the \"%s\" condition or one of its subconditions is not valid: %w", condition, err)
	}

	return nil
}

func (dep Deployment) validateCondition(condition Condition) error {
	var (
		hasType = condition.Type != ""
		hasAny  = len(condition.Any) > 0
		hasAll  = len(condition.All) > 0
	)

	fields := make([]string, 0, 3)
	if hasType {
		fields = append(fields, "type")
	}
	if hasAny {
		fields = append(fields, "any")
	}
	if hasAll {
		fields = append(fields, "all")
	}

	switch len(fields) {
	case 0:
		return conditionSelfError(condition, errors.New("the condition does not specify a type"))
	case 1:
	default:
		return conditionSelfError(condition, fmt.Errorf("the following fields are present, which are mutually exclusive: %s", strings.Join(fields, ", ")))
	}

	// Validate "any" conditions.
	for i, subcondition := range condition.Any {
		if err := dep.validateCondition(subcondition); err != nil {
			return ConditionError{
				Label:        condition.Label,
				Type:         condition.Type,
				Element:      ConditionElementAny,
				SubCondition: i,
				Err:          err,
			}
		}
	}

	// Validate "all" conditions.
	for i, subcondition := range condition.All {
		if err := dep.validateCondition(subcondition); err != nil {
			return ConditionError{
				Label:        condition.Label,
				Type:         condition.Type,
				Element:      ConditionElementAll,
				SubCondition: i,
				Err:          err,
			}
		}
	}

	if !hasType {
		return nil
	}

	// Validate the condition based on its type.
	err := func() error {
		switch condition.Type {
		case ConditionTypeSubcondition:
			if condition.Subject == "" {
				return errors.New("the condition does not provide a condition ID")
			}
			if _, found := dep.Conditions[ConditionID(condition.Subject)]; !found {
				return fmt.Errorf("the condition references a condition ID that is not defined: %s", condition.Subject)
			}
			// TODO: Check for recursive conditions?
		case ConditionTypeProcessIsRunning:
			if condition.Subject == "" {
				return errors.New("the condition does not provide a process resource ID")
			}
			if _, found := dep.Resources.Processes[ProcessResourceID(condition.Subject)]; !found {
				return fmt.Errorf("the condition references a process resource ID that is not defined: %s", condition.Subject)
			}
		case ConditionTypeMutexExists:
			if condition.Subject == "" {
				return errors.New("the condition does not provide a mutex resource ID")
			}
			if _, found := dep.Resources.Mutexes[MutexID(condition.Subject)]; !found {
				return fmt.Errorf("the condition references a mutex resource ID that is not defined: %s", condition.Subject)
			}
		case ConditionTypeRegistryKeyExists:
			if condition.Subject == "" {
				return errors.New("the condition does not provide a registry key resource ID")
			}
			if _, found := dep.Resources.Registry.Keys[RegistryKeyResourceID(condition.Subject)]; !found {
				return fmt.Errorf("the condition references a registry key resource ID that is not defined: %s", condition.Subject)
			}
		case ConditionTypeRegistryValueExists, ConditionTypeRegistryValueComparison:
			if condition.Subject == "" {
				return errors.New("the condition does not provide a registry value resource ID")
			}
			if _, found := dep.Resources.Registry.Values[RegistryValueResourceID(condition.Subject)]; !found {
				return fmt.Errorf("the condition references a registry value resource ID that is not defined: %s", condition.Subject)
			}
		case ConditionTypeDirectoryExists:
			if condition.Subject == "" {
				return errors.New("the condition does not provide a directory resource ID")
			}
			if _, found := dep.Resources.FileSystem.Directories[DirectoryResourceID(condition.Subject)]; !found {
				return fmt.Errorf("the condition references a directory resource ID that is not defined: %s", condition.Subject)
			}
		case ConditionTypeFileExists:
			if condition.Subject == "" {
				return errors.New("the condition does not provide a file resource ID")
			}
			if _, found := dep.Resources.FileSystem.Files[FileResourceID(condition.Subject)]; !found {
				return fmt.Errorf("the condition references a file resource ID that is not defined: %s", condition.Subject)
			}
		default:
			return fmt.Errorf("the condition type is not recognized: %s", condition.Type)
		}
		return nil
	}()

	if err != nil {
		return conditionSelfError(condition, err)
	}

	return nil
}

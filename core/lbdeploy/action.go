package lbdeploy

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ActionList is a list of deployment actions.
type ActionList []Action

// UnmarshalJSON attempts to unmarshal the given JSON data into a list of
// actions of various underlying types.
func (list *ActionList) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal deployment action list: %w", err)
	}

	output := make(ActionList, len(raw))
	for i, item := range raw {
		action, err := UnmarshalActionJSON(item)
		if err != nil {
			return fmt.Errorf("failed to unmarshal deployment action list: element %d: %w", i, err)
		}
		output[i] = action
	}

	*list = output

	return nil
}

// ActionType identifies the type of action.
type ActionType string

// Recognized action types.
const (
	ActionTypeStartFlow            ActionType = "start-flow"
	ActionTypePreparePackage       ActionType = "prepare-package"
	ActionTypeInvokeCommand        ActionType = "invoke-command"
	ActionTypeCopyPackageFile      ActionType = "copy-package-file"
	ActionTypeCopyPackageDirectory ActionType = "copy-package-directory"
	ActionTypeCopyFile             ActionType = "copy-file"
	ActionTypeCopyDirectory        ActionType = "copy-directory"
	ActionTypeDeleteFile           ActionType = "delete-file"
	ActionTypeDeleteDirectory      ActionType = "delete-directory"
)

// Action is a common interface implemented by all LeafBridge deployment
// actions.
type Action interface {
	Type() ActionType
}

// UnmarshalJSON attempts to unmarshal the given JSON data into a deployment
// action of varying types.
func UnmarshalActionJSON(b []byte) (action Action, err error) {
	var header struct {
		Type ActionType `json:"action"`
	}
	if err := json.Unmarshal(b, &header); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment action header: %w", err)
	}

	switch header.Type {
	case ActionTypeStartFlow:
		var data StartFlowAction
		err = json.Unmarshal(b, &data)
		action = data
	case ActionTypePreparePackage:
		var data PreparePackageAction
		err = json.Unmarshal(b, &data)
		action = data
	case ActionTypeInvokeCommand:
		var data InvokeCommandAction
		err = json.Unmarshal(b, &data)
		action = data
	case ActionTypeCopyPackageFile:
		var data CopyPackageFileAction
		err = json.Unmarshal(b, &data)
		action = data
	case ActionTypeCopyPackageDirectory:
		var data CopyPackageDirectoryAction
		err = json.Unmarshal(b, &data)
		action = data
	case ActionTypeCopyFile:
		var data CopyFileAction
		err = json.Unmarshal(b, &data)
		action = data
	case ActionTypeCopyDirectory:
		var data CopyDirectoryAction
		err = json.Unmarshal(b, &data)
		action = data
	case ActionTypeDeleteFile:
		var data DeleteFileAction
		err = json.Unmarshal(b, &data)
		action = data
	case ActionTypeDeleteDirectory:
		var data DeleteDirectoryAction
		err = json.Unmarshal(b, &data)
		action = data
	case "":
		err = errors.New("missing action type")
	default:
		err = errors.New("unrecognized action type")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment action of type \"%s\": %w", header.Type, err)
	}

	return action, nil
}

// StartFlowAction is an action that starts another flow within a deployment.
type StartFlowAction struct {
	Flow FlowID `json:"flow"`
}

// Type returns the type of the action.
func (StartFlowAction) Type() ActionType {
	return ActionTypeStartFlow
}

// MarshalJSON marshals the action as JSON data.
func (action StartFlowAction) MarshalJSON() ([]byte, error) {
	type Action StartFlowAction
	return json.Marshal(struct {
		Type ActionType `json:"action"`
		Action
	}{
		Type:   action.Type(),
		Action: Action(action),
	})
}

// PreparePackageAction is an action that prepares a package for future
// use.
type PreparePackageAction struct {
	Package PackageID `json:"package"`
}

// Type returns the type of the action.
func (PreparePackageAction) Type() ActionType {
	return ActionTypePreparePackage
}

// MarshalJSON marshals the action as JSON data.
func (action PreparePackageAction) MarshalJSON() ([]byte, error) {
	type Action PreparePackageAction
	return json.Marshal(struct {
		Type ActionType `json:"action"`
		Action
	}{
		Type:   action.Type(),
		Action: Action(action),
	})
}

// InvokeCommandAction is an action that invokes a command.
type InvokeCommandAction struct {
	Package PackageID   `json:"package,omitempty"`
	Command CommandID   `json:"command"`
	Mode    CommandMode `json:"mode,omitempty"`
	Force   bool        `json:"force,omitempty"`
}

// Type returns the type of the action.
func (InvokeCommandAction) Type() ActionType {
	return ActionTypeInvokeCommand
}

// MarshalJSON marshals the action as JSON data.
func (action InvokeCommandAction) MarshalJSON() ([]byte, error) {
	type Action InvokeCommandAction
	return json.Marshal(struct {
		Type ActionType `json:"action"`
		Action
	}{
		Type:   action.Type(),
		Action: Action(action),
	})
}

// CopyPackageFileAction is an action that copies a package file.
type CopyPackageFileAction struct {
	Package         PackageID      `json:"package"`
	SourceFile      PackageFileID  `json:"source-file,omitempty"`
	DestinationFile FileResourceID `json:"destination-file"`
}

// Type returns the type of the action.
func (CopyPackageFileAction) Type() ActionType {
	return ActionTypeCopyPackageFile
}

// SourceName returns a name describing the source in one of two forms,
// depending on whether a source file has been specified in the action:
//
//  {package}
//  {package}.{source-file}
func (action CopyPackageFileAction) SourceName() string {
	if action.SourceFile == "" {
		return string(action.Package)
	}
	return fmt.Sprintf("%s.%s", action.Package, action.SourceFile)
}

// MarshalJSON marshals the action as JSON data.
func (action CopyPackageFileAction) MarshalJSON() ([]byte, error) {
	type Action CopyPackageFileAction
	return json.Marshal(struct {
		Type ActionType `json:"action"`
		Action
	}{
		Type:   action.Type(),
		Action: Action(action),
	})
}

// CopyPackageDirectoryAction is an action that copies the content of a
// package to a directory.
type CopyPackageDirectoryAction struct {
	Package        PackageID           `json:"package"`
	DestinationDir DirectoryResourceID `json:"destination-directory"`
}

// Type returns the type of the action.
func (CopyPackageDirectoryAction) Type() ActionType {
	return ActionTypeCopyPackageDirectory
}

// MarshalJSON marshals the action as JSON data.
func (action CopyPackageDirectoryAction) MarshalJSON() ([]byte, error) {
	type Action CopyPackageDirectoryAction
	return json.Marshal(struct {
		Type ActionType `json:"action"`
		Action
	}{
		Type:   action.Type(),
		Action: Action(action),
	})
}

// CopyFileAction is an action that copies a file.
type CopyFileAction struct {
	SourceFile      FileResourceID `json:"source-file"`
	DestinationFile FileResourceID `json:"destination-file"`
}

// Type returns the type of the action.
func (CopyFileAction) Type() ActionType {
	return ActionTypeCopyFile
}

// MarshalJSON marshals the action as JSON data.
func (action CopyFileAction) MarshalJSON() ([]byte, error) {
	type Action CopyFileAction
	return json.Marshal(struct {
		Type ActionType `json:"action"`
		Action
	}{
		Type:   action.Type(),
		Action: Action(action),
	})
}

// CopyDirectoryAction is an action that copies a file.
type CopyDirectoryAction struct {
	SourceDir      DirectoryResourceID `json:"source-directory"`
	DestinationDir DirectoryResourceID `json:"destination-directory"`
}

// Type returns the type of the action.
func (CopyDirectoryAction) Type() ActionType {
	return ActionTypeCopyFile
}

// MarshalJSON marshals the action as JSON data.
func (action CopyDirectoryAction) MarshalJSON() ([]byte, error) {
	type Action CopyDirectoryAction
	return json.Marshal(struct {
		Type ActionType `json:"action"`
		Action
	}{
		Type:   action.Type(),
		Action: Action(action),
	})
}

// DeleteFileAction is an action that deletes a file.
type DeleteFileAction struct {
	File FileResourceID `json:"file"`
}

// Type returns the type of the action.
func (DeleteFileAction) Type() ActionType {
	return ActionTypeDeleteFile
}

// MarshalJSON marshals the action as JSON data.
func (action DeleteFileAction) MarshalJSON() ([]byte, error) {
	type Action DeleteFileAction
	return json.Marshal(struct {
		Type ActionType `json:"action"`
		Action
	}{
		Type:   action.Type(),
		Action: Action(action),
	})
}

// DeleteDirectoryAction is an action that deletes a file.
type DeleteDirectoryAction struct {
	Dir DirectoryResourceID `json:"directory"`
}

// Type returns the type of the action.
func (DeleteDirectoryAction) Type() ActionType {
	return ActionTypeDeleteDirectory
}

// MarshalJSON marshals the action as JSON data.
func (action DeleteDirectoryAction) MarshalJSON() ([]byte, error) {
	type Action DeleteDirectoryAction
	return json.Marshal(struct {
		Type ActionType `json:"action"`
		Action
	}{
		Type:   action.Type(),
		Action: Action(action),
	})
}

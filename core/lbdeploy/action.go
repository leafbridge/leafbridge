package lbdeploy

// ActionType identifies the type of action.
type ActionType string

// Recognized action types.
const (
	ActionStartFlow      ActionType = "start-flow"
	ActionPreparePackage ActionType = "prepare-package"
	ActionInvokeCommand  ActionType = "invoke-command"
	ActionCopyFile       ActionType = "copy-file"
	ActionDeleteFile     ActionType = "delete-file"
)

// Action describes an action to be taken as part of a flow.
type Action struct {
	Type            ActionType          `json:"action"`
	Package         PackageID           `json:"package,omitempty"`
	Command         CommandID           `json:"command,omitempty"`
	Force           bool                `json:"force,omitempty"`
	Flow            FlowID              `json:"flow,omitempty"`
	SourceFile      FileResourceID      `json:"source-file,omitempty"`
	SourceDir       DirectoryResourceID `json:"source-directory,omitempty"`
	DestinationFile FileResourceID      `json:"destination-file,omitempty"`
	DestinationDir  DirectoryResourceID `json:"destination-directory,omitempty"`
}

/*
// PreparePackageAction is an action that prepares a package for use
// in the future.
type PreparePackageAction struct {
	Package PackageID `json:"package"`
}

// InvokePackageAction is an action that invokes a command on a package.
type InvokePackageAction struct {
	Package PackageID `json:"package"`
}
*/

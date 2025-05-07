package lbdeploy

import (
	"fmt"
	"strconv"

	"github.com/gentlemanautomaton/structformat"
)

// CommandType identifies the type of a command.
type CommandType string

// Command types.
const (
	CommandTypeExe                     = "exe"
	CommandTypeMSIInstall              = "msi-install"
	CommandTypeMSIUpdate               = "msi-update"
	CommandTypeMSIUninstall            = "msi-uninstall"
	CommandTypeMSIUninstallProductCode = "msi-uninstall-product-code"
)

// IsAppBased returns true if the command applies to an application's product
// code, and not to a provided executable or installer file.
func (t CommandType) IsAppBased() bool {
	return t == CommandTypeMSIUninstallProductCode
}

// IsMSI returns true if the command invokes msiexec.
func (t CommandType) IsMSI() bool {
	switch t {
	case CommandTypeMSIInstall, CommandTypeMSIUpdate, CommandTypeMSIUninstall, CommandTypeMSIUninstallProductCode:
		return true
	default:
		return false
	}
}

// CommandMap defines a set of commands that can be issued, mapped by their
// identifiers.
type CommandMap map[CommandID]Command

// CommandID is a unique identifier for a command.
type CommandID string

// ExecutableID is either a FileResourceID or a PackageFileID, depending on
// whether the command is a regular command or a package command.
type ExecutableID string

// Command defines a command that can be invoked for a deployment or
// package.
//
// TODO: Support variable expansion when building arguments.
type Command struct {
	// Installs is a list of applications that the command installs.
	Installs AppList `json:"installs,omitzero"`

	// Uninstalls is a list of applicaitons that the command uninstalls.
	Uninstalls AppList `json:"uninstalls,omitzero"`

	// Type is the type of command to be run.
	Type CommandType `json:"type,omitempty"`

	// WorkingDirectory specifies a working directory for a command. If no
	// working directory is specified, the directory containing the executable
	// will be used.
	WorkingDirectory DirectoryResourceID `json:"working-directory,omitempty"`

	// Executable identifies an executable file to be run.
	//
	// For commands applied to archive packages, it identifies the executable
	// file within the archive, and will be interpreted as a PackageFileID.
	//
	// For non-pacakge commands, it identifies the executable file to be
	// invoked, and will be interpreted as a FileResourceID.
	//
	// For msi-based commands, the file will be provided to the msiexec
	// utility.
	Executable ExecutableID `json:"executable,omitempty"`

	// Args is the set of arguments to be passed to the command.
	Args []string `json:"args,omitzero"`

	// ExitCodes provide a map of known exit codes for the command.
	ExitCodes ExitCodeMap `json:"exit-codes,omitzero"`
}

// ExitCodeMap defines a set of expected exit codes.
type ExitCodeMap map[ExitCode]ExitCodeInfo

// ExitCode is an exit code returned from a command.
type ExitCode int

// ExitCodeInfo stores information about an exit code.
type ExitCodeInfo struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	OK          bool   `json:"ok,omitempty"`
}

// CommandResult stores information about an exit code returned by a command.
type CommandResult struct {
	ExitCode ExitCode
	Info     ExitCodeInfo
}

// String returns a string representation of the command result.
func (r CommandResult) String() string {
	var builder structformat.Builder
	builder.WritePrimary("exit code")
	if r.Info.OK {
		builder.WritePrimary(fmt.Sprintf("%d [OK]", r.ExitCode))
	} else {
		builder.WritePrimary(strconv.Itoa(int(r.ExitCode)))
	}
	if r.Info.Name != "" {
		builder.WritePrimary(r.Info.Name)
	}
	if r.Info.Description != "" {
		builder.WriteStandard(r.Info.Description)
	}
	return builder.String()
}

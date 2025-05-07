package msiresult

import (
	"fmt"
	"runtime"
	"strconv"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// ExitCode is an exit code produced by msiexec.
type ExitCode int

// Info returns information about the exit code if it is recognized.
func (code ExitCode) Info() lbdeploy.ExitCodeInfo {
	return InfoMap[code]
}

// Error returns an error string for the exit code.
func (code ExitCode) Error() string {
	var out string

	// Start by formatting in the same manner as exec.ExitCode.Error().
	if runtime.GOOS == "windows" && uint(code) >= 1<<16 {
		out = "exit status " + fmt.Sprintf("%x", uint(code))
	} else {
		out = "exit status " + strconv.Itoa(int(code))
	}

	// If we have more information about this particular exit code, include
	// it.
	info := code.Info()
	if info.Name != "" {
		out += ": " + info.Name
	}
	if info.Description != "" {
		out += ": " + info.Description
	}

	return out
}

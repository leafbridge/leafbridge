package lbengine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbdeployevent"
	"github.com/leafbridge/leafbridge/core/lbevent"
	"github.com/leafbridge/leafbridge/core/msi/msiresult"
	"github.com/leafbridge/leafbridge/internal/mergereader"
	"github.com/leafbridge/leafbridge/platform/windows/localfs"
	"github.com/leafbridge/leafbridge/platform/windows/stagingfs"
	"github.com/leafbridge/leafbridge/platform/windows/tempfs"
	"github.com/leafbridge/leafbridge/utility/bytesconv"
)

// commandData holds the ID and definition for a command.
type commandData struct {
	ID         lbdeploy.CommandID
	Definition lbdeploy.Command
}

// commandEngine manages invocation of a command.
type commandEngine struct {
	deployment lbdeploy.Deployment
	flow       flowData
	action     actionData
	pkg        packageData
	command    commandData
	apps       lbdeploy.AppEvaluation
	events     lbevent.Recorder
	force      bool
	state      *engineState
}

// InvokeStandard runs the command without a package affiliation.
func (engine *commandEngine) InvokeStandard(ctx context.Context) error {
	// Prepare a local file system resolver.
	resolver := localfs.NewResolver(engine.deployment.Resources.FileSystem)

	// Get information about the executable file from the file system.
	fileID := lbdeploy.FileResourceID(engine.command.Definition.Executable)
	fileRef, err := resolver.ResolveFile(fileID)
	if err != nil {
		return fmt.Errorf("%s refers to an executable file \"%s\" that could not be resolved: %w", engine.cmdDesc(), fileID, err)
	}

	// Open the directory above the executable file.
	fileDir, err := localfs.OpenDir(fileRef.Dir())
	if err != nil {
		return fmt.Errorf("verification of the command executable failed: %w", err)
	}
	defer fileDir.Close()

	// Verify that the executable file exists and is a regular file.
	fi, err := fileDir.System().Stat(fileRef.FilePath)
	if err != nil {
		return fmt.Errorf("verification of the command executable failed: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return errors.New("verification of the command executable failed: the executable file path is not a regular file")
	}

	// Prepare an absolute path for the command.
	localized, err := filepath.Localize(fileRef.FilePath)
	if err != nil {
		return fmt.Errorf("an executable file path could not be prepared for %s: %w", engine.cmdDesc(), err)
	}
	execPath := filepath.Join(fileDir.Path(), localized)

	return engine.invokePath(ctx, execPath)
}

// InvokePackage runs the command on a package contained in dir.
func (engine *commandEngine) InvokePackage(ctx context.Context, dir stagingfs.PackageDir) error {
	// Verify that the executable file exists within the package's staging
	// directory.
	fi, err := dir.Stat(engine.pkg.Definition)
	if err != nil {
		return fmt.Errorf("verification of the command executable failed: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return errors.New("verification of the command executable failed: the executable file path is not a regular file")
	}

	// Prepare an absolute path for the command.
	execPath, err := dir.FilePath(engine.pkg.Definition)
	if err != nil {
		return fmt.Errorf("an executable file path could not be prepared for %s: %w", engine.cmdDesc(), err)
	}

	return engine.invokePath(ctx, execPath)
}

// InvokeArchive runs the command on a set of extracted archive package files.
func (engine *commandEngine) InvokeArchive(ctx context.Context, files tempfs.ExtractionDir) error {
	// Get information about the executable file from the package.
	fileID := lbdeploy.PackageFileID(engine.command.Definition.Executable)
	fileData, exists := engine.pkg.Definition.Files[fileID]
	if !exists {
		return fmt.Errorf("%s refers to an executable file \"%s\" that is not defined in the \"%s\" package", engine.cmdDesc(), fileID, engine.pkg.ID)
	}

	// Verify that the executable file exists within the extracted file set.
	fi, err := files.Stat(fileData.Path)
	if err != nil {
		return fmt.Errorf("verification of the command executable failed: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return errors.New("verification of the command executable failed: the executable file path is not a regular file")
	}

	// Prepare an absolute path for the command.
	execPath, err := files.FilePath(fileData.Path)
	if err != nil {
		return fmt.Errorf("an executable file path could not be prepared for %s: %w", engine.cmdDesc(), err)
	}

	return engine.invokePath(ctx, execPath)
}

// InvokeApp runs the command against an application's product code.
func (engine *commandEngine) InvokeApp(ctx context.Context) error {
	// Determine what application we will be operting on.
	var app lbdeploy.AppID
	switch engine.command.Definition.Type {
	case lbdeploy.CommandTypeMSIUninstallProductCode:
		if len(engine.command.Definition.Uninstalls) != 1 {
			return fmt.Errorf("%s must provide a single application ID to be uninstalled", engine.cmdDesc())
		}
		app = engine.command.Definition.Uninstalls[0]
	default:
		return fmt.Errorf("%s uses a \"%s\" command type that is not recognized or is not suitable for app-based invocation", engine.cmdDesc(), engine.command.Definition.Type)
	}

	// Get information about the application from the deployment.
	appData, exists := engine.deployment.Apps[app]
	if !exists {
		return fmt.Errorf("%s refers to an application \"%s\" that is not defined in the \"%s\" deployment", engine.cmdDesc(), app, engine.deployment.ID)
	}

	// Make sure a product code is defined.
	if appData.ProductCode == "" {
		return fmt.Errorf("%s refers to an application \"%s\" that does not have a product code", engine.cmdDesc(), app)
	}

	// Prepare the command arguments.
	args := engine.command.Definition.Args

	// Handle app-based command types.
	//
	// TODO: Switch to the Microsoft Installer API:
	// https://learn.microsoft.com/en-us/windows/win32/api/msi/nf-msi-msiinstallproductw
	switch engine.command.Definition.Type {
	case lbdeploy.CommandTypeMSIUninstallProductCode:
		args = append([]string{"/x", string(appData.ProductCode), "/quiet", "/norestart"}, args...)
	default:
		return fmt.Errorf("%s uses a \"%s\" command type that is not recognized or is not suitable for app-based invocation", engine.cmdDesc(), engine.command.Definition.Type)
	}

	// If a working directory was specified, resolve it.
	workingDir, err := engine.workingDirectory()
	if err != nil {
		return fmt.Errorf("a working directory could not be determined for %s: %w", engine.cmdDesc(), err)
	}

	// Find the msiexec executable.
	execPath, err := exec.LookPath("msiexec.exe")
	if err != nil {
		return fmt.Errorf("failed to locate the Windows Installer executable: %w", err)
	}

	return engine.invoke(ctx, workingDir, execPath, args)
}

func (engine *commandEngine) invokePath(ctx context.Context, execPath string) (err error) {
	// Determine a working directory for the command.
	workingDir, err := engine.workingDirectoryForExecutable(execPath)
	if err != nil {
		return fmt.Errorf("a working directory could not be determined for %s: %w", engine.cmdDesc(), err)
	}

	// Prepare the command arguments.
	args := engine.command.Definition.Args

	// Special handling for use of msiexec.
	//
	// TODO: Switch to the Microsoft Installer API:
	// https://learn.microsoft.com/en-us/windows/win32/api/msi/nf-msi-msiinstallproductw
	switch engine.command.Definition.Type {
	case lbdeploy.CommandTypeExe, "":
		return engine.invoke(ctx, workingDir, execPath, args)
	case lbdeploy.CommandTypeMSIInstall:
		args = append([]string{"/i", execPath, "/quiet", "/norestart"}, args...)
	case lbdeploy.CommandTypeMSIUpdate:
		args = append([]string{"/update", execPath, "/quiet", "/norestart"}, args...)
	case lbdeploy.CommandTypeMSIUninstall:
		args = append([]string{"/x", execPath, "/quiet", "/norestart"}, args...)
	default:
		return fmt.Errorf("an unknown command type was specified: %s", engine.command.Definition.Type)
	}

	// Find the msiexec executable.
	execPath, err = exec.LookPath("msiexec.exe")
	if err != nil {
		return fmt.Errorf("failed to locate the Windows Installer executable: %w", err)
	}

	return engine.invoke(ctx, workingDir, execPath, args)
}

func (engine *commandEngine) invoke(ctx context.Context, workingDir, execPath string, args []string) (err error) {
	// Check for cancellation before starting the command.
	if err := ctx.Err(); err != nil {
		return err
	}

	// Prepare a command that will be terminated when ctx is cancelled.
	cmd := exec.CommandContext(ctx, execPath, args...)

	// Set the command's working directory.
	cmd.Dir = workingDir

	// Configure the command to wait up to one minute for the command to close
	// out gracefully when its context is cancelled.
	//
	// TODO: Make this configurable.
	cmd.WaitDelay = time.Minute

	// Prepare two sets of output pipes for the command.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Record the start of the command.
	engine.events.Record(lbdeployevent.CommandStarted{
		Deployment:           engine.deployment.ID,
		Flow:                 engine.flow.ID,
		ActionIndex:          engine.action.Index,
		ActionType:           engine.action.Definition.Type,
		Package:              engine.pkg.ID,
		Command:              engine.command.ID,
		CommandLine:          cmd.String(),
		WorkingDirectory:     engine.command.Definition.WorkingDirectory,
		WorkingDirectoryPath: workingDir,
		Apps:                 engine.apps,
	})

	// Prepare a buffer to hold the combined command output.
	var output bytes.Buffer

	// Record the time that the command started.
	started := time.Now()

	// Start the command.
	err = cmd.Start()

	// If the command started successfully, send its output to stdout and
	// stderr as well as the output buffer, then wait for it to finish.
	if err == nil {
		// Tee stdout and stderr to the console.
		r1 := io.TeeReader(stdout, os.Stdout)
		r2 := io.TeeReader(stderr, os.Stderr)

		// Combine the output of both stdout and stderr.
		merged := mergereader.New(r1, r2)

		// Read the combined output from the command.
		io.Copy(&output, merged)

		// Wait for the command to be completed.
		err = cmd.Wait()
	}

	// Record the time that the command stopped.
	stopped := time.Now()

	// Analyze the exit code of the command.
	result, err := engine.buildResult(err)

	// Special handling for some exit codes returned by msiexec.
	switch engine.command.Definition.Type {
	case lbdeploy.CommandTypeMSIUninstall, lbdeploy.CommandTypeMSIUninstallProductCode:
		if exitCode, ok := err.(msiresult.ExitCode); ok {
			if exitCode == msiresult.UnknownProduct {
				err = nil // Already uninstalled
			}
		}
	}

	// Evaluate the effectiveness of any expected application changes.
	ae := NewAppEngine(engine.deployment)
	appSummary, appSummaryErr := ae.SummarizeAppChanges(engine.apps)
	if appSummaryErr != nil {
		appSummaryErr = fmt.Errorf("failed to determine the state of installed applications after the command was invoked: %w", appSummaryErr)
		if err == nil {
			err = appSummaryErr
		}
	}

	// Record the end of the command.
	engine.events.Record(lbdeployevent.CommandStopped{
		Deployment:           engine.deployment.ID,
		Flow:                 engine.flow.ID,
		ActionIndex:          engine.action.Index,
		ActionType:           engine.action.Definition.Type,
		Package:              engine.pkg.ID,
		Command:              engine.command.ID,
		CommandLine:          cmd.String(),
		Result:               result,
		Output:               bytesconv.DecodeString(output.Bytes()),
		WorkingDirectory:     engine.command.Definition.WorkingDirectory,
		WorkingDirectoryPath: workingDir,
		AppsBefore:           engine.apps,
		AppsAfter:            appSummary,
		Started:              started,
		Stopped:              stopped,
		Err:                  err,
	})

	// Wait 5 seconds to let the file system and file locks quiesce before
	// continuing on. This is especially important if this command is the last
	// action running for an extracted archive, and LeafBridge attempts to
	// delete extracted files immediately after this command has run.
	//
	// TODO: Make this delay configurable.
	//
	// TODO: Consider moving this to the state cleanup that actually deletes
	// the extracted files.
	timer := time.NewTimer(time.Second * 5)
	select {
	case <-ctx.Done():
		timer.Stop()
	case <-timer.C:
	}

	// If the command returned an error, return that.
	if err != nil {
		return err
	}

	// If the application summary indicates that an expected change to the
	// installed set of applications didn't take effect, return the error.
	return appSummary.Err()
}

// cmdDesc returns a string describing the command. It is used to build
// error messages.
func (engine *commandEngine) cmdDesc() string {
	if engine.pkg.ID != "" {
		return fmt.Sprintf("the \"%s\" command in the \"%s\" package", engine.command.ID, engine.pkg.ID)
	}
	return fmt.Sprintf("the \"%s\" command", engine.command.ID)
}

// workingDirectoryForExecutable returns an absolute path to the command's
// working directory. If a working directory was not provided for the command,
// it returns the directory containing the executable.
//
// If the working directory could not be resolved or does not exist, it
// returns an error.
func (engine *commandEngine) workingDirectoryForExecutable(execPath string) (path string, err error) {
	path, err = engine.workingDirectory()
	if err != nil || path != "" {
		return path, err
	}
	path = filepath.Dir(execPath)
	if path == "" {
		return "", fmt.Errorf("a directory could not be determined for the executable's path: %s", execPath)
	}
	return path, nil
}

// workingDirectory returns an absolute path to the command's working
// directory. If a working directory was not provided for the command, it
// returns an empty string.
//
// If the working directory could not be resolved or does not exist, it
// returns an error.
func (engine *commandEngine) workingDirectory() (path string, err error) {
	dirID := engine.command.Definition.WorkingDirectory
	if dirID == "" {
		return "", nil
	}

	resolver := localfs.NewResolver(engine.deployment.Resources.FileSystem)
	dirRef, err := resolver.ResolveDirectory(dirID)
	if err != nil {
		return "", err
	}

	dir, err := localfs.OpenDir(dirRef)
	if err != nil {
		return "", err
	}
	defer dir.Close()

	return dir.Path(), nil
}

func (engine *commandEngine) buildResult(cmdError error) (result lbdeploy.CommandResult, err error) {
	// If the command returned an error, examine it.
	if cmdError != nil {
		// Assume that any error returned by cmd.Wait() is a real error,
		// unless we later succeed in looking up an exit code that we're
		// familiar with and proving that it's okay.
		err = cmdError

		// If we can't interpret the error as an exit error, then something
		// strange happened when trying to run the command.
		var exitErr *exec.ExitError
		if !errors.As(cmdError, &exitErr) {
			return
		}

		// If the process state is missing, then the command didn't run, and there
		// is no exit code.
		if exitErr.ProcessState == nil {
			return
		}

		// Make sure the process has exited.
		if !exitErr.ProcessState.Exited() {
			return
		}

		// Record the exit code returned by the command.
		result.ExitCode = lbdeploy.ExitCode(exitErr.ExitCode())
	} else {
		// The command returned an exit code of zero.
		result.ExitCode = 0
	}

	// Attempt to look up the error code information in the command.
	if info, found := engine.command.Definition.ExitCodes[result.ExitCode]; found {
		result.Info = info
		if info.OK {
			err = nil
		}
		return
	}

	// If this is an msiexec command, look for an exit code that is well
	// known.
	if engine.command.Definition.Type.IsMSI() {
		code := msiresult.ExitCode(result.ExitCode)
		if info, found := msiresult.InfoMap[code]; found {
			result.Info = info
			if info.OK {
				err = nil
			} else {
				err = code // Return an msiexec exit code.
			}
			return
		}
	}

	return
}

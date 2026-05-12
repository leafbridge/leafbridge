package lbengine

import (
	"context"
	"fmt"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbdeployevent"
	"github.com/leafbridge/leafbridge/core/lbevent"
	"github.com/leafbridge/leafbridge/platform/windows/stagingfs"
	"github.com/leafbridge/leafbridge/platform/windows/tempfs"
)

// packageData holds the ID and definition for a package.
type packageData struct {
	ID         lbdeploy.PackageID
	Definition lbdeploy.Package
}

// packageEngine manages package-related actions.
type packageEngine struct {
	invocation lbdeploy.Invocation
	deployment lbdeploy.Deployment
	flow       flowData
	action     actionData
	pkg        packageData
	events     lbevent.Recorder
	output     CommandOutput
	state      *engineState
}

// preparePackage performs a package preparation action.
func (engine *packageEngine) PreparePackage(ctx context.Context) error {
	// Download and verify this package if necessary.
	_, err := engine.preparePackage(ctx)
	return err
}

// InvokeCommand performs a package command invocation action.
func (engine *packageEngine) InvokeCommand(ctx context.Context, command lbdeploy.CommandID, mode lbdeploy.CommandMode) error {
	// Interpret action data.
	action, ok := engine.action.Definition.(lbdeploy.InvokeCommandAction)
	if !ok {
		return fmt.Errorf("unable to invoke command \"%s\" within the \"%s\" package: the action is of type \"%s\"", command, engine.pkg.ID, engine.action.Definition.Type())
	}

	// Find the command within the package.
	commandDefinition, exists := engine.pkg.Definition.Commands[command]
	if !exists {
		return fmt.Errorf("the command \"%s\" does not exist within the \"%s\" package", command, engine.pkg.ID)
	}
	data := commandData{ID: command, Definition: commandDefinition}

	// Determine whether any app changes are anticipated.
	ae := NewAppEngine(engine.deployment)
	appEvaluation, err := ae.EvaluateAppChanges(commandDefinition.Installs, commandDefinition.Repairs, commandDefinition.Uninstalls)
	if err != nil {
		return fmt.Errorf("the evaluation of potential application changes did not succeed: %w", err)
	}

	// If the command declares that it installs, repairs or uninstalls
	// something, review the app evaluation to determine whether any
	// application changes are anticpated.
	if len(commandDefinition.Installs) > 0 || len(commandDefinition.Repairs) > 0 || len(commandDefinition.Uninstalls) > 0 {
		if !appEvaluation.ActionsNeeded(mode) {
			// If all app installs, repairs and uninstalls are already in
			// effect, and command invocation isn't forced, skip this command.
			if (!engine.invocation.Force && !action.Force) || commandDefinition.Type.IsAppBased() {
				// Record that this command is being skipped.
				engine.events.Record(lbdeployevent.CommandSkipped{
					Invocation:  engine.invocation.ID,
					Deployment:  engine.deployment.ID,
					Flow:        engine.flow.ID,
					ActionIndex: engine.action.Index,
					ActionType:  engine.action.Definition.Type(),
					Package:     engine.pkg.ID,
					Command:     command,
					CommandMode: mode,
					Apps:        appEvaluation,
				})

				return nil
			}
		}
	}

	// Handle app-based commands that are affiliated with a package but don't
	// require the package to actually be present. This is most common for
	// packages that are uninstalled through msiexec.
	if commandDefinition.Type.IsAppBased() {
		return engine.invokeAppCommand(ctx, data, appEvaluation)
	}

	// Handle commands for archive packages that must be downloaded and
	// extracted first.
	if engine.pkg.Definition.Type.IsArchive() {
		return engine.invokeArchiveCommand(ctx, data, appEvaluation)
	}

	// Handle commands for regular packages that must be downloaded first.
	return engine.invokePackageCommand(ctx, data, appEvaluation)
}

// invokePackageCommand runs a command on an normal package.
func (engine *packageEngine) invokePackageCommand(ctx context.Context, command commandData, apps lbdeploy.AppEvaluation) error {
	// Download and verify this package if necessary.
	packageDir, err := engine.preparePackage(ctx)
	if err != nil {
		return err
	}

	// Prepare a command engine.
	ce := commandEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		pkg:        engine.pkg,
		command:    command,
		apps:       apps,
		events:     engine.events,
		output:     engine.output,
		state:      engine.state,
	}

	// Invoke the command.
	return ce.InvokePackage(ctx, packageDir)
}

// invokeArchiveCommand runs a command on an archive package.
func (engine *packageEngine) invokeArchiveCommand(ctx context.Context, command commandData, apps lbdeploy.AppEvaluation) error {
	// Download, verify and extract the files in this package if necessary.
	extractedFiles, err := engine.prepareAndExtractArchivePackage(ctx)
	if err != nil {
		return err
	}

	// Prepare a command engine.
	ce := commandEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		pkg:        engine.pkg,
		command:    command,
		apps:       apps,
		events:     engine.events,
		output:     engine.output,
		state:      engine.state,
	}

	// Invoke the command.
	return ce.InvokeArchive(ctx, extractedFiles)
}

// invokeAppCommand runs a command on an application.
func (engine *packageEngine) invokeAppCommand(ctx context.Context, command commandData, apps lbdeploy.AppEvaluation) error {
	// Prepare a command engine.
	ce := commandEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		pkg:        engine.pkg,
		command:    command,
		apps:       apps,
		events:     engine.events,
		output:     engine.output,
		state:      engine.state,
	}

	// Invoke the command.
	return ce.InvokeApp(ctx)
}

func (engine *packageEngine) prepareAndExtractArchivePackage(ctx context.Context) (tempfs.ExtractionDir, error) {
	// Check the state to see whether we've already downloaded, verified and
	// extracted the files in this package.
	extractedFiles, alreadyExtracted := engine.state.extractedPackages[engine.pkg.ID]
	if alreadyExtracted {
		return extractedFiles, nil
	}

	// Download and verify the package if we haven't done so already.
	_, packageFile, err := engine.prepareAndOpenPackage(ctx)
	if err != nil {
		return tempfs.ExtractionDir{}, err
	}
	defer packageFile.Close()

	// Create a temporary directory to hold the extracted files.
	extractedFiles, err = tempfs.OpenExtractionDirForPackage(lbdeploy.PackageContent{
		ID:          engine.pkg.ID,
		PrimaryHash: engine.pkg.Definition.Attributes.Hashes.Primary(),
	}, tempfs.Options{
		DeleteOnClose: true,
	})
	if err != nil {
		return tempfs.ExtractionDir{}, fmt.Errorf("failed to prepare a directory for file extraction: %w", err)
	}

	// Prepare an extraction engine.
	ee := extractionEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		events:     engine.events,
		state:      engine.state,
	}

	// Extract the files.
	if err := ee.ExtractPackage(ctx, packageFile, extractedFiles); err != nil {
		// If the package files could not be extracted, close the extraction
		// directory without adding it to the state, then return the
		// error.
		extractedFiles.Close()

		return tempfs.ExtractionDir{}, fmt.Errorf("extraction failed: %w", err)
	}

	// Add the extracted files to the engine's state, so that they'll be
	// available for other flows.
	//
	// This will also cause the deployment engine to close the extracted
	// files after the deployment's invocation has finished.
	engine.state.extractedPackages[engine.pkg.ID] = extractedFiles

	return extractedFiles, nil
}

// preparePackage will download and verify a package file. It returns an open
// staging directory for the package.
//
// The returned [stagingfs.PackageDir] will be added to the engine's state,
// and thus will be closed automatically during state cleanup. As a result,
// the caller must not close the returned [stagingfs.PackageDir].
func (engine *packageEngine) preparePackage(ctx context.Context) (stagingfs.PackageDir, error) {
	// Check the state to see whether we've already downloaded and verified
	// the package file.
	if packageDir, alreadyVerified := engine.state.verifiedPackageFiles[engine.pkg.ID]; alreadyVerified {
		return packageDir, nil
	}

	// Prepare the package file.
	packageDir, packageFile, err := engine.prepareAndOpenPackage(ctx)
	if err != nil {
		return stagingfs.PackageDir{}, err
	}

	// Close the package file, as we won't be using it.
	packageFile.Close()

	return packageDir, nil
}

// prepareAndOpenPackage will download, verify and open a package file. It
// returns an open staging directory and file for the package.
//
// The returned [stagingfs.PackageDir] will be added to the engine's state,
// and thus will be closed automatically during state cleanup. As a result,
// the caller must not close the returned [stagingfs.PackageDir].
//
// It is the caller's responsibility to close the returned
// [stagingfs.PackageFile] when finished with it.
func (engine *packageEngine) prepareAndOpenPackage(ctx context.Context) (stagingfs.PackageDir, stagingfs.PackageFile, error) {
	// Check the state to see whether we've already downloaded and verified
	// the package file.
	if packageDir, alreadyVerified := engine.state.verifiedPackageFiles[engine.pkg.ID]; alreadyVerified {
		packageFile, err := packageDir.OpenFile(engine.pkg.Definition.FileName())
		if err != nil {
			return stagingfs.PackageDir{}, stagingfs.PackageFile{}, fmt.Errorf("failed to open package file: %w", err)
		}
		return packageDir, packageFile, nil
	}

	// Prepare the package directory.
	packageDir, err := engine.openPackageDir()
	if err != nil {
		return stagingfs.PackageDir{}, stagingfs.PackageFile{}, fmt.Errorf("failed to open package file directory: %w", err)
	}

	// Open the package file, or create it if it doesn't exist.
	packageFile, err := packageDir.OpenFile(engine.pkg.Definition.FileName())
	if err != nil {
		// If the package file could not be opened, close the package
		// directory without adding it to the state, then return the
		// error.
		packageDir.Close()

		return stagingfs.PackageDir{}, stagingfs.PackageFile{}, fmt.Errorf("failed to open package file: %w", err)
	}

	// Prepare a download engine.
	de := downloadEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		events:     engine.events,
		state:      engine.state,
	}

	// Download and verify the package data.
	//
	// If the file already contains the expected data, the
	// download will be skipped.
	//
	// If the file was partially downloaded, the download will be
	// resumed.
	if err := de.DownloadAndVerifyPackage(ctx, engine.pkg, packageFile); err != nil {
		// If the package file could not be prepared, close the package
		// file and directory without adding them to the state, then return
		// the error.
		packageFile.Close()
		packageDir.Close()

		return stagingfs.PackageDir{}, stagingfs.PackageFile{}, fmt.Errorf("failed to download and verify package file: %w", err)
	}

	// TODO: Consider calling the ReOpenFile API to change the file's desired
	// access to read-only.

	// Add the verified package directory to the engine's state, so that it
	// will be available for other flows.
	//
	// This will also cause the deployment engine to close the package
	// directory after the deployment's invocation has finished.
	engine.state.verifiedPackageFiles[engine.pkg.ID] = packageDir

	return packageDir, packageFile, nil
}

func (engine *packageEngine) openPackageDir() (stagingfs.PackageDir, error) {
	// Open the deployment's staging directory.
	deployDir, err := stagingfs.OpenDeployment(engine.deployment.ID)
	if err != nil {
		return stagingfs.PackageDir{}, err
	}
	defer deployDir.Close()

	// Open the package's staging directory.
	return deployDir.OpenPackage(
		lbdeploy.PackageContent{
			ID:          engine.pkg.ID,
			PrimaryHash: engine.pkg.Definition.Attributes.Hashes.Primary(),
		},
		engine.pkg.Definition.Type,
		engine.pkg.Definition.Format,
	)
}

func (engine *packageEngine) openPackageFile() (stagingfs.PackageFile, error) {
	// Open the package's staging directory.
	packageDir, err := engine.openPackageDir()
	if err != nil {
		return stagingfs.PackageFile{}, err
	}
	defer packageDir.Close()

	// Open the package file, or create it if it doesn't exist.
	return packageDir.OpenFile(engine.pkg.Definition.FileName())
}

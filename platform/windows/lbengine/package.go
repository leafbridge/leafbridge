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
	deployment lbdeploy.Deployment
	flow       flowData
	action     actionData
	pkg        packageData
	events     lbevent.Recorder
	force      bool
	state      *engineState
}

// preparePackage performs a package preparation action.
func (engine *packageEngine) PreparePackage(ctx context.Context) error {
	// Open the package file, or create it if it doesn't exist.
	file, err := engine.openPackageFile()
	if err != nil {
		return err
	}
	defer file.Close()

	// Prepare a download engine.
	de := downloadEngine{
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		events:     engine.events,
		state:      engine.state,
	}

	// Download and verify the package data.
	//
	// If the file already contains the expected data, the download will be
	// skipped.
	//
	// If the file was partially downloaded, the download will be resumed.
	return de.DownloadAndVerifyPackage(ctx, engine.pkg, file)
}

// InvokeCommand performs a package command invocation action.
func (engine *packageEngine) InvokeCommand(ctx context.Context, command lbdeploy.CommandID) error {
	// Find the command within the package.
	commandDefinition, exists := engine.pkg.Definition.Commands[command]
	if !exists {
		return fmt.Errorf("the command \"%s\" does not exist within the \"%s\" package", command, engine.pkg.ID)
	}
	data := commandData{ID: command, Definition: commandDefinition}

	// Determine whether any app changes are anticipated.
	ae := NewAppEngine(engine.deployment)
	appEvaluation, err := ae.EvaluateAppChanges(commandDefinition.Installs, commandDefinition.Uninstalls)
	if err != nil {
		return fmt.Errorf("the evaluation of potential application changes did not succeed: %w", err)
	}

	// If the command declares that it installs or uninstalls something,
	// review the app evaluation to determine whether any application changes
	// are anticpated.
	if len(commandDefinition.Installs) > 0 || len(commandDefinition.Uninstalls) > 0 {
		if !appEvaluation.ActionsNeeded() {
			// If all app installs and uninstalls are already in effect,
			// and command invocation isn't forced, skip this command.
			if !engine.force && !engine.action.Definition.Force {
				// Record that this command is being skipped.
				engine.events.Record(lbdeployevent.CommandSkipped{
					Deployment:  engine.deployment.ID,
					Flow:        engine.flow.ID,
					ActionIndex: engine.action.Index,
					ActionType:  engine.action.Definition.Type,
					Package:     engine.pkg.ID,
					Command:     command,
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
	// Check the state to see whether we've already downloaded and verified
	// the package file.
	packageDir, alreadyVerified := engine.state.verifiedPackageFiles[engine.pkg.ID]
	if !alreadyVerified {
		// Prepare the package directory.
		var err error
		packageDir, err = engine.openPackageDir()
		if err != nil {
			return fmt.Errorf("failed to prepare package file: %w", err)
		}

		// Prepare the package file.
		err = func() error {
			// Open the package file, or create it if it doesn't exist.
			packageFile, err := packageDir.OpenFile(engine.pkg.Definition)
			if err != nil {
				return fmt.Errorf("failed to prepare package file: %w", err)
			}
			defer packageFile.Close()

			// Prepare a download engine.
			de := downloadEngine{
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
				return err
			}

			return nil
		}()

		// If the package file could not be prepared, close the package
		// directory without adding it to the state, then return the
		// error.
		if err != nil {
			packageDir.Close()
			return err
		}

		// Add the verified package file to the engine's state, so that
		// it will be available for other flows.
		//
		// This will also cause the deployment engine to close the package
		// directory after the deployment's invocation has finished.
		engine.state.verifiedPackageFiles[engine.pkg.ID] = packageDir
	}

	// Prepare a command engine.
	ce := commandEngine{
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		pkg:        engine.pkg,
		command:    command,
		apps:       apps,
		events:     engine.events,
		force:      engine.force,
		state:      engine.state,
	}

	// Invoke the command.
	return ce.InvokePackage(ctx, packageDir)
}

// invokeArchiveCommand runs a command on an archive package.
func (engine *packageEngine) invokeArchiveCommand(ctx context.Context, command commandData, apps lbdeploy.AppEvaluation) error {
	// Check the state to see whether we've already downloaded, verified and
	// extracted the files in this package.
	extractedFiles, alreadyExtracted := engine.state.extractedPackages[engine.pkg.ID]

	// Download, verify and extract the package if we haven't done so already.
	if !alreadyExtracted {
		// Open the package file, or create it if it doesn't exist.
		packageFile, err := engine.openPackageFile()
		if err != nil {
			return fmt.Errorf("failed to prepare package file: %w", err)
		}
		defer packageFile.Close()

		// Prepare a download engine.
		de := downloadEngine{
			deployment: engine.deployment,
			flow:       engine.flow,
			action:     engine.action,
			events:     engine.events,
			state:      engine.state,
		}

		// Download and verify the package data.
		//
		// If the file already contains the expected data, the download will be
		// skipped.
		//
		// If the file was partially downloaded, the download will be resumed.
		if err := de.DownloadAndVerifyPackage(ctx, engine.pkg, packageFile); err != nil {
			return err
		}

		// Create a temporary directory to hold the extracted files.
		extractedFiles, err = tempfs.OpenExtractionDirForPackage(lbdeploy.PackageContent{
			ID:          engine.pkg.ID,
			PrimaryHash: engine.pkg.Definition.Attributes.Hashes.Primary(),
		}, tempfs.Options{
			DeleteOnClose: true,
		})
		if err != nil {
			return fmt.Errorf("failed to prepare a directory for file extraction: %w", err)
		}

		// Prepare an extraction engine.
		ee := extractionEngine{
			deployment: engine.deployment,
			flow:       engine.flow,
			action:     engine.action,
			events:     engine.events,
			state:      engine.state,
		}

		// Extract the files.
		if err := ee.ExtractPackage(ctx, packageFile, extractedFiles); err != nil {
			extractedFiles.Close()
			return fmt.Errorf("extraction failed: %w", err)
		}

		// Add the extracted files to the engine's state, so that they'll be
		// available for other flows.
		//
		// This will also cause the deployment engine to close the extracted
		// files after the deployment's invocation has finished.
		engine.state.extractedPackages[engine.pkg.ID] = extractedFiles
	}

	// Prepare a command engine.
	ce := commandEngine{
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		pkg:        engine.pkg,
		command:    command,
		apps:       apps,
		events:     engine.events,
		force:      engine.force,
		state:      engine.state,
	}

	// Invoke the command.
	return ce.InvokeArchive(ctx, extractedFiles)
}

// invokeAppCommand runs a command on an application.
func (engine *packageEngine) invokeAppCommand(ctx context.Context, command commandData, apps lbdeploy.AppEvaluation) error {
	// Prepare a command engine.
	ce := commandEngine{
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		pkg:        engine.pkg,
		command:    command,
		apps:       apps,
		events:     engine.events,
		force:      engine.force,
		state:      engine.state,
	}

	// Invoke the command.
	return ce.InvokeApp(ctx)
}

func (engine *packageEngine) openPackageDir() (stagingfs.PackageDir, error) {
	// Open the deployment's staging directory.
	deployDir, err := stagingfs.OpenDeployment(engine.deployment.ID)
	if err != nil {
		return stagingfs.PackageDir{}, err
	}
	defer deployDir.Close()

	// Open the package's staging directory.
	return deployDir.OpenPackage(lbdeploy.PackageContent{
		ID:          engine.pkg.ID,
		PrimaryHash: engine.pkg.Definition.Attributes.Hashes.Primary(),
	})
}

func (engine *packageEngine) openPackageFile() (stagingfs.PackageFile, error) {
	// Open the package's staging directory.
	packageDir, err := engine.openPackageDir()
	if err != nil {
		return stagingfs.PackageFile{}, err
	}
	defer packageDir.Close()

	// Open the package file, or create it if it doesn't exist.
	return packageDir.OpenFile(engine.pkg.Definition)
}

package lbengine

import (
	"context"
	"fmt"
	"time"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbdeployevent"
	"github.com/leafbridge/leafbridge/core/lbevent"
)

// actionData holds the index and definition for an action.
type actionData struct {
	Index      int
	Definition lbdeploy.Action
}

// actionEngine manages execution of an action within a flow.
type actionEngine struct {
	invocation lbdeploy.Invocation
	deployment lbdeploy.Deployment
	flow       flowData
	action     actionData
	events     lbevent.Recorder
	output     CommandOutput
	state      *engineState
}

func (engine *actionEngine) Invoke(ctx context.Context) error {
	// Record the start of the action.
	engine.events.Record(lbdeployevent.ActionStarted{
		Invocation:  engine.invocation.ID,
		Deployment:  engine.deployment.ID,
		Flow:        engine.flow.ID,
		ActionIndex: engine.action.Index,
		ActionType:  engine.action.Definition.Type(),
	})

	// Record the time that the action started.
	started := time.Now()

	// Execute the action.
	var err error
	{
		switch action := engine.action.Definition.(type) {
		case lbdeploy.StartFlowAction:
			err = engine.startFlow(ctx, action)
		case lbdeploy.PreparePackageAction:
			err = engine.preparePackage(ctx, action)
		case lbdeploy.InvokeCommandAction:
			err = engine.invokeCommand(ctx, action)
		case lbdeploy.CopyPackageFileAction:
			err = engine.copyPackageFile(ctx, action)
		case lbdeploy.CopyFileAction:
			err = engine.copyFile(ctx)
		case lbdeploy.DeleteFileAction:
			err = engine.deleteFile(ctx)
		default:
			err = fmt.Errorf("unrecognized deployment action type \"%s\"", engine.action.Definition.Type())
		}
	}

	// Record the time that the action stopped.
	stopped := time.Now()

	// Record the end of the action.
	engine.events.Record(lbdeployevent.ActionStopped{
		Invocation:  engine.invocation.ID,
		Deployment:  engine.deployment.ID,
		Flow:        engine.flow.ID,
		ActionIndex: engine.action.Index,
		ActionType:  engine.action.Definition.Type(),
		Started:     started,
		Stopped:     stopped,
		Err:         err,
	})

	return err
}

// startFlow starts another flow within the LeafBridge deployment.
func (engine *actionEngine) startFlow(ctx context.Context, action lbdeploy.StartFlowAction) error {
	flow := action.Flow

	// Find the requested flow within the deployment.
	definition, found := engine.deployment.Flows[flow]
	if !found {
		return fmt.Errorf("the \"%s\" flow does not exist within the \"%s\" deployment", flow, engine.deployment.ID)
	}

	// Prepare the flow engine.
	fe := flowEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow: flowData{
			ID:         flow,
			Definition: definition,
		},
		events: engine.events,
		output: engine.output,
		state:  engine.state,
	}

	// Invoke the requested flow.
	return fe.Invoke(ctx)
}

// preparePackage performs a package preparation action as part of a
// LeafBridge deployment.
func (engine *actionEngine) preparePackage(ctx context.Context, action lbdeploy.PreparePackageAction) error {
	// Look up the package by its ID.
	pkg, found := engine.deployment.Resources.Packages[action.Package]
	if !found {
		return fmt.Errorf("the \"%s\" package does not exist within the \"%s\" deployment", action.Package, engine.deployment.ID)
	}

	// Prepare a package engine.
	pe := packageEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		pkg: packageData{
			ID:         action.Package,
			Definition: pkg,
		},
		events: engine.events,
		output: engine.output,
		state:  engine.state,
	}

	// Execute the prepare-package action via the package engine.
	return pe.PreparePackage(ctx)
}

// invokeCommand invokes a command action.
func (engine *actionEngine) invokeCommand(ctx context.Context, action lbdeploy.InvokeCommandAction) error {
	// Special handling for package-based commands.
	if action.Package != "" {
		// Look up the package by its ID.
		pkg, found := engine.deployment.Resources.Packages[action.Package]
		if !found {
			return fmt.Errorf("the \"%s\" package does not exist within the \"%s\" deployment", action.Package, engine.deployment.ID)
		}

		// Prepare a package engine.
		pe := packageEngine{
			invocation: engine.invocation,
			deployment: engine.deployment,
			flow:       engine.flow,
			action:     engine.action,
			pkg: packageData{
				ID:         action.Package,
				Definition: pkg,
			},
			events: engine.events,
			output: engine.output,
			state:  engine.state,
		}

		// Execute the package command via the package engine.
		return pe.InvokeCommand(ctx, action.Command, action.Mode)
	}

	// Look up the command by its ID.
	var command commandData
	{
		definition, found := engine.deployment.Commands[action.Command]
		if !found {
			return fmt.Errorf("the \"%s\" command does not exist within the \"%s\" deployment", action.Command, engine.deployment.ID)
		}
		command = commandData{ID: action.Command, Definition: definition, Mode: action.Mode}
	}

	// Determine whether any app changes are anticipated.
	ae := NewAppEngine(engine.deployment)
	appEvaluation, err := ae.EvaluateAppChanges(command.Definition.Installs, command.Definition.Repairs, command.Definition.Uninstalls)
	if err != nil {
		return fmt.Errorf("the evaluation of potential application changes did not succeed: %w", err)
	}

	// If the command declares that it installs, repairs or uninstalls
	// something, review the app evaluation to determine whether any
	// application changes are anticpated.
	if len(command.Definition.Installs) > 0 || len(command.Definition.Repairs) > 0 || len(command.Definition.Uninstalls) > 0 {
		if !appEvaluation.ActionsNeeded(command.Mode) {
			// If all app installs, repairs and uninstalls are already in
			// effect, and command invocation isn't forced, skip this command.
			if (!engine.invocation.Force && !action.Force) || command.Definition.Type.IsAppBased() {
				// Record that this command is being skipped.
				engine.events.Record(lbdeployevent.CommandSkipped{
					Invocation:  engine.invocation.ID,
					Deployment:  engine.deployment.ID,
					Flow:        engine.flow.ID,
					ActionIndex: engine.action.Index,
					ActionType:  action.Type(),
					Command:     command.ID,
					CommandMode: command.Mode,
					Apps:        appEvaluation,
				})

				return nil
			}
		}
	}

	// Prepare a command engine.
	ce := commandEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		command:    command,
		apps:       appEvaluation,
		events:     engine.events,
		output:     engine.output,
		state:      engine.state,
	}

	// Special handling for commands that apply to an application's product
	// codes, and not to a provided executable or installer file.
	if command.Definition.Type.IsAppBased() {
		return ce.InvokeApp(ctx)
	}

	// Invoke the command.
	return ce.InvokeStandard(ctx)
}

// copyPackageFile performs a package file copy operation.
func (engine *actionEngine) copyPackageFile(ctx context.Context, action lbdeploy.CopyPackageFileAction) error {
	// Look up the package by its ID.
	pkg, found := engine.deployment.Resources.Packages[action.Package]
	if !found {
		return fmt.Errorf("the \"%s\" package does not exist within the \"%s\" deployment", action.Package, engine.deployment.ID)
	}

	// Prepare a package engine.
	pe := packageEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		pkg: packageData{
			ID:         action.Package,
			Definition: pkg,
		},
		events: engine.events,
		output: engine.output,
		state:  engine.state,
	}

	// Execute the package command via the package engine.
	return pe.CopyPackageFile(ctx)
}

// copyFile performs a file copy operation.
func (engine *actionEngine) copyFile(ctx context.Context) error {
	// Prepare a file engine.
	fe := fileEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		events:     engine.events,
		state:      engine.state,
	}

	// Execute the copy-file action via the file engine.
	return fe.CopyFile(ctx)
}

// deleteFile performs a file delete operation.
func (engine *actionEngine) deleteFile(ctx context.Context) error {
	// Prepare a file engine.
	fe := fileEngine{
		invocation: engine.invocation,
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		events:     engine.events,
		state:      engine.state,
	}

	// Execute the delete-file action via the file engine.
	return fe.DeleteFile(ctx)
}

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
	deployment lbdeploy.Deployment
	flow       flowData
	action     actionData
	events     lbevent.Recorder
	force      bool
	state      *engineState
}

func (engine *actionEngine) Invoke(ctx context.Context) error {
	// Record the start of the action.
	engine.events.Record(lbdeployevent.ActionStarted{
		Deployment:  engine.deployment.ID,
		Flow:        engine.flow.ID,
		ActionIndex: engine.action.Index,
		ActionType:  engine.action.Definition.Type,
	})

	// Record the time that the action started.
	started := time.Now()

	// Execute the action.
	err := func() error {
		switch engine.action.Definition.Type {
		case lbdeploy.ActionStartFlow:
			if err := engine.startFlow(ctx); err != nil {
				return err
			}
		case lbdeploy.ActionPreparePackage:
			if err := engine.preparePackage(ctx); err != nil {
				return err
			}
		case lbdeploy.ActionInvokeCommand:
			if err := engine.invokeCommand(ctx); err != nil {
				return err
			}
		case lbdeploy.ActionCopyFile:
			if err := engine.copyFile(ctx); err != nil {
				return err
			}
		case lbdeploy.ActionDeleteFile:
			if err := engine.deleteFile(ctx); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unrecognized deployment action type \"%s\"", engine.action.Definition.Type)
		}
		return nil
	}()

	// Record the time that the action stopped.
	stopped := time.Now()

	// Record the end of the action.
	engine.events.Record(lbdeployevent.ActionStopped{
		Deployment:  engine.deployment.ID,
		Flow:        engine.flow.ID,
		ActionIndex: engine.action.Index,
		ActionType:  engine.action.Definition.Type,
		Started:     started,
		Stopped:     stopped,
		Err:         err,
	})

	return err
}

// startFlow starts another flow within the LeafBridge deployment.
func (engine *actionEngine) startFlow(ctx context.Context) error {
	flow := engine.action.Definition.Flow

	// Find the requested flow within the deployment.
	definition, found := engine.deployment.Flows[flow]
	if !found {
		return fmt.Errorf("the \"%s\" flow does not exist within the \"%s\" deployment", flow, engine.deployment.ID)
	}

	// Prepare the flow engine.
	fe := flowEngine{
		deployment: engine.deployment,
		flow: flowData{
			ID:         flow,
			Definition: definition,
		},
		events: engine.events,
		force:  engine.force,
		state:  engine.state,
	}

	// Invoke the requested flow.
	return fe.Invoke(ctx)
}

// preparePackage performs a package preparation action as part of a
// LeafBridge deployment.
func (engine *actionEngine) preparePackage(ctx context.Context) error {
	// Look up the package by its ID.
	pkg, found := engine.deployment.Resources.Packages[engine.action.Definition.Package]
	if !found {
		return fmt.Errorf("the \"%s\" package does not exist within the \"%s\" deployment", engine.action.Definition.Package, engine.deployment.ID)
	}

	// Prepare a package engine.
	pe := packageEngine{
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		pkg: packageData{
			ID:         engine.action.Definition.Package,
			Definition: pkg,
		},
		events: engine.events,
		force:  engine.force,
		state:  engine.state,
	}

	// Execute the prepare-package action via the package engine.
	return pe.PreparePackage(ctx)
}

// invokeCommand invokes a command action.
func (engine *actionEngine) invokeCommand(ctx context.Context) error {
	// Special handling for package-based commands.
	if engine.action.Definition.Package != "" {
		// Look up the package by its ID.
		pkg, found := engine.deployment.Resources.Packages[engine.action.Definition.Package]
		if !found {
			return fmt.Errorf("the \"%s\" package does not exist within the \"%s\" deployment", engine.action.Definition.Package, engine.deployment.ID)
		}

		// Prepare a package engine.
		pe := packageEngine{
			deployment: engine.deployment,
			flow:       engine.flow,
			action:     engine.action,
			pkg: packageData{
				ID:         engine.action.Definition.Package,
				Definition: pkg,
			},
			events: engine.events,
			force:  engine.force,
			state:  engine.state,
		}

		// Execute the package command via the package engine.
		return pe.InvokeCommand(ctx, engine.action.Definition.Command)
	}

	// Look up the command by its ID.
	var command commandData
	{
		definition, found := engine.deployment.Commands[engine.action.Definition.Command]
		if !found {
			return fmt.Errorf("the \"%s\" command does not exist within the \"%s\" deployment", engine.action.Definition.Command, engine.deployment.ID)
		}
		command = commandData{ID: engine.action.Definition.Command, Definition: definition}
	}

	// Determine whether any app changes are anticipated.
	ae := NewAppEngine(engine.deployment)
	appEvaluation, err := ae.EvaluateAppChanges(command.Definition.Installs, command.Definition.Uninstalls)
	if err != nil {
		return fmt.Errorf("the evaluation of potential application changes did not succeed: %w", err)
	}

	// If the command declares that it installs or uninstalls something,
	// review the app evaluation to determine whether any application changes
	// are anticpated.
	if len(command.Definition.Installs) > 0 || len(command.Definition.Uninstalls) > 0 {
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
					Command:     command.ID,
					Apps:        appEvaluation,
				})

				return nil
			}
		}
	}

	// Prepare a command engine.
	ce := commandEngine{
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		command:    command,
		apps:       appEvaluation,
		events:     engine.events,
		force:      engine.force,
		state:      engine.state,
	}

	// Special handling for commands that apply to an application's product
	// code, and not to a provided executable or installer file.
	if command.Definition.Type.IsAppBased() {
		return ce.InvokeApp(ctx)
	}

	// Invoke the command.
	return ce.InvokeStandard(ctx)
}

// copyFile performs a file copy operation.
func (engine *actionEngine) copyFile(ctx context.Context) error {
	// Prepare a file engine.
	fe := fileEngine{
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
		deployment: engine.deployment,
		flow:       engine.flow,
		action:     engine.action,
		events:     engine.events,
		state:      engine.state,
	}

	// Execute the delete-file action via the file engine.
	return fe.DeleteFile(ctx)
}

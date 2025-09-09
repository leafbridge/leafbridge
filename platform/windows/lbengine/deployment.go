package lbengine

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbdeployevent"
	"github.com/leafbridge/leafbridge/core/lbevent"
	"github.com/leafbridge/leafbridge/core/lbrand"
)

// DeploymentEngine is a LeafBridge engine that is responsible for invocation
// of deployments.
type DeploymentEngine struct {
	deployment lbdeploy.Deployment
	events     lbevent.Recorder
	force      bool
	state      *engineState
}

// NewDeploymentEngine returns a new LeafBridge deployment engine for the
// given deployment and options.
func NewDeploymentEngine(deployment lbdeploy.Deployment, opts Options) DeploymentEngine {
	return DeploymentEngine{
		deployment: deployment,
		events:     opts.Events,
		force:      opts.Force,
		state:      newEngineState(),
	}
}

// Invoke executes a flow within a LeafBridge deployment.
func (engine DeploymentEngine) Invoke(ctx context.Context, flow lbdeploy.FlowID) error {
	// Check for context cancellation.
	if err := ctx.Err(); err != nil {
		return err
	}

	// Record the time that the deployment started.
	started := time.Now()

	// Ensure that the deployment is valid.
	if err := engine.deployment.Validate(); err != nil {
		return err
	}

	// Prepare an invocation.
	host, _ := os.Hostname()
	invocation := lbdeploy.Invocation{
		ID:          lbrand.NewID[lbdeploy.InvocationID](),
		Host:        host,
		InitialFlow: flow,
		Started:     started,
	}

	// Find the requested flow within the deployment.
	definition, found := engine.deployment.Flows[flow]
	if !found {
		return fmt.Errorf("the flow \"%s\" does not exist within the \"%s\" deployment", flow, engine.deployment.ID)
	}

	// Release resources when we are finished.
	defer func() {
		// Close and remove any extracted files in temporary directories.
		for packageID, extractedFiles := range engine.state.extractedPackages {
			extractedFiles.Close()
			delete(engine.state.extractedPackages, packageID)
		}

		// Close any open package directories.
		for packageID, packageDir := range engine.state.verifiedPackageFiles {
			packageDir.Close()
			delete(engine.state.verifiedPackageFiles, packageID)
		}

		// Release and close all locks.
		engine.state.locks.CloseAll()
	}()

	// Record the start of the deployment.
	engine.events.Record(lbdeployevent.DeploymentStarted{
		Invocation: invocation,
		Deployment: engine.deployment,
	})

	// Invoke the requested flow.
	fe := flowEngine{
		invocation: invocation,
		deployment: engine.deployment,
		flow: flowData{
			ID:         flow,
			Definition: definition,
		},
		events: engine.events,
		force:  engine.force,
		state:  engine.state,
	}

	err := fe.Invoke(ctx)

	// Record the time that the flow stopped.
	stopped := time.Now()

	// Record the end of the deployment.
	engine.events.Record(lbdeployevent.DeploymentStopped{
		Invocation: invocation.ID,
		Deployment: engine.deployment.ID,
		Started:    started,
		Stopped:    stopped,
		Err:        err,
	})

	return err
}

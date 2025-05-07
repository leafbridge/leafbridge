package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbevent"
	"github.com/leafbridge/leafbridge/platform/windows/lbengine"
	"github.com/leafbridge/leafbridge/platform/windows/windowsevent"
)

// DeployCmd deploys software according to a LeafBridge deployment
// configuration.
type DeployCmd struct {
	ConfigFile string          `kong:"required,name='config-file',help='Path to a deployment file describing the deployment.'"`
	Flow       lbdeploy.FlowID `kong:"required,name='flow',help='The flow to invoke within the deployment.'"`
	Force      bool            `kong:"optional,name='force',help='Force processing of the commands that would normally be skipped.'"`
	Verbose    bool            `kong:"optional,name='verbose',short='v',help='Show debug messages on the command line.'"`
}

// Run executes the LeafBridge deploy command.
func (cmd DeployCmd) Run(ctx context.Context) error {
	// Read the deployment file.
	dep, err := loadDeployment(cmd.ConfigFile)
	if err != nil {
		return err
	}

	// Select an event recorder.
	/*
		recorder := lbevent.Recorder{Handler: lbevent.LoggedHandler{}}
		recorder := lbevent.Recorder{Handler: lbevent.LoggedHandler{
			Handler: slog.NewJSONHandler(os.Stdout, nil),
		}}
	*/

	// Attempt to use a Windows event handler, but carry on regardless if it
	// doens't work out. The most likely reason it won't work is if the
	// running process isn't elevated.
	var handler lbevent.Handler
	{
		min := slog.LevelInfo
		if cmd.Verbose {
			min = slog.LevelDebug
		}
		basicHandler := lbevent.NewBasicHandler(os.Stdout, min)
		windowsHandler, err := windowsevent.NewHandler()
		if err != nil {
			handler = basicHandler
		} else {
			handler = lbevent.MultiHandler{basicHandler, windowsHandler}
		}
	}
	recorder := lbevent.Recorder{Handler: handler}

	// Prepare a new deployment engine for the deployment.
	engine := lbengine.NewDeploymentEngine(dep, lbengine.Options{
		Events: recorder,
		Force:  cmd.Force,
	})

	// Invoke the requested flow within the deployment.
	return engine.Invoke(ctx, cmd.Flow)
}

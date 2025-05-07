package lbengine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbdeployevent"
	"github.com/leafbridge/leafbridge/core/lbevent"
)

// flowData holds the ID and definition for a flow.
type flowData struct {
	ID         lbdeploy.FlowID
	Definition lbdeploy.Flow
}

// flowEngine manages execution of a flow within a deployment.
type flowEngine struct {
	deployment lbdeploy.Deployment
	flow       flowData
	events     lbevent.Recorder
	force      bool
	state      *engineState
}

func (engine flowEngine) Invoke(ctx context.Context) error {
	// Check for context cancellation.
	if err := ctx.Err(); err != nil {
		return err
	}

	// Check for a flow cycle and stop if one is detected.
	if engine.state.activeFlows.Contains(engine.flow.ID) {
		// Record the failure to start the flow.
		engine.events.Record(lbdeployevent.FlowAlreadyRunning{
			Deployment: engine.deployment.ID,
			Flow:       engine.flow.ID,
		})
		return fmt.Errorf("the \"%s\" flow is already running", engine.flow.ID)
	}

	// Evaluate all constraints for the flow.
	if conditions := engine.flow.Definition.Constraints; len(conditions) > 0 {
		// Prepare a condition engine.
		ce := NewConditionEngine(engine.deployment)

		// Evaluate each condition.
		var passed, failed lbdeploy.ConditionList
		for i, condition := range conditions {
			result, err := ce.Evaluate(condition)
			if err != nil {
				// Record the evaluation failure.
				engine.events.Record(lbdeployevent.FlowCondition{
					Deployment: engine.deployment.ID,
					Flow:       engine.flow.ID,
					Use:        lbdeploy.ConditionUseConstraint,
					Err:        err,
				})

				return fmt.Errorf("the \"%s\" flow failed to evaluate constraint %d: %w", engine.flow.ID, i+1, err)
			}
			if !result {
				failed = append(failed, condition)
			} else {
				passed = append(passed, condition)
			}
		}

		// Record the results of the evaluation.
		engine.events.Record(lbdeployevent.FlowCondition{
			Deployment: engine.deployment.ID,
			Flow:       engine.flow.ID,
			Use:        lbdeploy.ConditionUseConstraint,
			Passed:     passed,
			Failed:     failed,
		})

		// If any of the constrains failed, skip execution.
		if len(failed) > 0 {
			return nil
		}
	}

	// Evaluate all preconditions for the flow.
	if conditions := engine.flow.Definition.Preconditions; len(conditions) > 0 {
		// Prepare a condition engine.
		ce := NewConditionEngine(engine.deployment)

		// Evaluate each condition.
		var passed, failed lbdeploy.ConditionList
		for i, condition := range conditions {
			result, err := ce.Evaluate(condition)
			if err != nil {
				// Record the evaluation failure.
				engine.events.Record(lbdeployevent.FlowCondition{
					Deployment: engine.deployment.ID,
					Flow:       engine.flow.ID,
					Use:        lbdeploy.ConditionUsePrecondition,
					Err:        err,
				})

				return fmt.Errorf("the \"%s\" flow failed to evaluate precondition %d: %w", engine.flow.ID, i+1, err)
			}
			if !result {
				failed = append(failed, condition)
			} else {
				passed = append(passed, condition)
			}
		}

		// Record the results of the evaluation.
		engine.events.Record(lbdeployevent.FlowCondition{
			Deployment: engine.deployment.ID,
			Flow:       engine.flow.ID,
			Use:        lbdeploy.ConditionUsePrecondition,
			Passed:     passed,
			Failed:     failed,
		})

		// If any of the preconditions failed, stop execution.
		if len(failed) > 0 {
			return fmt.Errorf("the \"%s\" flow is unable to run because one or more preconditions failed: %s", engine.flow.ID, failed)
		}
	}

	// Attempt to acquire all of the locks required for this flow.
	if locks := engine.flow.Definition.Locks; len(locks) > 0 {
		// The lock manager ensures that all locks are reentrant, which means
		// they can be locked repeateadly within the program without causing
		// a deadlock.
		//
		// Each call to Lock() must be paired with a call to Unlock() so that
		// the reference counts can be maintained.

		// Create a lock group.
		group, err := engine.state.locks.Create(engine.deployment.Resources, locks...)
		if err != nil {
			return fmt.Errorf("the \"%s\" flow failed to prepare its lock group: %w", engine.flow.ID, err)
		}

		// Try to lock all members of the group.
		if err := group.Lock(); err != nil {
			// We failed to acquire one of the locks. Find out which one
			// failed.
			var lockID lbdeploy.LockID
			{
				var lockErr LockError
				if errors.As(err, &lockErr) {
					lockID = lockErr.LockID
				}
			}

			// Record the lock acquisition failure.
			engine.events.Record(lbdeployevent.FlowLockNotAcquired{
				Deployment: engine.deployment.ID,
				Flow:       engine.flow.ID,
				Lock:       lockID,
				Err:        err,
			})

			return fmt.Errorf("the \"%s\" flow failed to acquire locks for its entire lock group: %w", engine.flow.ID, err)
		}

		// Unlock all members when finished.
		defer group.Unlock()
	}

	// Prepare the behavior for this flow.
	behavior := lbdeploy.OverlayBehavior(engine.deployment.Behavior, engine.flow.Definition.Behavior)

	// Record this as a running flow as long as it is running.
	engine.state.activeFlows.Add(engine.flow.ID)
	defer engine.state.activeFlows.Remove(engine.flow.ID)

	// Record the start of the flow.
	engine.events.Record(lbdeployevent.FlowStarted{
		Deployment: engine.deployment.ID,
		Flow:       engine.flow.ID,
	})

	// Record the time that the flow started.
	started := time.Now()

	// Collect statistics.
	var stats lbdeploy.FlowStats

	// Execute each action in the flow.
	err := func() error {
		var errs []error
		for i, action := range engine.flow.Definition.Actions {
			// Check for context cancellation.
			if err := ctx.Err(); err != nil {
				errs = append(errs, err)
				break
			}

			// Create an action engine.
			ae := actionEngine{
				deployment: engine.deployment,
				flow:       engine.flow,
				action: actionData{
					Index:      i,
					Definition: action,
				},
				events: engine.events,
				force:  engine.force,
				state:  engine.state,
			}

			// Invoke the action.
			if err := ae.Invoke(ctx); err != nil {
				if ctx.Err() == err {
					break // Always stop when the context is cancelled.
				}

				stats.ActionsFailed++

				errs = append(errs, err)
				if behavior.OnError != lbdeploy.OnErrorContinue {
					break
				}
			} else {
				stats.ActionsCompleted++
			}
		}
		return errors.Join(errs...)
	}()

	// Record the time that the flow stopped.
	stopped := time.Now()

	// Record the end of the flow.
	engine.events.Record(lbdeployevent.FlowStopped{
		Deployment: engine.deployment.ID,
		Flow:       engine.flow.ID,
		Stats:      stats,
		Started:    started,
		Stopped:    stopped,
		Err:        err,
	})

	return err
}

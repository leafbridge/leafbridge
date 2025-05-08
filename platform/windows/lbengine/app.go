package lbengine

import (
	"fmt"
	"os"

	"github.com/gentlemanautomaton/winapp/appcode"
	"github.com/gentlemanautomaton/winapp/unpackaged"
	"github.com/gentlemanautomaton/winapp/unpackaged/appregistry"
	"github.com/gentlemanautomaton/winapp/unpackaged/appscope"
	"github.com/leafbridge/leafbridge/core/datatype"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbvalue"
	"github.com/leafbridge/leafbridge/platform/windows/localregistry"
)

// AppEngine is responsible for evaluating the status of applications on the
// local system.
type AppEngine struct {
	deployment lbdeploy.Deployment
}

// NewAppEngine prepares an app engine for the given deployment.
func NewAppEngine(dep lbdeploy.Deployment) AppEngine {
	return AppEngine{
		deployment: dep,
	}
}

// IsInstalled returns true if the application is installed on the local
// system.
//
// If it is unable to make a determination, it returns an error.
func (engine AppEngine) IsInstalled(app lbdeploy.AppID) (bool, error) {
	// Find the app within the deployment.
	definition, found := engine.deployment.Apps[app]
	if !found {
		return false, fmt.Errorf("the \"%s\" app does not exist within the \"%s\" deployment", app, engine.deployment.ID)
	}

	// If a presence condition has been supplied, use that to determine the
	// application's status.
	if definition.Detection.Present != "" {
		ce := NewConditionEngine(engine.deployment)
		return ce.Evaluate(definition.Detection.Present)
	}

	// Use the application registry that matches the application's
	// architecture (x64 or x86) and scope (machine or user).
	view, err := appregistry.ViewFor(appcode.Architecture(definition.Architecture), appscope.Scope(definition.Scope))
	if err != nil {
		return false, err
	}

	// Look for the application in the registry.
	return view.Contains(unpackaged.AppID(definition.ProductCode))
}

// Version returns the version number of the application if it is installed
// on the local system. If it is not present, it returns an empty string.
//
// If it is unable to make a determination, it returns an error.
func (engine AppEngine) Version(app lbdeploy.AppID) (datatype.Version, error) {
	// Find the app within the deployment.
	definition, found := engine.deployment.Apps[app]
	if !found {
		return "", fmt.Errorf("the \"%s\" app does not exist within the \"%s\" deployment", app, engine.deployment.ID)
	}

	// If a registry value that identifies the currently installed version has
	// been supplied, return its value.
	if definition.Detection.Version != "" {
		resolver := localregistry.NewResolver(engine.deployment.Resources.Registry)
		ref, err := resolver.ResolveValue(definition.Detection.Version)
		if err != nil {
			return "", err
		}
		key, err := localregistry.OpenKey(ref.Key())
		if err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", err
		}
		defer key.Close()
		value, err := key.GetValue(ref.Name, ref.Type)
		if err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", err
		}
		if value.Kind() == lbvalue.KindVersion {
			return value.Version(), nil
		}
		return "", fmt.Errorf("the \"%s\" registry value exists but does not contain a version", ref.Name)
	}

	// Use the application registry that matches the application's
	// architecture (x64 or x86) and scope (machine or user).
	view, err := appregistry.ViewFor(appcode.Architecture(definition.Architecture), appscope.Scope(definition.Scope))
	if err != nil {
		return "", err
	}

	// Retrieve the properties of the app from the registry.
	properties, err := view.Get(unpackaged.AppID(definition.ProductCode))
	if err != nil {
		return "", err
	}

	// If a DisplayVersion property is present, return it.
	return datatype.Version(properties.Attributes.GetString("DisplayVersion")), nil
}

// InstalledApps returns any of the apps in the list that are installed on the
// local system.
func (engine AppEngine) InstalledApps(list lbdeploy.AppList) (installed lbdeploy.AppList, err error) {
	for _, appID := range list {
		appIsInstalled, err := engine.IsInstalled(appID)
		if err != nil {
			return nil, fmt.Errorf("unable to determine the installation state of application \"%s\": %w", appID, err)
		}
		if appIsInstalled {
			installed = append(installed, appID)
		}
	}
	return
}

// MissingApps returns any of the apps in the list that are not installed on the
// local system.
func (engine AppEngine) MissingApps(list lbdeploy.AppList) (missing lbdeploy.AppList, err error) {
	for _, appID := range list {
		appIsInstalled, err := engine.IsInstalled(appID)
		if err != nil {
			return nil, fmt.Errorf("unable to determine the installation state of application \"%s\": %w", appID, err)
		}
		if !appIsInstalled {
			missing = append(missing, appID)
		}
	}
	return
}

// EvaluateAppChanges evaluates the changes needed to effect the given set of
// application installs and uninstalls.
func (engine AppEngine) EvaluateAppChanges(installs, uninstalls lbdeploy.AppList) (changes lbdeploy.AppEvaluation, err error) {
	alreadyInstalled, err := engine.InstalledApps(installs)
	if err != nil {
		return changes, err
	}
	toInstall := installs.Difference(alreadyInstalled)

	alreadyUninstalled, err := engine.MissingApps(uninstalls)
	if err != nil {
		return changes, err
	}
	toUninstall := uninstalls.Difference(alreadyUninstalled)

	return lbdeploy.AppEvaluation{
		AlreadyInstalled:   alreadyInstalled,
		AlreadyUninstalled: alreadyUninstalled,
		ToInstall:          toInstall,
		ToUninstall:        toUninstall,
	}, nil
}

// SummarizeAppChanges summarizes the effectiveness of application installs
// and uninstalls anticipated by a previous evaluation.
func (engine AppEngine) SummarizeAppChanges(evaluation lbdeploy.AppEvaluation) (changes lbdeploy.AppSummary, err error) {
	stillNotInstalled, err := engine.MissingApps(evaluation.ToInstall)
	if err != nil {
		return changes, err
	}
	installed := evaluation.ToInstall.Difference(stillNotInstalled)

	stillNotUninstalled, err := engine.InstalledApps(evaluation.ToUninstall)
	if err != nil {
		return changes, err
	}
	uninstalled := evaluation.ToUninstall.Difference(stillNotUninstalled)

	return lbdeploy.AppSummary{
		Installed:           installed,
		Uninstalled:         uninstalled,
		StillNotInstalled:   stillNotInstalled,
		StillNotUninstalled: stillNotUninstalled,
	}, nil
}

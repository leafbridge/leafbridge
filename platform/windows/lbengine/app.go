package lbengine

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/gentlemanautomaton/winapp/appcode"
	"github.com/gentlemanautomaton/winapp/unpackaged"
	"github.com/gentlemanautomaton/winapp/unpackaged/appregistry"
	"github.com/gentlemanautomaton/winapp/unpackaged/appscope"
	"github.com/leafbridge/leafbridge/core/datatype"
	"github.com/leafbridge/leafbridge/core/idset"
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

// Status returns the status of the application on the local system.
//
// If it is unable to make a determination, it returns an error.
func (engine AppEngine) Status(app lbdeploy.AppID) (status lbdeploy.AppStatus, err error) {
	// Find the app within the deployment.
	definition, found := engine.deployment.Apps[app]
	if !found {
		return status, fmt.Errorf("the status of the \"%s\" app could not be evaluated: the app does not exist within the \"%s\" deployment", app, engine.deployment.ID)
	}

	// Prepare a version set.
	status.Versions = make(datatype.VersionSet)

	// If a presence condition has been supplied, use that to determine the
	// application's installed condition.
	if definition.Detection.Present != "" {
		ce := NewConditionEngine(engine.deployment)
		installed, err := ce.Evaluate(definition.Detection.Present)
		if err != nil {
			return status, fmt.Errorf("the status of the \"%s\" app could not be determined: evaluation of the app's presence condition failed: %w", app, err)
		}
		if installed {
			status.Installed = true
		}
	}

	// If a registry value that identifies the currently installed version has
	// been supplied, exmaine it.
	if definition.Detection.Version != "" {
		version, err := engine.getVersionFromRegistry(definition.Detection.Version)
		if err != nil {
			return status, fmt.Errorf("the status of the \"%s\" app could not be determined: retrieval of the app's version value from the registry failed: %w", app, err)
		}
		if version != "" {
			status.Installed = true // Implied by a detected version.
			status.Versions.Add(version)
		}
	}

	// If a product code has been provide, retrieve the properties of the app
	// from the registry.
	if definition.ProductCode != "" {
		// Use the application registry that matches the application's
		// architecture (x64 or x86) and scope (machine or user).
		//
		// TODO: Consider throwing an error if the app is in user scope,
		// because we're probably running as SYSTEM and not as the user whose
		// scope we would want to check.
		view, err := appregistry.ViewFor(appcode.Architecture(definition.Architecture), appscope.Scope(definition.Scope))
		if err != nil {
			return status, fmt.Errorf("the status of the \"%s\" app could not be determined: preparation of the application registry view failed: %w", app, err)
		}

		// Use the existince of the product code within the registry as an
		// indication of the app's installation status.
		installed, err := view.Contains(unpackaged.AppID(definition.ProductCode))
		if err != nil {
			return status, fmt.Errorf("the status of the \"%s\" app could not be determined: the application registry could not be queried for the \"%s\" product code: %w", app, definition.ProductCode, err)
		}
		if installed {
			status.Installed = true
		}

		// Retrieve the properties for the application.
		//
		// TODO: Either develop a better way to retrieve just the property we
		// want, or store these properties somewhere useful.
		properties, err := view.Get(unpackaged.AppID(definition.ProductCode))
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return status, fmt.Errorf("the status of the \"%s\" app could not be determined: the data for the \"%s\" product code could not be retrieved from the application registry: %w", app, definition.ProductCode, err)
			}
		} else {
			// If a DisplayVersion property is present, add the version to the
			// list.
			version := datatype.Version(properties.Attributes.GetString("DisplayVersion"))
			if version != "" {
				status.Installed = true // Implied by a detected version.
				status.Versions.Add(version)
			}
		}
	}

	// Detect specific releases, if possible.
	for _, release := range definition.Releases {
		// Skip releases that don't define a version.
		if release.Version == "" {
			continue
		}

		if release.Detection.Present != "" {
			ce := NewConditionEngine(engine.deployment)
			installed, err := ce.Evaluate(release.Detection.Present)
			if err != nil {
				return status, fmt.Errorf("the status of the \"%s\" app could not be determined: evaluation of the \"%s\" release's presence condition failed: %w", app, release.Version, err)
			}
			if installed {
				status.Installed = true
				status.Versions.Add(release.Version)
			}
		}

		if release.ProductCode != "" {
			view, err := appregistry.ViewFor(appcode.Architecture(definition.Architecture), appscope.Scope(definition.Scope))
			if err != nil {
				return status, fmt.Errorf("the status of the \"%s\" app could not be determined: preparation of the application registry view failed: %w", app, err)
			}

			installed, err := view.Contains(unpackaged.AppID(release.ProductCode))
			if err != nil {
				return status, fmt.Errorf("the status of the \"%s\" app could not be determined: the application registry could not be queried for the \"%s\" release's \"%s\" product code: %w", app, release.Version, definition.ProductCode, err)
			}
			if installed {
				status.Installed = true
				status.Versions.Add(release.Version)
			}
		}
	}

	return status, nil
}

func (engine AppEngine) getVersionFromRegistry(resource lbdeploy.RegistryValueResourceID) (datatype.Version, error) {
	resolver := localregistry.NewResolver(engine.deployment.Resources.Registry)
	ref, err := resolver.ResolveValue(resource)
	if err != nil {
		return "", err
	}
	key, err := localregistry.OpenKey(ref.Key())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	defer key.Close()
	value, err := key.GetValue(ref.Name, ref.Type)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	if value.Kind() == lbvalue.KindVersion {
		return value.Version(), nil
	}
	return "", fmt.Errorf("the \"%s\" registry value exists but does not contain a version", ref.Name)
}

// StatusMap returns the app status for each of the given app IDs in a status
// map.
func (engine AppEngine) StatusMap(apps ...lbdeploy.AppID) (lbdeploy.AppStatusMap, error) {
	m := make(lbdeploy.AppStatusMap, len(apps))
	for _, app := range apps {
		if _, exists := m[app]; exists {
			continue // Skip duplicates
		}
		status, err := engine.Status(app)
		if err != nil {
			return nil, err
		}
		m[app] = status
	}
	return m, nil
}

// EvaluateAppChanges evaluates the changes needed to effect the given set of
// application installs and uninstalls.
func (engine AppEngine) EvaluateAppChanges(installs lbdeploy.AppList, uninstalls lbdeploy.AppCriteriaList) (changes lbdeploy.AppEvaluation, err error) {
	apps := make(appSet, len(installs)+len(uninstalls))
	for _, entry := range installs {
		apps.Add(entry.App)
	}
	for _, entry := range uninstalls {
		apps.Add(entry.App)
	}

	status, err := engine.StatusMap(apps.List()...)
	if err != nil {
		return changes, err
	}

	return lbdeploy.AppEvaluation{
		Status:       status,
		Installation: status.EvaluateAppInstallation(installs),
		Removal:      status.EvaluateAppRemoval(uninstalls),
	}, nil
}

// SummarizeAppChanges summarizes the effectiveness of application installs
// and uninstalls anticipated by a previous evaluation.
func (engine AppEngine) SummarizeAppChanges(evaluation lbdeploy.AppEvaluation) (changes lbdeploy.AppSummary, err error) {
	status, err := engine.StatusMap(evaluation.Status.Keys()...)
	if err != nil {
		return changes, err
	}

	var installed, stillNotInstalled lbdeploy.AppList
	for _, subject := range evaluation.Installation.ToInstall {
		if status.IsInstalled(subject) {
			installed = append(installed, subject)
		} else {
			stillNotInstalled = append(stillNotInstalled, subject)
		}
	}

	var uninstalled, stillNotUninstalled lbdeploy.AppList
	for _, subject := range evaluation.Removal.ToUninstall {
		if status.IsInstalled(subject) {
			stillNotUninstalled = append(stillNotUninstalled, subject)
		} else {
			uninstalled = append(uninstalled, subject)
		}
	}

	return lbdeploy.AppSummary{
		Installation: lbdeploy.AppInstallationSummary{
			Installed:         installed,
			StillNotInstalled: stillNotInstalled,
		},
		Removal: lbdeploy.AppRemovalSummary{
			Uninstalled:         uninstalled,
			StillNotUninstalled: stillNotUninstalled,
		},
	}, nil
}

// appSet keeps track of a set of application IDs.
type appSet = idset.SetOf[lbdeploy.AppID]

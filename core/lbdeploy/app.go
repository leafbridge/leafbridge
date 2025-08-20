package lbdeploy

import (
	"fmt"
	"slices"
	"strings"

	"github.com/leafbridge/leafbridge/core/datatype"
)

// AppMap holds a set of applications mapped by their identifiers.
//
// It is used to identify relevant applications for a deployment.
type AppMap map[AppID]Application

// AppID is a unique identifier for an application within LeafBridge.
type AppID string

// AppArchitecture identifies the processor architecture targeted by
// application code.
type AppArchitecture string

// AppScope identifies the scope of an application's installation.
type AppScope string

// ProductCode is an application's product code that uniquely identifies
// it to the operating system.
type ProductCode string

// Application holds identifying information for an application.
//
// If it defines an architecture, scope and unpackaged app ID, these will be
// used to determine if the application is installed in the Windows app
// registry.
//
// Alternatively, a condition may be specified that determines whether the
// application is installed.
type Application struct {
	Name         string          `json:"name"`
	Architecture AppArchitecture `json:"architecture,omitempty"`
	Scope        AppScope        `json:"scope,omitempty"`
	ProductCode  ProductCode     `json:"product-code,omitempty"`
	Detection    AppDetection    `json:"detection,omitzero"`
	Tags         AppReleaseTags  `json:"tags,omitempty"`
	Releases     AppReleaseList  `json:"releases,omitempty"`
}

// FindProductCode returns the product code for the given application version.
//
// If a product code is not defined for the application version, it returns
// the product code for the application as a whole.
func (app Application) FindProductCode(subject AppVersion) ProductCode {
	if subject.Version == "" {
		return app.ProductCode
	}

	if release, found := app.Releases.FindVersion(subject.Version); found {
		if release.ProductCode != "" {
			return release.ProductCode
		}
	}

	return app.ProductCode
}

// AppDetection describes how to detect the presence of an installed
// application and how to determine what version is installed.
type AppDetection struct {
	Present ConditionID             `json:"present,omitempty"`
	Version RegistryValueResourceID `json:"version,omitempty"`
}

// AppReleaseDetection describes how to detect the presence of a specific
// application release.
type AppReleaseDetection struct {
	Present ConditionID `json:"present,omitempty"`
}

// AppReleaseTag is a tag that can be applied to an application release.
type AppReleaseTag string

// AppReleaseTags is a map that assigns application release tags to release
// versions.
type AppReleaseTags map[AppReleaseTag]datatype.Version

// ReleaseTags returns the set of tags that are assigned to the given release
// version.
func (m AppReleaseTags) ReleaseTags(version datatype.Version) []AppReleaseTag {
	var tags []AppReleaseTag
	for tag, candidate := range m {
		if datatype.CompareVersions(candidate, version) == 0 {
			tags = append(tags, tag)
		}
	}
	slices.Sort(tags)
	return tags
}

// AppReleaseList holds a set of application releases.
type AppReleaseList []AppRelease

// FindVersion looks for a release within the list that matches the given
// version. If successful, it returns the release.
//
// TODO: Support more flexible matching of releases.
func (list AppReleaseList) FindVersion(version datatype.Version) (release AppRelease, ok bool) {
	for i := range list {
		if datatype.CompareVersions(datatype.Version(list[i].Version), version) == 0 {
			return list[i], true
		}
	}
	return
}

// AppRelease describes a specific release of an application.
type AppRelease struct {
	Name        string              `json:"name,omitempty"`
	Version     datatype.Version    `json:"version,omitempty"`
	Date        string              `json:"date,omitzero"`
	ProductCode ProductCode         `json:"product-code,omitempty"`
	Detection   AppReleaseDetection `json:"detection,omitzero"`
}

// AppList is a list of relevant applications for a deployment.
type AppList []AppVersion

// String returns a string representation of the list.
func (list AppList) String() string {
	var out strings.Builder
	for i, item := range list {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(item.String())
	}
	return out.String()
}

// AppVersion identifies an application and optionally a specific version.
type AppVersion struct {
	App     AppID            `json:"app"`
	Version datatype.Version `json:"version,omitempty"`
}

// String returns a string representation of the application and its version.
func (entry AppVersion) String() string {
	if entry.Version == "" {
		return string(entry.App)
	}
	return string(entry.App) + " v" + string(entry.Version.Canonical())
}

// AppCriteriaList is a list of applications that may also include version
// match criteria.
type AppCriteriaList []AppCriteria

// AppSearch identifies an application and optionally provides matching
// criteria for a specific version or range of versions.
type AppCriteria struct {
	App          AppID                 `json:"app"`
	Version      datatype.Version      `json:"version,omitempty"`
	VersionRange datatype.VersionRange `json:"version-range,omitzero"`
}

// Matches returns true if the app criteria matches the given application
// and version.
func (criteria AppCriteria) Matches(app AppVersion) bool {
	if app.App != criteria.App {
		return false
	}
	if criteria.Version != "" {
		if datatype.CompareVersions(criteria.Version, app.Version) != 0 {
			return false
		}
	}
	if !criteria.VersionRange.Includes(app.Version) {
		return false
	}
	return true
}

// AppStatusMap holds the installation status of a set of applications.
type AppStatusMap map[AppID]AppStatus

// Keys returns the set of application IDs that are present in the status map.
func (m AppStatusMap) Keys() (keys []AppID) {
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return
}

// IsInstalled returns true if the provided application is installed,
// according to the status map. If the subject contains a specific version,
// it will only return true if that version is installed.
func (m AppStatusMap) IsInstalled(subject AppVersion) bool {
	status, ok := m[subject.App]
	if !ok {
		return false
	}
	if !status.Installed {
		return false
	}
	if subject.Version == "" {
		return true
	}
	return status.Versions.Contains(subject.Version)
}

// InstalledApps returns the subset of the provided application list that is
// currently installed on the local system, according to the status map.
func (m AppStatusMap) InstalledApps(list AppList) (installed AppList, err error) {
	for _, subject := range list {
		if status, ok := m[subject.App]; ok {
			if !status.Installed {
				continue
			}
			if subject.Version == "" {
				if len(status.Versions) > 0 {
					for _, version := range status.Versions.List() {
						installed = append(installed, AppVersion{App: subject.App, Version: version})
					}
				} else {
					installed = append(installed, AppVersion{App: subject.App})
				}
			} else {
				if status.Versions.Contains(subject.Version) {
					installed = append(installed, AppVersion{App: subject.App, Version: subject.Version.Canonical()})
				}
			}
		}
	}
	return
}

// EvaluateAppInstallation evaluates the potential installation of any
// applications in the given list, according to the status map.
func (m AppStatusMap) EvaluateAppInstallation(list AppList) (evaluation AppInstallationEvaluation) {
	for _, subject := range list {
		if status, ok := m[subject.App]; ok {
			if !status.Installed {
				evaluation.ToInstall = append(evaluation.ToInstall, subject)
				evaluation.Missing = append(evaluation.Missing, subject)
				continue
			}
			if subject.Version == "" {
				if len(status.Versions) > 0 {
					for _, version := range status.Versions.List() {
						evaluation.AlreadyInstalled = append(evaluation.AlreadyInstalled, AppVersion{App: subject.App, Version: version})
					}
				} else {
					evaluation.AlreadyInstalled = append(evaluation.AlreadyInstalled, AppVersion{App: subject.App})
				}
			} else {
				if len(status.Versions) == 0 {
					evaluation.AlreadyInstalled = append(evaluation.AlreadyInstalled, AppVersion{App: subject.App})
				} else {
					if status.Versions.Contains(subject.Version) {
						evaluation.AlreadyInstalled = append(evaluation.AlreadyInstalled, AppVersion{App: subject.App, Version: subject.Version.Canonical()})
					} else if max := status.Versions.Max(); datatype.CompareVersions(subject.Version, max) > 0 {
						evaluation.ToInstall = append(evaluation.ToInstall, subject)
						evaluation.Outdated = append(evaluation.Outdated, AppVersion{App: subject.App, Version: max})
					} else {
						evaluation.Superseded = append(evaluation.Superseded, AppVersion{App: subject.App, Version: max})
					}
				}
			}
		} else {
			evaluation.ToInstall = append(evaluation.ToInstall, subject)
			evaluation.Missing = append(evaluation.Missing, subject)
		}
	}
	return
}

// EvaluateAppRemoval evaluates the potential removal of any applications
// matching the given criteria, according to the status map.
func (m AppStatusMap) EvaluateAppRemoval(list AppCriteriaList) (evaluation AppRemovalEvaluation) {
	for _, criteria := range list {
		if status, ok := m[criteria.App]; ok {
			if !status.Installed {
				evaluation.Missing = append(evaluation.Missing, AppVersion{App: criteria.App, Version: criteria.Version})
				continue
			}
			if len(status.Versions) > 0 {
				for _, version := range status.Versions.List() {
					if criteria.Matches(AppVersion{App: criteria.App, Version: version}) {
						evaluation.ToUninstall = append(evaluation.ToUninstall, AppVersion{App: criteria.App, Version: version})
					}
				}
			} else {
				if criteria.Matches(AppVersion{App: criteria.App}) {
					evaluation.ToUninstall = append(evaluation.ToUninstall, AppVersion{App: criteria.App})
				}
			}
		} else {
			evaluation.Missing = append(evaluation.Missing, AppVersion{App: criteria.App, Version: criteria.Version})
		}
	}
	return
}

// AppStatus reports the installation status of an application.
//
// If an the app's definition contains enough information to detect installed
// versions, it will also include a list of versions that are currently
// installed.
type AppStatus struct {
	Installed bool                `json:"installed"`
	Versions  datatype.VersionSet `json:"versions,omitempty"`
}

// AppEvaluation is an evaluation of potential changes to the set of installed
// applications.
type AppEvaluation struct {
	Status       AppStatusMap
	Installation AppInstallationEvaluation
	Removal      AppRemovalEvaluation
}

// IsZero returns true if the app evaluation is empty.
func (e AppEvaluation) IsZero() bool {
	if len(e.Status) > 0 {
		return false
	}
	if !e.Installation.IsZero() {
		return false
	}
	if !e.Removal.IsZero() {
		return false
	}
	return true
}

// ActionsNeeded returns true if any apps need to be installed or uninstalled.
func (e AppEvaluation) ActionsNeeded(mode CommandMode) bool {
	switch mode {
	case CommandModeUpdate:
		if len(e.Installation.ToInstall) > 0 && len(e.Installation.Outdated) > 0 {
			return true
		}
	default:
		if len(e.Installation.ToInstall) > 0 {
			return true
		}
		if len(e.Removal.ToUninstall) > 0 {
			return true
		}
	}
	return false
}

// AppEvaluation is an evaluation of potential changes to the set of installed
// applications.
type AppInstallationEvaluation struct {
	ToInstall        AppList
	AlreadyInstalled AppList
	Superseded       AppList
	Outdated         AppList
	Missing          AppList
}

// IsZero returns true if the app installation evaluation is empty.
func (e AppInstallationEvaluation) IsZero() bool {
	if len(e.ToInstall) > 0 {
		return false
	}
	if len(e.AlreadyInstalled) > 0 {
		return false
	}
	if len(e.Superseded) > 0 {
		return false
	}
	if len(e.Outdated) > 0 {
		return false
	}
	if len(e.Missing) > 0 {
		return false
	}
	return true
}

// AppEvaluation is an evaluation of potential changes to the set of installed
// applications.
type AppRemovalEvaluation struct {
	ToUninstall AppList
	Missing     AppList
}

// IsZero returns true if the app removal evaluation is empty.
func (e AppRemovalEvaluation) IsZero() bool {
	if len(e.ToUninstall) > 0 {
		return false
	}
	if len(e.Missing) > 0 {
		return false
	}
	return true
}

// AppSummary is a summary of changes to the set of installed applications.
type AppSummary struct {
	Installation AppInstallationSummary
	Removal      AppRemovalSummary
}

// IsZero returns true if the app summary is empty.
func (s AppSummary) IsZero() bool {
	if !s.Installation.IsZero() {
		return false
	}
	if !s.Removal.IsZero() {
		return false
	}
	return true
}

// Err returns a non-nil error if any of the expected application changes did
// not take effect.
func (s AppSummary) Err() error {
	switch {
	case len(s.Installation.StillNotInstalled) > 0 && len(s.Removal.StillNotUninstalled) > 0:
		return fmt.Errorf("some applications were not installed (%s) and some applications were not uninstalled (%s)", s.Installation.StillNotInstalled, s.Removal.StillNotUninstalled)
	case len(s.Installation.StillNotInstalled) > 0:
		return fmt.Errorf("the following applications were not installed properly: %s", s.Installation.StillNotInstalled)
	case len(s.Removal.StillNotUninstalled) > 0:
		return fmt.Errorf("the following applications were not uninstalled properly: %s", s.Removal.StillNotUninstalled)
	default:
		return nil
	}
}

// AppInstallationSummary records the results of an attempt to install
// applications.
type AppInstallationSummary struct {
	Installed         AppList
	StillNotInstalled AppList
}

// IsZero returns true if the app installation summary records no activity.
func (s AppInstallationSummary) IsZero() bool {
	if len(s.Installed) > 0 {
		return false
	}
	if len(s.StillNotInstalled) > 0 {
		return false
	}
	return true
}

// AppRemovalSummary records the results of an attempt to uninstall
// applications.
type AppRemovalSummary struct {
	Uninstalled         AppList
	StillNotUninstalled AppList
}

// IsZero returns true if the app removal summary records no activity.
func (s AppRemovalSummary) IsZero() bool {
	if len(s.Uninstalled) > 0 {
		return false
	}
	if len(s.StillNotUninstalled) > 0 {
		return false
	}
	return true
}

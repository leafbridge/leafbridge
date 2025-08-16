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

// AppList is a list of relevant applications for a deployment.
type AppList []AppID

// Difference returns all members of list that are not members of other.
func (list AppList) Difference(other AppList) AppList {
	lookup := make(map[AppID]struct{}, len(other))
	for _, app := range other {
		lookup[app] = struct{}{}
	}
	var diff AppList
	for _, app := range list {
		if _, excepted := lookup[app]; !excepted {
			diff = append(diff, app)
		}
	}
	return diff
}

// String returns a string represenation of the list.
func (list AppList) String() string {
	var out strings.Builder
	for i, item := range list {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(string(item))
	}
	return out.String()
}

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
	Detection    AppDetection    `json:"detection,omitempty"`
	Tags         AppReleaseTags  `json:"tags,omitempty"`
	Releases     AppReleaseList  `json:"releases,omitempty"`
}

// AppDetection describes how to detect the presence of an installed
// application and how to determine what version is installed.
type AppDetection struct {
	Present ConditionID             `json:"present,omitempty"`
	Version RegistryValueResourceID `json:"version,omitempty"`
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
	Name    string           `json:"name,omitempty"`
	Version datatype.Version `json:"version,omitempty"`
	Date    string           `json:"date,omitzero"`
}

// AppEvaluation is an evaluation of potential changes to the set of installed
// applications.
type AppEvaluation struct {
	AlreadyInstalled   AppList
	AlreadyUninstalled AppList
	ToInstall          AppList
	ToUninstall        AppList
}

// IsZero returns true if the app evaluation is empty.
func (e AppEvaluation) IsZero() bool {
	if len(e.AlreadyInstalled) > 0 {
		return false
	}
	if len(e.AlreadyUninstalled) > 0 {
		return false
	}
	if len(e.ToInstall) > 0 {
		return false
	}
	if len(e.ToUninstall) > 0 {
		return false
	}
	return true
}

// ActionsNeeded returns true if any apps need to be installed or uninstalled.
func (e AppEvaluation) ActionsNeeded() bool {
	if len(e.ToInstall) > 0 {
		return true
	}
	if len(e.ToUninstall) > 0 {
		return true
	}
	return false
}

// AppSummary is a summary of changes to the set of installed applications.
type AppSummary struct {
	Installed           AppList
	Uninstalled         AppList
	StillNotInstalled   AppList
	StillNotUninstalled AppList
}

// IsZero returns true if the app summary is empty.
func (s AppSummary) IsZero() bool {
	if len(s.Installed) > 0 {
		return false
	}
	if len(s.Uninstalled) > 0 {
		return false
	}
	if len(s.StillNotInstalled) > 0 {
		return false
	}
	if len(s.StillNotUninstalled) > 0 {
		return false
	}

	return true
}

// Err returns a non-nil error if any of the expected application changes did
// not take effect.
func (s AppSummary) Err() error {
	switch {
	case len(s.StillNotInstalled) > 0 && len(s.StillNotUninstalled) > 0:
		return fmt.Errorf("some applications were not installed (%s) and some applications were not uninstalled (%s)", s.StillNotInstalled, s.StillNotUninstalled)
	case len(s.StillNotInstalled) > 0:
		return fmt.Errorf("the following applications were not installed properly: %s", s.StillNotInstalled)
	case len(s.StillNotUninstalled) > 0:
		return fmt.Errorf("the following applications were not uninstalled properly: %s", s.StillNotUninstalled)
	default:
		return nil
	}
}

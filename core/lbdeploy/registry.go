package lbdeploy

import (
	"fmt"
	"path/filepath"

	"github.com/leafbridge/leafbridge/core/idset"
	"github.com/leafbridge/leafbridge/core/lbvalue"
)

// RegistryResources describes resources accessed through the Windows
// registry.
type RegistryResources struct {
	Keys   RegistryKeyResourceMap   `json:"keys,omitempty"`
	Values RegistryValueResourceMap `json:"values,omitempty"`
}

// RegistryKeyResourceMap holds a set of registry key resources mapped by
// their identifiers.
type RegistryKeyResourceMap map[RegistryKeyResourceID]RegistryKeyResource

// RegistryKeyResourceID is a unique identifier for a registry key.
type RegistryKeyResourceID string

// RegistryKeyResource describes a registry key in the Windows registry.
//
// Its name and path fields are mutually exclusive.
type RegistryKeyResource struct {
	// Location is a well-known registry root ID, or another key's
	// resource ID.
	Location RegistryKeyResourceID `json:"location,omitempty"`

	// View identifies a registry view to use when accessing the registry. It
	// is only applied if Location is a well-known registry root ID.
	View RegistryView `json:"view,omitempty"`

	// Name is the name of the key within its location.
	Name string `json:"name,omitempty"`

	// Path is the relative path of the key within its location.
	// Both forward slashes and backslashes will be interpreted as path
	// separators.
	Path string `json:"path,omitempty"`
}

// RegistryKeyRef is a resolved reference to a registry key on the local
// system.
type RegistryKeyRef struct {
	Root    RegistryRoot
	Lineage []RegistryKeyResource
}

// Path returns the path of the registry key on the local system.
func (ref RegistryKeyRef) Path() (string, error) {
	path, err := ref.Root.AbsolutePath()
	if err != nil {
		return "", err
	}

	for _, key := range ref.Lineage {
		switch {
		case key.Name != "":
			path = path + `\` + key.Name
		case key.Path != "":
			localized, err := filepath.Localize(key.Path)
			if err != nil {
				return "", err
			}
			path = filepath.Join(path, localized)
		default:
			return "", fmt.Errorf("a registry key resource does not specify a name or path")
		}
	}

	return path, nil
}

// RegistryKeyResourceSet holds a set of registry key resource IDs.
type RegistryKeyResourceSet = idset.SetOf[RegistryKeyResourceID]

// RegistryValueResourceMap holds a set of registry value resources mapped by
// their identifiers.
type RegistryValueResourceMap map[RegistryValueResourceID]RegistryValueResource

// RegistryValueResourceID is a unique identifier for a registry value.
type RegistryValueResourceID string

// RegistryValueResource describes a value within the Windows registry.
type RegistryValueResource struct {
	// Key is the registry key resource ID of the key to which the value
	// belongs, or the well-known resource ID of a registry root.
	Key RegistryKeyResourceID `json:"key"`

	// Name is the name of the value within its registry key.
	Name string `json:"name"`

	// Type is the type of data the value holds.
	Type lbvalue.Kind `json:"type"`
}

// RegistryValueRef is a resolved reference to a registry key on the local
// system.
type RegistryValueRef struct {
	Root    RegistryRoot
	Lineage []RegistryKeyResource
	ID      RegistryValueResourceID
	Name    string
	Type    lbvalue.Kind
}

// Key returns a reference to the values's registry key.
func (ref RegistryValueRef) Key() RegistryKeyRef {
	return RegistryKeyRef{
		Root:    ref.Root,
		Lineage: ref.Lineage,
	}
}

// RegistryView identifies an [Alternate Registry View] of the Windows
// registry. It can be unspecified, which will use a view that is consistent
// with the calling application's architecture, or it can specify a 32-bit or
// 64-bit view of the registry.
//
// When marshaled as a string, the specified values are marshaled as "32-bit"
// and "64-bit" respectively.
//
// [Alternate Registry View]: https://learn.microsoft.com/en-us/windows/win32/winprog64/accessing-an-alternate-registry-view
type RegistryView int

// Possible views of the Windows registry.
const (
	RegistryViewUnspecified RegistryView = iota
	RegistryView32Bit
	RegistryView64Bit
)

// UnmarshalText attempts to unmarshal the given text into view.
func (view *RegistryView) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*view = RegistryViewUnspecified
	case "32-bit":
		*view = RegistryView32Bit
	case "64-bit":
		*view = RegistryView64Bit
	default:
		return fmt.Errorf("unrecognized registry view: %s", b)
	}
	return nil
}

// MarshalText marshals the registry view as text.
func (view RegistryView) MarshalText() ([]byte, error) {
	switch view {
	case RegistryView32Bit:
		return []byte("32-bit"), nil
	case RegistryView64Bit:
		return []byte("64-bit"), nil
	case RegistryViewUnspecified:
		return nil, nil
	}
	return nil, fmt.Errorf("unrecognized registry view: %d", view)
}

// String returns a string representation of the registry view.
func (view RegistryView) String() string {
	switch view {
	case RegistryView32Bit:
		return "32-bit"
	case RegistryView64Bit:
		return "64-bit"
	}
	return ""
}

// RegistryRoot is a root location within the Windows registry.
type RegistryRoot struct {
	ID            RegistryKeyResourceID
	PredefinedKey PredefinedRegistryKey
	View          RegistryView
	Path          string
}

// AbsolutePath returns the absolute path to the registry root on the
// local system, including the predefined key.
func (root RegistryRoot) AbsolutePath() (path string, err error) {
	path = root.PredefinedKey.String()
	if root.Path != "" {
		path = filepath.Join(path, root.Path)
	}
	return
}

// PredefinedRegistryKey identifies a predefined key within the Windows
// registry.
type PredefinedRegistryKey int

// Predefined keys within the Windows registry that are recognized by
// LeafBridge.
const (
	PredefinedKeyUnknown PredefinedRegistryKey = iota
	PredefinedKeyLocalMachine
)

var predefinedRegistryKeyStrings = []string{
	"HKEY_UNKNOWN",
	"HKEY_LOCAL_MACHINE",
}

// String returns a string representation of the key in its canonical form,
// such as HKEY_LOCAL_MACHINE.
func (key PredefinedRegistryKey) String() string {
	if key := int(key); key >= 0 && key < len(predefinedRegistryKeyStrings) {
		return predefinedRegistryKeyStrings[key]
	}
	return fmt.Sprintf("<unknown registry key \"%d\">", key)
}

// UnmarshalText attempts to unmarshal the given text into key.
func (key *PredefinedRegistryKey) UnmarshalText(b []byte) error {
	switch string(b) {
	case "HKEY_UNKNOWN":
		*key = PredefinedKeyUnknown
	case "HKEY_LOCAL_MACHINE":
		*key = PredefinedKeyLocalMachine
	default:
		return fmt.Errorf("unrecognized or unsupported registry key: %s", b)
	}
	return nil
}

// MarshalText marshals the key as text.
func (key PredefinedRegistryKey) MarshalText() ([]byte, error) {
	if key := int(key); key >= 0 && key < len(predefinedRegistryKeyStrings) {
		return []byte(predefinedRegistryKeyStrings[key]), nil
	}
	return nil, fmt.Errorf("unrecognized or unsupported registry key: %d", key)
}

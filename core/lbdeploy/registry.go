package lbdeploy

import (
	"fmt"
	"path/filepath"
	"strings"

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

// LogicalPath returns the logical path of the registry key on the local
// system, before it has been redirected to a physical path by the
// [Registry Redirector].
//
// [Registry Redirector]: https://learn.microsoft.com/en-us/windows/win32/winprog64/registry-redirector
func (ref RegistryKeyRef) LogicalPath() (string, error) {
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

// PhysicalPath returns the physical path of the registry key on the local
// system, after it has been redirected by the [Registry Redirector].
//
// This function mimics the actions of the redirector in order to generate
// the physical path. It is not guaranteed to be accurate and should be
// considered a best effort. See the official documentation on
// [Shared Registry Keys] for more details.
//
// [Registry Redirector]: https://learn.microsoft.com/en-us/windows/win32/winprog64/registry-redirector
// [Shared Registry Keys]: https://learn.microsoft.com/en-us/windows/win32/winprog64/shared-registry-keys
func (ref RegistryKeyRef) PhysicalPath() (string, error) {
	path, err := ref.LogicalPath()
	if err != nil {
		return "", err
	}

	return RegistryKeyPhysicalPath(path, ref.Root.View), nil
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
//
// The returned path is a logical path before it has been redirected to
// a physical path by the [Registry Redirector].
//
// [Registry Redirector]: https://learn.microsoft.com/en-us/windows/win32/winprog64/registry-redirector
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

// RegistryKeyPhysicalPath returns the physical path of a registry key after
// it has been redirected by the [Registry Redirector].
//
// This function mimics the actions of the redirector in order to generate
// the physical path. It is not guaranteed to be accurate and should be
// considered a best effort. See the official documentation on
// [Shared Registry Keys] for more details.
//
// [Registry Redirector]: https://learn.microsoft.com/en-us/windows/win32/winprog64/registry-redirector
// [Shared Registry Keys]: https://learn.microsoft.com/en-us/windows/win32/winprog64/shared-registry-keys
func RegistryKeyPhysicalPath(path string, view RegistryView) string {
	// TODO: Consider improving the efficiency of this code.
	switch view {
	case RegistryView32Bit:
		// https://learn.microsoft.com/en-us/windows/win32/winprog64/shared-registry-keys
		redirections := []struct {
			From string
			To   string
		}{
			{From: `HKEY_LOCAL_MACHINE`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE`, To: `HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node`},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Classes`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Classes\Appid`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Classes\CLSID`, To: `HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\Classes\CLSID`},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Classes\DirectShow`, To: `HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\Classes\DirectShow`},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Classes\HCP`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Classes\Interface`, To: `HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\Classes\Interface`},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Classes\Media Type`, To: `HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\Classes\Media Type`},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Classes\MediaFoundation`, To: `HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\Classes\MediaFoundation`},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Clients`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\COM3`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography\Calais\Current`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography\Calais\Readers`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography\Services`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\CTF\SystemShared`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\CTF\TIP`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\DFS`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Driver Signing`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\EnterpriseCertificates`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\EventSystem`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\MSMQ`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Non-Driver Signing`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Notepad\DefaultFonts`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\OLE`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\RAS`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\RPC`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SOFTWARE\Microsoft\Shared Tools\MSInfo`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SystemCertificates`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\TermServLicensing`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\TransactionServer`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Control Panel\Cursors\Schemes`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\AutoplayHandlers`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\DriveIcons`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\KindMap`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Group Policy`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\PreviewHandlers`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Setup`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Telephony\Locations`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Console`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\FontDpi`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\FontLink`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\FontMapper`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\FontSubstitutes`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Gre_Initialize`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Language Pack`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\NetworkCards`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Perflib`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Ports`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Print`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Time Zones`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\Policies`, To: ``},
			{From: `HKEY_LOCAL_MACHINE\SOFTWARE\RegisteredApplications`, To: ``},
		}

		// Search the list from the end to the beginning so that we
		// always evaluate children before their parents.
		for i := len(redirections) - 1; i >= 0; i-- {
			if cut, matched := strings.CutPrefix(path, redirections[i].From); matched {
				if redirections[i].To == "" {
					return path
				}
				return redirections[i].To + cut
			}
		}
	}

	return path
}

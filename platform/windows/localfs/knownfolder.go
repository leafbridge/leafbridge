package localfs

import (
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"golang.org/x/sys/windows"
)

// knownFolderMap is a map of predefined directory resource IDs to known
// folder GUIDs and properties.
type knownFolderMap map[lbdeploy.DirectoryResourceID]knownFolder

// knownFolder holds the GUID and properties for a known folder in Windows.
type knownFolder struct {
	guid      *windows.KNOWNFOLDERID
	protected bool
}

// Known folders that are recognized by their resource IDs.
var knownFolders = knownFolderMap{
	"common-start-menu": knownFolder{guid: windows.FOLDERID_CommonStartMenu},
	"public-desktop":    knownFolder{guid: windows.FOLDERID_PublicDesktop},
	"program-data":      knownFolder{guid: windows.FOLDERID_ProgramData},
	"program-files":     knownFolder{guid: windows.FOLDERID_ProgramFiles},
	"program-files-x86": knownFolder{guid: windows.FOLDERID_ProgramFilesX86},
	"program-files-x64": knownFolder{guid: windows.FOLDERID_ProgramFilesX64},
	"system":            knownFolder{guid: windows.FOLDERID_System, protected: true},
}

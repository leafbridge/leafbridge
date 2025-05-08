//go:build generate

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/josephspurrier/goversioninfo"
	"github.com/leafbridge/leafbridge/internal/buildinfo"
)

func main() {
	buildVersionInfo()
}

func buildVersionInfo() {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("leafbridge-deploy build information is not available")
		os.Exit(1)
	}

	version := buildinfo.GetVersion(buildInfo)

	fileVersion := goversioninfo.FileVersion{
		Major: version.Major(),
		Minor: version.Minor(),
		Patch: version.Patch(),
		Build: version.Build(),
	}
	vi := goversioninfo.VersionInfo{
		//IconPath: "icon.ico",
		FixedFileInfo: goversioninfo.FixedFileInfo{
			FileVersion:    fileVersion,
			ProductVersion: fileVersion,
			FileFlagsMask:  "3f",
			FileFlags:      "00",
			FileOS:         "040004",
			FileType:       "01",
			FileSubType:    "00",
		},
		StringFileInfo: goversioninfo.StringFileInfo{
			CompanyName:      "LeafBridge",
			FileDescription:  "LeafBridge Software Deployment Utility",
			FileVersion:      string(version),
			OriginalFilename: "leafbridge-deploy.exe",
			ProductName:      "LeafBridge",
			ProductVersion:   string(version),
		},
		VarFileInfo: goversioninfo.VarFileInfo{
			Translation: goversioninfo.Translation{
				LangID:    goversioninfo.LngUSEnglish,
				CharsetID: goversioninfo.CsUnicode,
			},
		},
	}
	vi.Build()
	vi.Walk()
	vi.WriteSyso("leafbridge-deploy.syso", runtime.GOARCH)
}

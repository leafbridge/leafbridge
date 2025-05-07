package lbengine

import (
	"github.com/leafbridge/leafbridge/core/idset"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/platform/windows/stagingfs"
	"github.com/leafbridge/leafbridge/platform/windows/tempfs"
)

// engineState keeps track of the overall state of an flow.
type engineState struct {
	activeFlows          flowSet
	verifiedPackageFiles map[lbdeploy.PackageID]stagingfs.PackageDir
	extractedPackages    map[lbdeploy.PackageID]tempfs.ExtractionDir
	locks                *lockManager
}

func newEngineState() *engineState {
	return &engineState{
		activeFlows:          make(flowSet),
		verifiedPackageFiles: make(map[lbdeploy.PackageID]stagingfs.PackageDir),
		extractedPackages:    make(map[lbdeploy.PackageID]tempfs.ExtractionDir),
		locks:                newLockManager(),
	}
}

// flowSet keeps track of a set of flows.
type flowSet = idset.SetOf[lbdeploy.FlowID]

package stagingfs

import (
	"os"
	"path/filepath"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"golang.org/x/sys/windows"
)

// File path constants.
const (
	RootDir    = "LeafBridge"
	StagingDir = "Deploy"
)

// DeploymentDir is a staging directory for a deployment in LeafBridge.
type DeploymentDir struct {
	deployment lbdeploy.DeploymentID
	path       string
	dir        *os.Root
}

// OpenDeployment opens the staging directory for a deployment in LeafBridge.
// If the directory does not already exist, it is created.
//
// It is the caller's responsibility to close the directory when finished
// with it.
func OpenDeployment(id lbdeploy.DeploymentID) (DeploymentDir, error) {
	// Look up the system's ProgramData directory path.
	programDataPath, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil {
		return DeploymentDir{}, err
	}

	// Open the ProgramData directory.
	programData, err := os.OpenRoot(programDataPath)
	if err != nil {
		return DeploymentDir{}, err
	}
	defer programData.Close()

	// Open the ProgramData/LeafBridge directory.
	root, err := openOrCreateRootInRoot(programData, RootDir, 0755)
	if err != nil {
		return DeploymentDir{}, err
	}
	defer root.Close()

	// Open the ProgramData/LeafBridge/Deploy directory.
	staging, err := openOrCreateRootInRoot(root, StagingDir, 0755)
	if err != nil {
		return DeploymentDir{}, err
	}
	defer staging.Close()

	// Open the ProgramData/LeafBridge/Deploy/{DeploymentID} directory.
	dir, err := openOrCreateRootInRoot(staging, string(id), 0755)
	if err != nil {
		return DeploymentDir{}, err
	}

	return DeploymentDir{
		deployment: id,
		path:       filepath.Join(programDataPath, RootDir, StagingDir, string(id)),
		dir:        dir,
	}, nil
}

// OpenPackage opens the staging directory for the given package content.
// If the directory does not already exist, it is created.
//
// It is the caller's responsibility to close the directory when finished
// with it.
func (r DeploymentDir) OpenPackage(content lbdeploy.PackageContent) (PackageDir, error) {
	dir, err := openOrCreateRootInRoot(r.dir, content.String(), 0755)
	if err != nil {
		return PackageDir{}, err
	}
	return PackageDir{
		content: content,
		path:    filepath.Join(r.path, content.String()),
		dir:     dir,
	}, nil
}

// Close releases any file handles or resources held by the deployment
// staging directory.
func (r DeploymentDir) Close() error {
	return r.dir.Close()
}

func openOrCreateRootInRoot(parent *os.Root, name string, perm os.FileMode) (*os.Root, error) {
	// Attempt to open an existing directory.
	child, err := parent.OpenRoot(name)
	if err == nil {
		return child, nil
	}

	// If the error is anything other than "not found", return it.
	if !os.IsNotExist(err) {
		return nil, err
	}

	// Attempt to create the directory.
	if err := parent.Mkdir(name, perm); err != nil {
		return nil, err
	}

	// Attempt to open the directory a second time.
	return parent.OpenRoot(name)
}

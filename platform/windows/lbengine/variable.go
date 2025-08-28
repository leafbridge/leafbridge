package lbengine

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/gentlemanautomaton/portableexecutable"
	"github.com/gentlemanautomaton/portableexecutable/imagefile"
	"github.com/gentlemanautomaton/portableexecutable/tables/resourcedirectory"
	"github.com/gentlemanautomaton/portableexecutable/tables/resourcedirectory/resourcetype"
	"github.com/gentlemanautomaton/portableexecutable/tables/resourcedirectory/resourcetype/versioninfo"
	"github.com/leafbridge/leafbridge/core/datatype"
	"github.com/leafbridge/leafbridge/core/idset"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbvalue"
	"github.com/leafbridge/leafbridge/platform/windows/localfs"
	"github.com/leafbridge/leafbridge/platform/windows/localregistry"
)

// variableSet keeps track of a set of variables as they are evaluated.
type variableSet = idset.SetOf[lbdeploy.VariableID]

// VariableEngine is responsible for evaluating variables on the local
// system.
type VariableEngine struct {
	deployment lbdeploy.Deployment
}

// NewVariableEngine prepares a variable engine for the given deployment.
func NewVariableEngine(dep lbdeploy.Deployment) VariableEngine {
	return VariableEngine{
		deployment: dep,
	}
}

// Value returns the current value of the requested variable.
func (engine VariableEngine) Value(variable lbdeploy.VariableID) (lbvalue.Value, error) {
	// Find the variable within the deployment.
	definition, found := engine.deployment.Variables[variable]
	if !found {
		return lbvalue.Value{}, fmt.Errorf("the \"%s\" variable does not exist within the \"%s\" deployment", variable, engine.deployment.ID)
	}

	return engine.evaluate(variable, definition, make(lbdeploy.VariableCache), make(variableSet))
}

func (engine VariableEngine) evaluate(id lbdeploy.VariableID, variable lbdeploy.Variable, cache lbdeploy.VariableCache, seen variableSet) (lbvalue.Value, error) {
	// Special handling for variables that are identified.
	if id != "" {
		// If this variable has already been evaluated, return the cached
		// value.
		if value, computed := cache[id]; computed {
			return value, nil
		}

		// Check for recursive calls.
		if seen.Contains(id) {
			return lbvalue.Value{}, fmt.Errorf("the \"%s\" variable is recursive and is already being evaluated", id)
		}

		// Add this variable to the evaluation set, then remove it when we're
		// finished.
		seen.Add(id)
		defer seen.Remove(id)
	}

	// Retrieve or calculate the variable's value.
	result, err := func() (lbvalue.Value, error) {
		switch variable.Source {
		case lbdeploy.VariableSourceRegistryValue:
			resolver := localregistry.NewResolver(engine.deployment.Resources.Registry)
			ref, err := resolver.ResolveValue(lbdeploy.RegistryValueResourceID(variable.Subject))
			if err != nil {
				return lbvalue.Value{}, err
			}
			key, err := localregistry.OpenKey(ref.Key())
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return lbvalue.Value{}, nil
				}
				return lbvalue.Value{}, err
			}
			return key.GetValue(ref.Name, ref.Type)
		case lbdeploy.VariableSourceFileVersion, lbdeploy.VariableSourceProductVersion:
			// Resolve and open the file.
			fileID := lbdeploy.FileResourceID(variable.Subject)
			resolver := localfs.NewResolver(engine.deployment.Resources.FileSystem)
			ref, err := resolver.ResolveFile(fileID)
			if err != nil {
				return lbvalue.Value{}, err
			}
			file, err := localfs.OpenFile(ref)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return lbvalue.Version(""), nil
				}
				path, _ := ref.Path()
				return lbvalue.Value{}, fileVersionError(fileID, path, err)
			}
			defer file.Close()

			// Make sure the file is a regular file.
			fi, err := file.System().Stat()
			if err != nil {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), err)
			}
			if !fi.Mode().IsRegular() {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), errors.New("the path exists but it is not a regular file"))
			}

			// Create a portable executable reader.
			pe, err := portableexecutable.NewReader(file.System())
			if err != nil {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), fmt.Errorf("the file could not be interpreted as a portable executable: %w", err))
			}

			// Look for a resource table.
			resources := pe.DataDirectories().Get(imagefile.ResourceTableID)
			if resources.IsZero() {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), errors.New("the file does not contain a resource directory"))
			}

			// Create a resource directory reader.
			resdir, err := resourcedirectory.NewReader(pe)
			if err != nil {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), fmt.Errorf("the file's resource directory could not be opened: %w", err))
			}

			// Ask the resource directory for version resources.
			versions, err := resdir.ReadType(resourcetype.Version)
			if err != nil {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), fmt.Errorf("the file's resource directory could not be queried: %w", err))
			}

			// Use the first version resource in the list.
			// TODO: Consider using a more intelligent selection process.
			if len(versions) == 0 || !versions[0].Reference.IsTable() {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), errors.New("the file's resource directory does not contain file version information"))
			}

			// Get the table of supported languages for this version.
			languages, err := resdir.ReadTable(versions[0].Reference.Table())
			if err != nil {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), fmt.Errorf("the file's version information language table could not be queried: %w", err))
			}

			// Either use language code 1033 (en-us) or use the first entry
			// in the language list.
			index := max(versions.Index(resourcedirectory.NewNumericID(1033)), 0)
			if len(languages) == 0 || languages[index].Reference.IsTable() {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), errors.New("the file's resource directory does not contain file version information"))
			}

			// Pull the file version information into memory.
			versionData, err := resdir.ReadData(languages[index].Reference.Data())
			if err != nil {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), fmt.Errorf("the file's version information data could not be read: %w", err))
			}

			// Search the file version data for suitable file and product
			// versions.
			fileVersion, productVersion, err := getFileVersionFromInfo(versionData)
			if err != nil {
				return lbvalue.Value{}, fileVersionError(fileID, file.Path(), fmt.Errorf("the file's version information data could not be parsed: %w", err))
			}

			// Return the requested value.
			if variable.Source == lbdeploy.VariableSourceFileVersion {
				return lbvalue.Version(fileVersion), nil
			}
			return lbvalue.Version(productVersion), nil
		default:
			return lbvalue.Value{}, fmt.Errorf("unrecognized variable source: %s", variable.Source)
		}
	}()

	// If we encountered an error, wrap it with information about the
	// variable.
	if err != nil {
		err = variableError(id, variable, err)
	}

	// Record the result in the cache if possible.
	if id != "" && err == nil {
		cache[id] = result
	}

	return result, err
}

func getFileVersionFromInfo(data []byte) (file, product datatype.Version, err error) {
	// Interpret the version info data as a root node.
	root, err := versioninfo.NewRoot(data)
	if err != nil {
		return "", "", err
	}

	// Look for versions in the fixed file info.
	info := root.FileInfo()
	if info.Valid() {
		if v := info.FileVersion(); !v.IsZero() {
			file = datatype.Version(v.String())
		}
		if v := info.ProductVersion(); !v.IsZero() {
			product = datatype.Version(v.String())
		}
	}

	// Look for versions in the string-based file info, which are preferred.
	if ok, container := getVersionInfoChild(versioninfo.Node(root), "VS_VERSION_INFO"); ok {
		if ok, fileInfo := getVersionInfoChild(container, "StringFileInfo"); ok {
			if ok, language := getVersionInfoChild(fileInfo, ""); ok {
				if ok, fv := getVersionInfoChild(language, "FileVersion"); ok {
					if value := fv.Value().String(); value != "" {
						file = datatype.Version(value)
					}
				}
				if ok, pv := getVersionInfoChild(language, "ProductVersion"); ok {
					if value := pv.Value().String(); value != "" {
						product = datatype.Version(value)
					}
				}
			}
		}
	}

	return
}

// getVersionInfoChild looks for a child with the given key and returns it.
// If key is empty, it returns the first child.
func getVersionInfoChild(parent versioninfo.Node, key string) (ok bool, node versioninfo.Node) {
	for candidate, err := range parent.Children() {
		if err != nil {
			break
		}
		if key == "" || candidate.Key() == key {
			return true, candidate
		}
	}
	return
}

func variableError(id lbdeploy.VariableID, v lbdeploy.Variable, err error) error {
	return lbdeploy.VariableError{
		ID:     id,
		Label:  v.Label,
		Source: v.Source,
		Err:    err,
	}
}

func fileVersionError(id lbdeploy.FileResourceID, path string, err error) error {
	return lbdeploy.FileVersionInfoError{
		ID:   id,
		Path: path,
		Err:  err,
	}
}

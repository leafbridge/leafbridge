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
	var (
		result lbvalue.Value
		err    error
	)
	if len(variable.Elements) > 0 {
		// Evaluate composite variables that are made up of one or more
		// subvariables.
		//
		// If criteria have been supplied for the variable, this call will
		// also apply them to each subvariable.
		result, err = engine.evaluateMultiple(id, variable, cache, seen)
	} else {
		// Evaluate individual variables.
		result, err = engine.evaluateSingle(id, variable, cache, seen)

		// If criteria have been supplied for the variable, use them to filter
		// its value.
		if err == nil && !variable.Criteria.IsZero() {
			result, err = variable.Criteria.Filter(result)
		}
	}

	// If we encountered an error, wrap it with information about the
	// variable.
	if err != nil {
		err = variableSelfError(id, variable, err)
	}

	// Record the result in the cache if possible.
	if id != "" && err == nil {
		cache[id] = result
	}

	return result, err
}

// evaluateMultiple evaluates a variable that is composed of one or more
// subvariable elements.
func (engine VariableEngine) evaluateMultiple(id lbdeploy.VariableID, variable lbdeploy.Variable, cache lbdeploy.VariableCache, seen variableSet) (lbvalue.Value, error) {
	// Collect the desired type. If it is unspecified, kind will be
	// KindUnknown.
	kind := variable.Type.Kind()

	// Special handling for version sets.
	var versions datatype.VersionSet
	if kind == lbvalue.KindVersionSet {
		versions = make(datatype.VersionSet)
	}

	// Process each subvariable element.
	for i, candidate := range variable.Elements {
		// Evaluate the subvariable.
		result, err := engine.evaluate("", candidate, cache, seen)

		// If criteria have been supplied, use them to filter subvariable
		// results. It's important that this happens before non-empty value
		// evaluation later on.
		if err == nil && !variable.Criteria.IsZero() {
			result, err = variable.Criteria.Filter(result)
		}

		// If there was an error during evaluation or during the application
		// of criteria, return it.
		if err != nil {
			return result, lbdeploy.VariableError{
				ID:      id,
				Label:   variable.Label,
				Origin:  lbdeploy.VariableErrorOriginElement,
				Element: i,
				Err:     err,
			}
		}

		// If we're building a version set, add all non-empty element
		// results to the set and then continue to the next element.
		if variable.Type.Kind() == lbvalue.KindVersionSet {
			switch result.Kind() {
			case lbvalue.KindVersion:
				version := result.Version()
				if version != "" {
					versions.Add(version)
				}
			case lbvalue.KindVersionSet:
				for version := range result.VersionSet() {
					if version != "" {
						versions.Add(version)
					}
				}
			default:
				return lbvalue.Value{}, lbdeploy.VariableError{
					ID:      id,
					Label:   variable.Label,
					Origin:  lbdeploy.VariableErrorOriginElement,
					Element: i,
					Err:     fmt.Errorf("the subvariable's result is of type \"%s\" which cannot be included in a version set", result.Kind()),
				}
			}
			continue
		}

		// If we aren't building a set, evaluate type compatibilty here.
		if kind == lbvalue.KindUnknown {
			// If the containing variable didn't define a type,
			// skip this evaluation and use the type of the
			// first result.
		} else if kind != result.Kind() {
			return lbvalue.Value{}, lbdeploy.VariableError{
				ID:      id,
				Label:   variable.Label,
				Origin:  lbdeploy.VariableErrorOriginElement,
				Element: i,
				Err:     fmt.Errorf("the subvariable's result is of type \"%s\" which is not compatible with the variable's \"%s\" type", result.Kind(), kind),
			}
		}

		// If we aren't building a set, return the first non-empty result.
		switch result.Kind() {
		case lbvalue.KindVersion:
			if result.Version() != "" {
				return result, nil
			}
		case lbvalue.KindVersionSet:
			if len(result.VersionSet()) > 0 {
				return result, nil
			}
		case lbvalue.KindUnknown:
			// Skip unknown values and go on to the next element.
		default:
			return result, nil
		}
	}

	// Special handling for version sets.
	if kind == lbvalue.KindVersionSet {
		return lbvalue.VersionSet(versions), nil
	}

	// None of the subvariables provided a value.
	return lbvalue.Zero(variable.Type.Kind()), nil
}

// evaluateSingle evaluates a single variable.
func (engine VariableEngine) evaluateSingle(id lbdeploy.VariableID, variable lbdeploy.Variable, cache lbdeploy.VariableCache, seen variableSet) (lbvalue.Value, error) {
	switch variable.Source {
	case lbdeploy.VariableSourceSubvariable:
		candidateID := lbdeploy.VariableID(variable.Subject)
		candidate, found := engine.deployment.Variables[candidateID]
		if !found {
			return lbvalue.Value{}, variableSelfError(id, variable, fmt.Errorf("the \"%s\" variable is not defined in the deployment", variable.Subject))
		}
		return engine.evaluate(candidateID, candidate, cache, seen)
	case lbdeploy.VariableSourceRegistryKeyValueNames:
		if variable.Type.Kind() != lbvalue.KindVersionSet {
			return lbvalue.Value{}, fmt.Errorf("only variables of type %s are currently supported when the variable source is %s, and the variable type is %s", lbvalue.KindVersionSet, variable.Source, variable.Type.Kind())
		}
		empty := lbvalue.Zero(lbvalue.KindVersionSet)
		resolver := localregistry.NewResolver(engine.deployment.Resources.Registry)
		ref, err := resolver.ResolveKey(lbdeploy.RegistryKeyResourceID(variable.Subject))
		if err != nil {
			return empty, err
		}
		key, err := localregistry.OpenKey(ref)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return empty, nil
			}
			return empty, err
		}
		defer key.Close()
		names, err := key.ReadValueNames()
		if err != nil {
			return empty, err
		}
		versions := make(datatype.VersionSet, len(names))
		for _, name := range names {
			versions.Add(datatype.Version(name))
		}
		return lbvalue.VersionSet(versions), nil
	case lbdeploy.VariableSourceRegistryValue:
		resolver := localregistry.NewResolver(engine.deployment.Resources.Registry)
		ref, err := resolver.ResolveValue(lbdeploy.RegistryValueResourceID(variable.Subject))
		if err != nil {
			return lbvalue.Value{}, err
		}
		key, err := localregistry.OpenKey(ref.Key())
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return lbvalue.Zero(ref.Type), nil
			}
			return lbvalue.Zero(ref.Type), err
		}
		defer key.Close()
		return key.GetValue(ref.Name, ref.Type)
	case lbdeploy.VariableSourceFileVersion, lbdeploy.VariableSourceProductVersion:
		// Resolve and open the file.
		none := lbvalue.Zero(lbvalue.KindVersion)
		fileID := lbdeploy.FileResourceID(variable.Subject)
		resolver := localfs.NewResolver(engine.deployment.Resources.FileSystem)
		ref, err := resolver.ResolveFile(fileID)
		if err != nil {
			return none, err
		}
		file, err := localfs.OpenFile(ref)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return none, nil
			}
			path, _ := ref.Path()
			return none, fileVersionError(fileID, path, err)
		}
		defer file.Close()

		// Make sure the file is a regular file.
		fi, err := file.System().Stat()
		if err != nil {
			return none, fileVersionError(fileID, file.Path(), err)
		}
		if !fi.Mode().IsRegular() {
			return none, fileVersionError(fileID, file.Path(), errors.New("the path exists but it is not a regular file"))
		}

		// Create a portable executable reader.
		pe, err := portableexecutable.NewReader(file.System())
		if err != nil {
			return none, fileVersionError(fileID, file.Path(), fmt.Errorf("the file could not be interpreted as a portable executable: %w", err))
		}

		// Look for a resource table.
		resources := pe.DataDirectories().Get(imagefile.ResourceTableID)
		if resources.IsZero() {
			return none, fileVersionError(fileID, file.Path(), errors.New("the file does not contain a resource directory"))
		}

		// Create a resource directory reader.
		resdir, err := resourcedirectory.NewReader(pe)
		if err != nil {
			return none, fileVersionError(fileID, file.Path(), fmt.Errorf("the file's resource directory could not be opened: %w", err))
		}

		// Ask the resource directory for version resources.
		versions, err := resdir.ReadType(resourcetype.Version)
		if err != nil {
			return none, fileVersionError(fileID, file.Path(), fmt.Errorf("the file's resource directory could not be queried: %w", err))
		}

		// Use the first version resource in the list.
		// TODO: Consider using a more intelligent selection process.
		if len(versions) == 0 || !versions[0].Reference.IsTable() {
			return none, fileVersionError(fileID, file.Path(), errors.New("the file's resource directory does not contain file version information"))
		}

		// Get the table of supported languages for this version.
		languages, err := resdir.ReadTable(versions[0].Reference.Table())
		if err != nil {
			return none, fileVersionError(fileID, file.Path(), fmt.Errorf("the file's version information language table could not be queried: %w", err))
		}

		// Either use language code 1033 (en-us) or use the first entry
		// in the language list.
		index := max(versions.Index(resourcedirectory.NewNumericID(1033)), 0)
		if len(languages) == 0 || languages[index].Reference.IsTable() {
			return none, fileVersionError(fileID, file.Path(), errors.New("the file's resource directory does not contain file version information"))
		}

		// Pull the file version information into memory.
		versionData, err := resdir.ReadData(languages[index].Reference.Data())
		if err != nil {
			return none, fileVersionError(fileID, file.Path(), fmt.Errorf("the file's version information data could not be read: %w", err))
		}

		// Search the file version data for suitable file and product
		// versions.
		fileVersion, productVersion, err := getFileVersionFromInfo(versionData)
		if err != nil {
			return none, fileVersionError(fileID, file.Path(), fmt.Errorf("the file's version information data could not be parsed: %w", err))
		}

		// Return the requested value.
		if variable.Source == lbdeploy.VariableSourceFileVersion {
			return lbvalue.Version(fileVersion), nil
		}
		return lbvalue.Version(productVersion), nil
	default:
		return lbvalue.Value{}, fmt.Errorf("unrecognized variable source: %s", variable.Source)
	}
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

func variableSelfError(id lbdeploy.VariableID, v lbdeploy.Variable, err error) error {
	return lbdeploy.VariableError{
		ID:     id,
		Label:  v.Label,
		Source: v.Source,
		Origin: lbdeploy.VariableErrorOriginSelf,
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

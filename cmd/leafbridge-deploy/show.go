package main

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/gentlemanautomaton/winobj/winmutex"
	"github.com/leafbridge/leafbridge/platform/windows/lbengine"
	"github.com/leafbridge/leafbridge/platform/windows/localfs"
	"github.com/leafbridge/leafbridge/platform/windows/localregistry"
)

// ShowCmd shows information that is relevant to a LeafBridge deployment.
type ShowCmd struct {
	Config     ShowConfigCmd     `kong:"cmd,help='Shows configuration loaded from a deployment configuration file.'"`
	Apps       ShowAppsCmd       `kong:"cmd,help='Shows the installation status of applications for a deployment.'"`
	Conditions ShowConditionsCmd `kong:"cmd,help='Shows the current conditions for a deployment.'"`
	Resources  ShowResourcesCmd  `kong:"cmd,help='Shows the relevant resources for a deployment.'"`
}

// ShowConfigCmd shows the configuration of a LeafBridge deployment.
type ShowConfigCmd struct {
	ConfigFile string `kong:"required,name='config-file',help='Path to a deployment file describing the deployment.'"`
}

// Run executes the LeafBridge show config command.
func (cmd ShowConfigCmd) Run(ctx context.Context) error {
	// Read the deployment file.
	dep, err := loadDeployment(cmd.ConfigFile)
	if err != nil {
		return err
	}

	// Print the loaded configuration.
	out, err := json.MarshalIndent(dep, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))

	return nil
}

// ShowAppsCmd shows the current status of applications for a LeafBridge
// deployment.
type ShowAppsCmd struct {
	ConfigFile string `kong:"required,name='config-file',help='Path to a deployment file describing the deployment.'"`
	Installed  bool   `kong:"optional,name='installed',help='Show apps that are installed.'"`
	Missing    bool   `kong:"optional,name='missing',help='Show apps that are missing.'"`
}

// Run executes the LeafBridge show apps command.
func (cmd ShowAppsCmd) Run(ctx context.Context) error {
	// Read the deployment file.
	dep, err := loadDeployment(cmd.ConfigFile)
	if err != nil {
		return err
	}

	// Validate the dpeloyment.
	if err := dep.Validate(); err != nil {
		fmt.Printf("The deployment contains invalid configuration: %s\n", err)
		os.Exit(1)
	}

	switch {
	case !cmd.showAll() && cmd.Installed:
		fmt.Printf("---- %s (%s): Installed Applications ----\n", dep.Name, cmd.ConfigFile)
	case !cmd.showAll() && cmd.Missing:
		fmt.Printf("---- %s (%s): Missing Applications ----\n", dep.Name, cmd.ConfigFile)
	default:
		fmt.Printf("---- %s (%s): Applications ----\n", dep.Name, cmd.ConfigFile)
	}

	// Prepare an application engine.
	ae := lbengine.NewAppEngine(dep)

	// Sort the app IDs for a deterministic order.
	ids := slices.Collect(maps.Keys(dep.Apps))
	slices.Sort(ids)

	// Print the status of each condition.
	for _, id := range ids {
		installed, installedErr := ae.IsInstalled(id)
		if installedErr == nil {
			if installed && !cmd.Installed && !cmd.showAll() {
				continue
			}
			if !installed && !cmd.Missing && !cmd.showAll() {
				continue
			}
		}

		app := dep.Apps[id]

		fmt.Printf("    %s\n", id)

		if app.ProductCode != "" {
			fmt.Printf("      Product Code: %s\n", app.ProductCode)
		}

		if app.Name != "" {
			fmt.Printf("      Name:         %s\n", app.Name)
		}

		{
			var info []string
			if scope := string(app.Scope); scope != "" {
				switch scope {
				case "machine":
					scope = "Machine"
				case "user":
					scope = "User"
				}
				info = append(info, scope)
			}
			if app.Architecture != "" {
				info = append(info, string(app.Architecture))
			}
			if version, err := ae.Version(id); err == nil && version != "" {
				info = append(info, fmt.Sprintf("v%s", version.Canonical()))
			} else if installedErr != nil {
				info = append(info, installedErr.Error())
			} else if installed {
				info = append(info, "Installed")
			} else {
				info = append(info, "Missing")
			}
			if len(info) > 0 {
				fmt.Printf("      Info:         %s\n", strings.Join(info, ", "))
			}
		}
	}

	return nil
}

// showAll returns true if all applications should be shown.
func (cmd ShowAppsCmd) showAll() bool {
	if cmd.Installed && cmd.Missing {
		return true
	}
	if !cmd.Installed && !cmd.Missing {
		return true
	}
	return false
}

// ShowConditionsCmd shows the current status of conditions for a
// LeafBridge deployment.
type ShowConditionsCmd struct {
	ConfigFile string `kong:"required,name='config-file',help='Path to a deployment file describing the deployment.'"`
}

// Run executes the LeafBridge show conditions command.
func (cmd ShowConditionsCmd) Run(ctx context.Context) error {
	// Read the deployment file.
	dep, err := loadDeployment(cmd.ConfigFile)
	if err != nil {
		return err
	}

	// Validate the dpeloyment.
	if err := dep.Validate(); err != nil {
		fmt.Printf("The deployment contains invalid configuration: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("---- %s (%s): Conditions ----\n", dep.Name, cmd.ConfigFile)

	// Prepare a condition engine.
	ce := lbengine.NewConditionEngine(dep)

	// Sort the condition IDs for a deterministic order.
	ids := slices.Collect(maps.Keys(dep.Conditions))
	slices.Sort(ids)

	// Print the status of each condition.
	for _, id := range ids {
		result, err := ce.Evaluate(id)
		if err != nil {
			fmt.Printf("    %s: %s\n", id, err)
		} else {
			fmt.Printf("    %s: %t\n", id, result)
		}
	}

	return nil
}

// ShowResourcesCmd shows the current condition of relevant resources for
// a LeafBridge deployment.
type ShowResourcesCmd struct {
	ConfigFile string `kong:"required,name='config-file',help='Path to a deployment file describing the deployment.'"`
}

// Run executes the LeafBridge show resources command.
func (cmd ShowResourcesCmd) Run(ctx context.Context) error {
	// Read the deployment file.
	dep, err := loadDeployment(cmd.ConfigFile)
	if err != nil {
		return err
	}

	// Validate the dpeloyment.
	if err := dep.Validate(); err != nil {
		fmt.Printf("The deployment contains invalid configuration: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("---- %s (%s): Resources ----\n", dep.Name, cmd.ConfigFile)

	// Print process resources.
	if processes := dep.Resources.Processes; len(processes) > 0 {
		// Sort the process resource IDs for a deterministic order.
		ids := slices.Collect(maps.Keys(processes))
		slices.Sort(ids)

		// Print information about each process.
		fmt.Printf("  Processes:\n")
		for _, id := range ids {
			process := processes[id]
			func() {
				// Print the resource ID and description.
				fmt.Printf("    %s:\n", id)
				fmt.Printf("      Description: %s\n", process.Description)

				// Look for running processes that match the criteria.
				total, err := lbengine.NumberOfRunningProcesses(process.Match)
				if err != nil {
					fmt.Printf("      Running:     (%v)\n", process.Description)
					return
				}

				// Print the number of running processes.
				switch total {
				case 0:
					fmt.Printf("      Running:     No\n")
				case 1:
					fmt.Printf("      Running:     Yes (%d process)\n", total)
				default:
					fmt.Printf("      Running:     Yes (%d processes)\n", total)
				}
			}()
		}
	}

	// Print mutex resources.
	if mutexes := dep.Resources.Mutexes; len(mutexes) > 0 {
		// Sort the mutex IDs for a deterministic order.
		ids := slices.Collect(maps.Keys(mutexes))
		slices.Sort(ids)

		// Print information about each mutex.
		fmt.Printf("  Mutexes:\n")
		for _, id := range ids {
			mutex := mutexes[id]
			func() {
				fmt.Printf("    %s:\n", id)

				// Print the object name of the mutex.
				name, err := mutex.ObjectName()
				if err != nil {
					fmt.Printf("      Name:        (%v)\n", err)
					return
				}
				fmt.Printf("      Name:        %s\n", name)

				// Print the status of the mutex.
				exists, err := winmutex.Exists(name)
				if err != nil {
					fmt.Printf("      Status:      (%v)\n", err)
					return
				}

				if exists {
					fmt.Printf("      Status:      Present\n")
				} else {
					fmt.Printf("      Status:      Missing\n")
				}
			}()
		}
	}

	// Print registry key resources.
	if keys := dep.Resources.Registry.Keys; len(keys) > 0 {
		// Sort the registry key IDs for a deterministic order.
		ids := slices.Collect(maps.Keys(keys))
		slices.Sort(ids)

		// Print information about each registry key.
		fmt.Printf("  Registry Keys:\n")
		for _, id := range ids {
			func() {
				fmt.Printf("    %s:\n", id)

				// Resolve the registry key reference.
				ref, err := dep.Resources.Registry.ResolveKey(id)
				if err != nil {
					fmt.Printf("      Path:        (%v)\n", err)
					return
				}

				// Generate a registry key path.
				path, err := ref.Path()
				if err != nil {
					fmt.Printf("      Path:        (%v)\n", err)
					return
				}

				// Open the registry key.
				key, err := localregistry.OpenKey(ref)
				if err != nil {
					fmt.Printf("      Path:        %s\n", path)
					if os.IsNotExist(err) {
						fmt.Printf("      Status:      Missing\n")
					} else {
						fmt.Printf("      Status:      (%v)\n", err)
					}
					return
				}
				defer key.Close()

				// Print the path and status.
				fmt.Printf("      Path:        %s\n", key.Path())
				fmt.Printf("      Status:      Present\n")
			}()
		}
	}

	// Print registry value resources.
	if values := dep.Resources.Registry.Values; len(values) > 0 {
		// Sort the registry value IDs for a deterministic order.
		ids := slices.Collect(maps.Keys(values))
		slices.Sort(ids)

		// Print information about each registry value.
		fmt.Printf("  Registry Values:\n")
		for _, id := range ids {
			func() {
				fmt.Printf("    %s:\n", id)

				// Resolve the registry value reference.
				ref, err := dep.Resources.Registry.ResolveValue(id)
				if err != nil {
					fmt.Printf("      Key:         (%v)\n", err)
					fmt.Printf("      Name:        %s\n", ref.Name)
					return
				}

				// Generate a registry key path.
				path, err := ref.Key().Path()
				if err != nil {
					fmt.Printf("      Key:         (%v)\n", err)
					fmt.Printf("      Name:        %s\n", ref.Name)
					return
				}

				// Attempt to open the parent key.
				key, err := localregistry.OpenKey(ref.Key())
				if err != nil {
					fmt.Printf("      Key:         %s\n", path)
					fmt.Printf("      Name:        %s\n", ref.Name)
					if os.IsNotExist(err) {
						fmt.Printf("      Status:      Missing\n")
					} else {
						fmt.Printf("      Status:      (%v)\n", err)
					}
					return
				}
				defer key.Close()

				// Print the key path and value name
				fmt.Printf("      Key:         %s\n", key.Path())
				fmt.Printf("      Name:        %s\n", ref.Name)

				// Determine whether the registry value exists.
				exists, err := key.HasValue(ref.Name)
				if err != nil {
					fmt.Printf("      Status:      (%v)\n", err)
				}

				if !exists {
					fmt.Printf("      Status:      Missing\n")
					return
				}

				value, err := key.GetValue(ref.Name, ref.Type)
				if err != nil {
					fmt.Printf("      Value:       (%v)\n", err)
					return
				}
				fmt.Printf("      Value:       %s\n", value)

				// TODO: Report statistics.
				//fmt.Printf("      Modified: %s\n", fi.ModTime())
				//fmt.Printf("      Size:     %d bytes(s)\n", fi.Size())
			}()
		}
	}

	{
		// Prepare a local file system resolver.
		resolver := localfs.NewResolver(dep.Resources.FileSystem)

		// Print directory resources.
		if dirs := dep.Resources.FileSystem.Directories; len(dirs) > 0 {
			// Sort the directory IDs for a deterministic order.
			ids := slices.Collect(maps.Keys(dirs))
			slices.Sort(ids)

			// Print information about each file.
			fmt.Printf("  Directories:\n")
			for _, id := range ids {
				func() {
					fmt.Printf("    %s:\n", id)

					// Resolve the directory reference.
					ref, err := resolver.ResolveDirectory(id)
					if err != nil {
						fmt.Printf("      Path:        (%v)\n", err)
						return
					}

					// Generate a file path.
					path, err := ref.Path()
					if err != nil {
						fmt.Printf("      Path:        (%v)\n", err)
						return
					}

					// Open the parent directory.
					dir, err := localfs.OpenDir(ref)
					if err != nil {
						fmt.Printf("      Path:        %s\n", path)
						if os.IsNotExist(err) {
							fmt.Printf("      Status:      Missing\n")
						} else {
							fmt.Printf("      Status:      (%v)\n", err)
						}
						return
					}
					defer dir.Close()

					// Print the path and status.
					fmt.Printf("      Path:        %s\n", dir.Path())
					fmt.Printf("      Status:      Present\n")
				}()
			}
		}

		// Print file resources.
		if files := dep.Resources.FileSystem.Files; len(files) > 0 {
			// Sort the file IDs for a deterministic order.
			ids := slices.Collect(maps.Keys(files))
			slices.Sort(ids)

			// Print information about each file.
			fmt.Printf("  Files:\n")
			for _, id := range ids {
				func() {
					fmt.Printf("    %s:\n", id)

					// Resolve the file reference.
					ref, err := resolver.ResolveFile(id)
					if err != nil {
						fmt.Printf("      Path:        (%v)\n", err)
						return
					}

					// Generate a file path.
					path, err := ref.Path()
					if err != nil {
						fmt.Printf("      Path:        (%v)\n", err)
						return
					}
					fmt.Printf("      Path:        %s\n", path)

					// Attempt to open the parent directory.
					dir, err := localfs.OpenDir(ref.Dir())
					if err != nil {
						if os.IsNotExist(err) {
							fmt.Printf("      Status:      Missing\n")
						} else {
							fmt.Printf("      Status:      (%v)\n", err)
						}
						return
					}
					defer dir.Close()

					// Stat the file path.
					fi, err := dir.System().Stat(ref.FilePath)
					if err != nil {
						if os.IsNotExist(err) {
							fmt.Printf("      Status:      Missing\n")
						} else {
							fmt.Printf("      Status:      (%v)\n", err)
						}
						return
					}

					// Make sure it's a regular file.
					if !fi.Mode().IsRegular() {
						fmt.Printf("      Status:      Not A File\n")
						return
					}

					// Report statistics.
					fmt.Printf("      Status:      Present\n")
					fmt.Printf("      Modified:    %s\n", fi.ModTime())
					fmt.Printf("      Size:        %d bytes(s)\n", fi.Size())
				}()
			}
		}
	}

	return nil
}

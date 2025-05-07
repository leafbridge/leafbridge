package lbdeployevent

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gentlemanautomaton/structformat"
	"github.com/gentlemanautomaton/structformat/fieldformat"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// CommandSkipped is an event that occurs when a command is skipped.
type CommandSkipped struct {
	Deployment  lbdeploy.DeploymentID
	Flow        lbdeploy.FlowID
	ActionIndex int
	ActionType  lbdeploy.ActionType
	Package     lbdeploy.PackageID
	Command     lbdeploy.CommandID
	Apps        lbdeploy.AppEvaluation
}

// Component identifies the component that generated the event.
func (e CommandSkipped) Component() string {
	return "command"
}

// Level returns the level of the event.
func (e CommandSkipped) Level() slog.Level {
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e CommandSkipped) Message() string {
	var builder structformat.Builder

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary(string(e.ActionType))
	if e.Package == "" {
		builder.WritePrimary(string(e.Command))
	} else {
		builder.WritePrimary(fmt.Sprintf("%s.%s", e.Package, e.Command))
	}
	builder.WriteStandard("Skipped command")
	if len(e.Apps.AlreadyInstalled) > 0 {
		builder.WriteNote(fmt.Sprintf("[%s]", e.Apps.AlreadyInstalled), fieldformat.Label("already installed"))
	}
	if len(e.Apps.AlreadyUninstalled) > 0 {
		builder.WriteNote(fmt.Sprintf("[%s]", e.Apps.AlreadyUninstalled), fieldformat.Label("already uninstalled"))
	}

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e CommandSkipped) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e CommandSkipped) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
	}
	if e.Package != "" {
		attrs = append(attrs, slog.String("package", string(e.Package)))
	}
	attrs = append(attrs, slog.Group("command", "id", e.Command))
	if !e.Apps.IsZero() {
		attrs = append(attrs, slog.Group("affected-apps",
			"already-installed", e.Apps.AlreadyInstalled,
			"already-uninstalled", e.Apps.AlreadyUninstalled,
			"to-install", e.Apps.ToInstall,
			"to-uninstall", e.Apps.ToUninstall))
	}
	return attrs
}

// CommandStarted is an event that occurs when a command has started.
type CommandStarted struct {
	Deployment           lbdeploy.DeploymentID
	Flow                 lbdeploy.FlowID
	ActionIndex          int
	ActionType           lbdeploy.ActionType
	Package              lbdeploy.PackageID
	Command              lbdeploy.CommandID
	CommandLine          string
	WorkingDirectory     lbdeploy.DirectoryResourceID
	WorkingDirectoryPath string
	Apps                 lbdeploy.AppEvaluation
}

// Component identifies the component that generated the event.
func (e CommandStarted) Component() string {
	return "command"
}

// Level returns the level of the event.
func (e CommandStarted) Level() slog.Level {
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e CommandStarted) Message() string {
	var builder structformat.Builder

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary(string(e.ActionType))
	if e.Package == "" {
		builder.WritePrimary(string(e.Command))
	} else {
		builder.WritePrimary(fmt.Sprintf("%s.%s", e.Package, e.Command))
	}
	switch installs, uninstalls := len(e.Apps.ToInstall), len(e.Apps.ToUninstall); {
	case installs > 0 && uninstalls > 0:
		builder.WritePrimary(fmt.Sprintf("Starting command to install %s and uninstall %s", e.Apps.ToInstall, e.Apps.ToUninstall))
	case installs > 0 && uninstalls > 0:
		builder.WritePrimary(fmt.Sprintf("Starting command to install %s", e.Apps.ToInstall))
	case uninstalls > 0:
		builder.WritePrimary(fmt.Sprintf("Starting command to uninstall %s", e.Apps.ToUninstall))
	default:
		builder.WritePrimary("Starting command")
	}
	builder.WriteStandard(e.CommandLine)

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e CommandStarted) Details() string {
	switch {
	case e.WorkingDirectoryPath != "":
		return fmt.Sprintf("Working Directory: %s", e.WorkingDirectoryPath)
	case e.WorkingDirectory != "":
		return fmt.Sprintf("Working Directory: %s", e.WorkingDirectory)
	default:
		return ""
	}
}

// Attrs returns a set of structured log attributes for the event.
func (e CommandStarted) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
	}
	if e.Package != "" {
		attrs = append(attrs, slog.String("package", string(e.Package)))
	}
	attrs = append(attrs, slog.Group("command", "id", e.Command, "invocation", e.CommandLine))
	if e.WorkingDirectory != "" || e.WorkingDirectoryPath != "" {
		attrs = append(attrs, slog.Group("working-directory", "id", e.WorkingDirectory, "path", e.WorkingDirectoryPath))
	}
	if !e.Apps.IsZero() {
		attrs = append(attrs, slog.Group("affected-apps",
			"already-installed", e.Apps.AlreadyInstalled,
			"already-uninstalled", e.Apps.AlreadyUninstalled,
			"to-install", e.Apps.ToInstall,
			"to-uninstall", e.Apps.ToUninstall))
	}
	return attrs
}

// CommandStopped is an event that occurs when a command has stopped.
type CommandStopped struct {
	Deployment           lbdeploy.DeploymentID
	Flow                 lbdeploy.FlowID
	ActionIndex          int
	ActionType           lbdeploy.ActionType
	Package              lbdeploy.PackageID
	Command              lbdeploy.CommandID
	CommandLine          string
	Result               lbdeploy.CommandResult
	Output               string
	WorkingDirectory     lbdeploy.DirectoryResourceID
	WorkingDirectoryPath string
	AppsBefore           lbdeploy.AppEvaluation
	AppsAfter            lbdeploy.AppSummary
	Started              time.Time
	Stopped              time.Time
	Err                  error
}

// Component identifies the component that generated the event.
func (e CommandStopped) Component() string {
	return "command"
}

// Level returns the level of the event.
func (e CommandStopped) Level() slog.Level {
	if e.Err != nil || e.AppsAfter.Err() != nil {
		return slog.LevelError
	}
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e CommandStopped) Message() string {
	var builder structformat.Builder

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary(string(e.ActionType))
	if e.Package == "" {
		builder.WritePrimary(string(e.Command))
	} else {
		builder.WritePrimary(fmt.Sprintf("%s.%s", e.Package, e.Command))
	}
	if e.Err != nil {
		builder.WriteStandard(fmt.Sprintf("Stopped command due to an error: %s", e.Err))
	} else if err := e.AppsAfter.Err(); err != nil {
		builder.WriteStandard(fmt.Sprintf("Completed command but %s", err))
	} else {
		builder.WriteStandard(fmt.Sprintf("Completed command"))
	}
	builder.WriteNote(e.Duration().Round(time.Millisecond * 10).String())
	if e.Result.ExitCode != 0 {
		builder.WriteNote(e.Result.String())
	}

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e CommandStopped) Details() string {
	var out strings.Builder

	switch {
	case e.WorkingDirectoryPath != "":
		out.WriteString(fmt.Sprintf("Working Directory: %s", e.WorkingDirectoryPath))
	case e.WorkingDirectory != "":
		out.WriteString(fmt.Sprintf("Working Directory: %s", e.WorkingDirectory))
	default:
	}

	if e.CommandLine != "" {
		if out.Len() > 0 {
			out.WriteString("\n\n")
		}
		out.WriteString(e.CommandLine)
	}

	if e.Output != "" {
		if out.Len() > 0 {
			out.WriteString("\n\n")
		}
		out.WriteString(e.Output)
	}

	return out.String()
}

// Attrs returns a set of structured log attributes for the event.
func (e CommandStopped) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
	}
	if e.Package != "" {
		attrs = append(attrs, slog.String("package", string(e.Package)))
	}
	attrs = append(attrs,
		slog.Group("command", "id", e.Command, "invocation", e.CommandLine),
		slog.Time("started", e.Started),
		slog.Time("stopped", e.Stopped),
	)
	if e.WorkingDirectory != "" || e.WorkingDirectoryPath != "" {
		attrs = append(attrs, slog.Group("working-directory", "id", e.WorkingDirectory, "path", e.WorkingDirectoryPath))
	}
	if !e.AppsBefore.IsZero() {
		attrs = append(attrs, slog.Group("affected-apps-before",
			"already-installed", e.AppsBefore.AlreadyInstalled,
			"already-uninstalled", e.AppsBefore.AlreadyUninstalled,
			"to-install", e.AppsBefore.ToInstall,
			"to-uninstall", e.AppsBefore.ToUninstall))
	}
	if !e.AppsAfter.IsZero() {
		attrs = append(attrs, slog.Group("affected-apps-after",
			"installed", e.AppsAfter.Installed,
			"uninstalled", e.AppsAfter.Uninstalled,
			"still-not-installed", e.AppsAfter.StillNotInstalled,
			"still-not-uninstalled", e.AppsAfter.StillNotUninstalled))
	}
	if e.Output != "" {
		attrs = append(attrs, slog.String("output", e.Output))
	}
	err := e.Err
	if err == nil {
		err = e.AppsAfter.Err()
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	return attrs
}

// Duration returns the duration of the action.
func (e CommandStopped) Duration() time.Duration {
	return e.Stopped.Sub(e.Started)
}

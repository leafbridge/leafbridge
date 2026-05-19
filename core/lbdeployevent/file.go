package lbdeployevent

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gentlemanautomaton/structformat"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbevent"
)

// Deployment file event types.
const (
	FileCopyType   = lbevent.Type("deployment.file:copy")
	FileDeleteType = lbevent.Type("deployment.file:delete")
)

// FileCopy is an event that occurs when a file is copied.
type FileCopy struct {
	Invocation         lbdeploy.InvocationID
	Deployment         lbdeploy.DeploymentID
	Flow               lbdeploy.FlowID
	ActionIndex        int
	ActionType         lbdeploy.ActionType
	SourceID           lbdeploy.FileResourceID
	SourcePath         string
	DestinationID      lbdeploy.FileResourceID
	DestinationPath    string
	DestinationExisted bool
	FileSize           int64
	Started            time.Time
	Stopped            time.Time
	Err                error
}

// Type returns the type of the event.
func (e FileCopy) Type() lbevent.Type {
	return FileCopyType
}

// Level returns the level of the event.
func (e FileCopy) Level() slog.Level {
	if e.Err != nil {
		return slog.LevelError
	}
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e FileCopy) Message() string {
	var builder structformat.Builder

	duration := e.Duration().Round(time.Millisecond * 10)

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary(string(e.ActionType))

	var from, to string
	if e.SourcePath != "" {
		from = fmt.Sprintf("%s (%s)", e.SourceID, e.SourcePath)
	} else {
		from = string(e.SourceID)
	}
	if e.DestinationPath != "" {
		to = fmt.Sprintf("%s (%s)", e.DestinationID, e.DestinationPath)
	} else {
		to = string(e.DestinationID)
	}
	if e.Err != nil {
		builder.WriteStandard(fmt.Sprintf("The file copy from %s to %s failed due to an error: %s.", from, to, e.Err))
	} else if !e.DestinationExisted {
		builder.WriteStandard(fmt.Sprintf("The file copy from %s to %s was completed in %s (%s mbps).", from, to, duration, e.BitrateInMbps()))
	} else {
		builder.WriteStandard(fmt.Sprintf("The file copy from %s to %s was unnecessary as the file already exists in the destination.", from, to))
	}

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e FileCopy) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e FileCopy) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("invocation", string(e.Invocation)),
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
		slog.Group("source", "path", e.SourcePath),
		slog.Group("destination", "path", e.DestinationPath, "existed", e.DestinationExisted),
		slog.Group("file", "size", e.FileSize),
		slog.Time("started", e.Started),
		slog.Time("stopped", e.Stopped),
	}
	if e.Err != nil {
		attrs = append(attrs, slog.String("error", e.Err.Error()))
	}
	return attrs
}

// Duration returns the duration of the file copy process.
func (e FileCopy) Duration() time.Duration {
	return e.Stopped.Sub(e.Started)
}

// BitrateInMbps returns the bitrate of the file copy in mebibits per second.
func (e FileCopy) BitrateInMbps() string {
	return bitrate(e.FileSize, e.Duration())
}

// FileDelete is an event that occurs when a file is deleted.
type FileDelete struct {
	Invocation  lbdeploy.InvocationID
	Deployment  lbdeploy.DeploymentID
	Flow        lbdeploy.FlowID
	ActionIndex int
	ActionType  lbdeploy.ActionType
	FileID      lbdeploy.FileResourceID
	FilePath    string
	FileSize    int64
	FileExisted bool
	Started     time.Time
	Stopped     time.Time
	Err         error
}

// Type returns the type of the event.
func (e FileDelete) Type() lbevent.Type {
	return FileDeleteType
}

// Level returns the level of the event.
func (e FileDelete) Level() slog.Level {
	if e.Err != nil {
		return slog.LevelError
	}
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e FileDelete) Message() string {
	var builder structformat.Builder

	duration := e.Duration().Round(time.Millisecond * 10)

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary(string(e.ActionType))

	var from string
	if e.FilePath != "" {
		from = fmt.Sprintf("%s (%s)", e.FileID, e.FilePath)
	} else {
		from = string(e.FileID)
	}
	if e.Err != nil {
		builder.WriteStandard(fmt.Sprintf("Deletion of %s failed due to an error: %s.", from, e.Err))
	} else if e.FileExisted {
		builder.WriteStandard(fmt.Sprintf("Deletion of %s was completed in %s (%s mbps).", from, duration, e.BitrateInMbps()))
	} else {
		builder.WriteStandard(fmt.Sprintf("Deletion of %s was unnecessary as the file did not exist.", from))
	}

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e FileDelete) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e FileDelete) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("invocation", string(e.Invocation)),
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
		slog.Group("file", "id", e.FileID, "path", e.FilePath, "size", e.FileSize, "existed", e.FileExisted),
		slog.Time("started", e.Started),
		slog.Time("stopped", e.Stopped),
	}
	if e.Err != nil {
		attrs = append(attrs, slog.String("error", e.Err.Error()))
	}
	return attrs
}

// Duration returns the duration of the file deletion process.
func (e FileDelete) Duration() time.Duration {
	return e.Stopped.Sub(e.Started)
}

// BitrateInMbps returns the bitrate of the file deletion in mebibits per second.
func (e FileDelete) BitrateInMbps() string {
	return bitrate(e.FileSize, e.Duration())
}

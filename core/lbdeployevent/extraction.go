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

// Deployment extraction event types.
const (
	ExtractionStartedType = lbevent.Type("deployment.extraction:started")
	ExtractionStoppedType = lbevent.Type("deployment.extraction:stopped")
)

// ExtractionStats holds information about of files that are being extracted.
//
// TODO: Remake this into a generic file set statistics and move it to
// a different package.
type ExtractionStats struct {
	Files       int
	Directories int
	TotalBytes  int64
}

// String returns a string representation of the stats in the form
// "100 files and 1000 directories".
func (stats ExtractionStats) String() string {
	switch {
	case stats.Files > 0 && stats.Directories > 0:
		return fmt.Sprintf("%d %s and %d %s",
			stats.Files,
			plural(stats.Files, "file", "files"),
			stats.Directories,
			plural(stats.Directories, "directory", "directories"))
	case stats.Files > 0:
		return fmt.Sprintf("%d %s",
			stats.Files,
			plural(stats.Files, "file", "files"))
	case stats.Directories > 0:
		return fmt.Sprintf("%d %s",
			stats.Files,
			plural(stats.Files, "file", "files"))
	default:
		return "no files and no directories"
	}
}

// ExtractionStarted is an event that occurs when archive extraction has
// started.
type ExtractionStarted struct {
	Deployment      lbdeploy.DeploymentID
	Flow            lbdeploy.FlowID
	ActionIndex     int
	ActionType      lbdeploy.ActionType
	SourcePath      string
	DestinationPath string
	SourceStats     ExtractionStats
}

// Type returns the type of the event.
func (e ExtractionStarted) Type() lbevent.Type {
	return ExtractionStartedType
}

// Level returns the level of the event.
func (e ExtractionStarted) Level() slog.Level {
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e ExtractionStarted) Message() string {
	var builder structformat.Builder

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary("extract-package")
	builder.WriteStandard(fmt.Sprintf("Starting extraction of %s contained in the \"%s\" archive to \"%s\".", e.SourceStats, e.SourcePath, e.DestinationPath))

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e ExtractionStarted) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e ExtractionStarted) Attrs() []slog.Attr {
	return []slog.Attr{
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
		slog.Group("source", "path", e.SourcePath, slog.Group("stats", "files", e.SourceStats.Files, "directories", e.SourceStats.Directories, "total-bytes", e.SourceStats.TotalBytes)),
		slog.Group("destination", "path", e.DestinationPath),
	}
}

// ExtractionStopped is an event that occurs when archive extraction has
// stopped.
type ExtractionStopped struct {
	Deployment       lbdeploy.DeploymentID
	Flow             lbdeploy.FlowID
	ActionIndex      int
	ActionType       lbdeploy.ActionType
	SourcePath       string
	DestinationPath  string
	SourceStats      ExtractionStats
	DestinationStats ExtractionStats
	Started          time.Time
	Stopped          time.Time
	Err              error
}

// Type returns the type of the event.
func (e ExtractionStopped) Type() lbevent.Type {
	return ExtractionStoppedType
}

// Level returns the level of the event.
func (e ExtractionStopped) Level() slog.Level {
	if e.Err != nil {
		return slog.LevelError
	}
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e ExtractionStopped) Message() string {
	var builder structformat.Builder

	duration := e.Duration().Round(time.Millisecond * 10)

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary("extract-package")
	if e.Err != nil {
		if e.DestinationStats.Files > 0 || e.DestinationStats.Directories > 0 {
			builder.WriteStandard(fmt.Sprintf("The extraction of %s from \"%s\" to \"%s\" failed after %s (%s mbps): %s.", e.SourceStats, e.SourcePath, e.DestinationPath, duration, e.BitrateInMbps(), e.Err))
		} else {
			builder.WriteStandard(fmt.Sprintf("The extraction of %s from \"%s\" to \"%s\" failed due to an error: %s.", e.SourceStats, e.SourcePath, e.DestinationPath, e.Err))
		}
	} else {
		builder.WriteStandard(fmt.Sprintf("The extraction of %s from \"%s\" to \"%s\" was completed in %s (%s mbps).", e.SourceStats, e.SourcePath, e.DestinationPath, duration, e.BitrateInMbps()))
	}

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e ExtractionStopped) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e ExtractionStopped) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
		slog.Group("source", "path", e.SourcePath, slog.Group("stats", "files", e.SourceStats.Files, "directories", e.SourceStats.Directories, "total-bytes", e.SourceStats.TotalBytes)),
		slog.Group("destination", "path", e.DestinationPath, slog.Group("stats", "files", e.SourceStats.Files, "directories", e.SourceStats.Directories, "total-bytes", e.SourceStats.TotalBytes)),
		slog.Time("started", e.Started),
		slog.Time("stopped", e.Stopped),
	}
	if e.Err != nil {
		attrs = append(attrs, slog.String("error", e.Err.Error()))
	}
	return attrs
}

// Duration returns the duration of the extraction process.
func (e ExtractionStopped) Duration() time.Duration {
	return e.Stopped.Sub(e.Started)
}

// BitrateInMbps returns the bitrate of the extraction in mebibits per second.
func (e ExtractionStopped) BitrateInMbps() string {
	return bitrate(e.DestinationStats.TotalBytes, e.Duration())
}

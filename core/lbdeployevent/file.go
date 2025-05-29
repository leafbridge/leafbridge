package lbdeployevent

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gentlemanautomaton/structformat"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbevent"
)

// Deployment file event types.
const (
	FileExtractionType   = lbevent.Type("deployment.file:extraction")
	FileVerificationType = lbevent.Type("deployment.file:verification")
	FileCopyType         = lbevent.Type("deployment.file:copy")
	FileDeleteType       = lbevent.Type("deployment.file:delete")
)

// FileExtraction is an event that occurs when an archived file has been
// extracted.
type FileExtraction struct {
	Deployment lbdeploy.DeploymentID
	Flow       lbdeploy.FlowID
	Action     lbdeploy.ActionType
	FileNumber int
	Path       string
	FileSize   int64
	Started    time.Time
	Stopped    time.Time
	Err        error
}

// Type returns the type of the event.
func (e FileExtraction) Type() lbevent.Type {
	return FileExtractionType
}

// Level returns the level of the event.
func (e FileExtraction) Level() slog.Level {
	if e.Err != nil {
		return slog.LevelError
	}
	return slog.LevelDebug
}

// Message returns a description of the event.
func (e FileExtraction) Message() string {
	duration := e.Duration().Round(time.Millisecond * 10)
	if e.Err != nil {
		return fmt.Sprintf("Extract: File %d: %s: Failed: %s. (%d %s, %s, %s mbps)", e.FileNumber, e.Path, e.Err, e.FileSize, plural(e.FileSize, "byte", "bytes"), duration, e.BitrateInMbps())
	}
	return fmt.Sprintf("Extract: File %d: %s: Completed. (%d %s, %s, %s mbps)", e.FileNumber, e.Path, e.FileSize, plural(e.FileSize, "byte", "bytes"), duration, e.BitrateInMbps())
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e FileExtraction) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e FileExtraction) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.String("action", string(e.Action)),
		slog.Int("file-number", e.FileNumber),
		slog.String("path", e.Path),
		slog.Int64("file-size", e.FileSize),
		slog.Time("started", e.Started),
		slog.Time("stopped", e.Stopped),
	}
	if e.Err != nil {
		attrs = append(attrs, slog.String("error", e.Err.Error()))
	}
	return attrs
}

// Duration returns the duration of the extraction process.
func (e FileExtraction) Duration() time.Duration {
	return e.Stopped.Sub(e.Started)
}

// BitrateInMbps returns the bitrate of the extraction in mebibits per second.
func (e FileExtraction) BitrateInMbps() string {
	return bitrate(e.FileSize, e.Duration())
}

// FileVerification is an event that records the result of verifying
// a downloaded file.
type FileVerification struct {
	Deployment  lbdeploy.DeploymentID
	Flow        lbdeploy.FlowID
	ActionIndex int
	ActionType  lbdeploy.ActionType
	Source      lbdeploy.PackageSource
	FileName    string
	Path        string
	Expected    lbdeploy.FileAttributes
	Actual      lbdeploy.FileAttributes
}

// Type returns the type of the event.
func (e FileVerification) Type() lbevent.Type {
	return FileVerificationType
}

// Level returns the level of the event.
func (e FileVerification) Level() slog.Level {
	if len(e.Expected.Features()) == 0 {
		return slog.LevelWarn
	}
	if !lbdeploy.EqualFileAttributes(e.Expected, e.Actual) {
		return slog.LevelError
	}
	if len(e.Expected.Hashes) == 0 {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e FileVerification) Message() string {
	var builder structformat.Builder

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary("verify-file")

	if len(e.Expected.Features()) == 0 {
		builder.WriteStandard(fmt.Sprintf("The \"%s\" file could not be verified because file verification data was not provided.", e.FileName))
	} else if !lbdeploy.EqualFileAttributes(e.Expected, e.Actual) {
		builder.WriteStandard(fmt.Sprintf("The \"%s\" file does not have the expected file attributes and has failed verification.", e.FileName))
	} else if len(e.Expected.Hashes) == 0 {
		builder.WriteStandard(fmt.Sprintf("The \"%s\" file has the expected file size, but no file hashes were provided for verification.", e.FileName))
	} else {
		builder.WriteStandard(fmt.Sprintf("The \"%s\" file was verified with the following features: %s.", e.FileName, strings.Join(e.Actual.Features(), ", ")))
	}

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e FileVerification) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e FileVerification) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
	}
	if e.Source.URL != "" {
		attrs = append(attrs, slog.Group("source", "type", string(e.Source.Type), "url", e.Source.URL))
	}
	if e.Path != "" {
		attrs = append(attrs, slog.String("path", string(e.Path)))
	}
	attrs = append(attrs, slog.Group("expected", "size", e.Expected.Size, "hashes", e.Expected.Hashes))
	attrs = append(attrs, slog.Group("actual", "size", e.Actual.Size, "hashes", e.Actual.Hashes))
	return attrs
}

// FileCopy is an event that occurs when a file is copied.
type FileCopy struct {
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

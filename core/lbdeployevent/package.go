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

// Deployment package event types.
const (
	PackageFileVerificationType = lbevent.Type("deployment.package.file:verification")
	PackageFileExtractionType   = lbevent.Type("deployment.package.file:extraction")
	PackageFileCopyType         = lbevent.Type("deployment.package.file:copy")
)

// PackageFileVerification is an event that records the result of verifying
// a downloaded package file.
type PackageFileVerification struct {
	Invocation  lbdeploy.InvocationID
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
func (e PackageFileVerification) Type() lbevent.Type {
	return PackageFileVerificationType
}

// Level returns the level of the event.
func (e PackageFileVerification) Level() slog.Level {
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
func (e PackageFileVerification) Message() string {
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
func (e PackageFileVerification) Details() string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "Expected Size: %d\nActual Size: %d", e.Expected.Size, e.Actual.Size)

	for _, hash := range e.Expected.Hashes.ToList() {
		expected := hash.Value
		actual, ok := e.Actual.Hashes[hash.Type]
		if ok {
			fmt.Fprintf(&builder, "\n\nExpected %s Hash: %s\nActual %s Hash: %s", hash.Type, expected, hash.Type, actual)
		} else {
			fmt.Fprintf(&builder, "\n\nExpected %s Hash: %s\nActual %s Hash: Missing", hash.Type, expected, hash.Type)
		}
	}

	return builder.String()
}

// Attrs returns a set of structured log attributes for the event.
func (e PackageFileVerification) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("invocation", string(e.Invocation)),
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

// PackageFileExtraction is an event that occurs when a file has been
// extracted from an archive package.
type PackageFileExtraction struct {
	Invocation lbdeploy.InvocationID
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
func (e PackageFileExtraction) Type() lbevent.Type {
	return PackageFileExtractionType
}

// Level returns the level of the event.
func (e PackageFileExtraction) Level() slog.Level {
	if e.Err != nil {
		return slog.LevelError
	}
	return slog.LevelDebug
}

// Message returns a description of the event.
func (e PackageFileExtraction) Message() string {
	duration := e.Duration().Round(time.Millisecond * 10)
	if e.Err != nil {
		return fmt.Sprintf("Extract: File %d: %s: Failed: %s. (%d %s, %s, %s mbps)", e.FileNumber, e.Path, e.Err, e.FileSize, plural(e.FileSize, "byte", "bytes"), duration, e.BitrateInMbps())
	}
	return fmt.Sprintf("Extract: File %d: %s: Completed. (%d %s, %s, %s mbps)", e.FileNumber, e.Path, e.FileSize, plural(e.FileSize, "byte", "bytes"), duration, e.BitrateInMbps())
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e PackageFileExtraction) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e PackageFileExtraction) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("invocation", string(e.Invocation)),
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
func (e PackageFileExtraction) Duration() time.Duration {
	return e.Stopped.Sub(e.Started)
}

// BitrateInMbps returns the bitrate of the extraction in mebibits per second.
func (e PackageFileExtraction) BitrateInMbps() string {
	return bitrate(e.FileSize, e.Duration())
}

// PackageFileCopy is an event that occurs when a package file is copied.
type PackageFileCopy struct {
	Invocation         lbdeploy.InvocationID
	Deployment         lbdeploy.DeploymentID
	Flow               lbdeploy.FlowID
	ActionIndex        int
	ActionType         lbdeploy.ActionType
	SourcePackage      lbdeploy.PackageID
	SourcePackageFile  lbdeploy.PackageFileID
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
func (e PackageFileCopy) Type() lbevent.Type {
	return FileCopyType
}

// Level returns the level of the event.
func (e PackageFileCopy) Level() slog.Level {
	if e.Err != nil {
		return slog.LevelError
	}
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e PackageFileCopy) Message() string {
	var builder structformat.Builder

	duration := e.Duration().Round(time.Millisecond * 10)

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary(string(e.ActionType))

	var from, to string
	if e.SourcePackageFile != "" {
		from = string(e.SourcePackage) + "." + string(e.SourcePackageFile)
	} else {
		from = string(e.SourcePackage)
	}
	if e.SourcePath != "" {
		from = fmt.Sprintf("%s (%s)", from, e.SourcePath)
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
func (e PackageFileCopy) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e PackageFileCopy) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("invocation", string(e.Invocation)),
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
	}
	if e.SourcePackageFile != "" {
		attrs = append(attrs, slog.Group("source", "package", e.SourcePackage, "file", e.SourcePackageFile, "path", e.SourcePath))
	} else {
		attrs = append(attrs, slog.Group("source", "package", e.SourcePackage, "path", e.SourcePath))
	}
	attrs = append(attrs,
		slog.Group("destination", "path", e.DestinationPath, "existed", e.DestinationExisted),
		slog.Group("file", "size", e.FileSize),
		slog.Time("started", e.Started),
		slog.Time("stopped", e.Stopped),
	)
	if e.Err != nil {
		attrs = append(attrs, slog.String("error", e.Err.Error()))
	}
	return attrs
}

// Duration returns the duration of the file copy process.
func (e PackageFileCopy) Duration() time.Duration {
	return e.Stopped.Sub(e.Started)
}

// BitrateInMbps returns the bitrate of the file copy in mebibits per second.
func (e PackageFileCopy) BitrateInMbps() string {
	return bitrate(e.FileSize, e.Duration())
}

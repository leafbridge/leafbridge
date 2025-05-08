package lbengine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbdeployevent"
	"github.com/leafbridge/leafbridge/core/lbevent"
	"github.com/leafbridge/leafbridge/platform/windows/filetime"
	"github.com/leafbridge/leafbridge/platform/windows/localfs"
)

// fileEngine handles file system operations within a deployment.
type fileEngine struct {
	deployment lbdeploy.Deployment
	flow       flowData
	action     actionData
	events     lbevent.Recorder
	state      *engineState
}

// CopyFile performs a file copy operation.
func (engine *fileEngine) CopyFile(ctx context.Context) error {
	// Prepare a local file system resolver.
	resolver := localfs.NewResolver(engine.deployment.Resources.FileSystem)

	// Find the relevant source file within the deployment.
	sourceFileID := engine.action.Definition.SourceFile
	sourceFileRef, err := resolver.ResolveFile(sourceFileID)
	if err != nil {
		return fmt.Errorf("source file: %w", err)
	}

	// Find the relevant destination file within the deployment.
	destFileID := engine.action.Definition.DestinationFile
	destFileRef, err := resolver.ResolveFile(destFileID)
	if err != nil {
		return fmt.Errorf("destination file: %w", err)
	}

	// Make sure that the destination file is not in a protected location.
	if destFileRef.Root.Protected {
		return fmt.Errorf("the destination file is located in the \"%s\" root, which is protected", destFileRef.Root.ID)
	}

	// Record the time that the file copy started.
	started := time.Now()

	var (
		sourceFilePath  string
		destFilePath    string
		destFileExisted bool
		fileSize        int64
	)
	err = func() error {
		// Open the root above the destination file.
		destDir, err := localfs.OpenDir(destFileRef.Dir())
		if err != nil {
			return fmt.Errorf("unable to open the destination directory: %w", err)
		}
		defer destDir.Close()

		// Record the destination path for event logging.
		{
			localized, err := filepath.Localize(destFileRef.FilePath)
			if err == nil {
				destFilePath = filepath.Join(destDir.Path(), localized)
			}
		}

		// If there is an existing file, stop.
		fi, err := destDir.System().Stat(destFileRef.FilePath)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("unable to evaluate the destination file: %w", err)
			}
		} else if fi.Mode().IsRegular() {
			// The file already exists.
			//
			// TODO: Support replacing existing files, optionally via
			// configuration.
			destFileExisted = true
			return nil
		} else {
			return errors.New("the destination file path already exists but is not a regular file")
		}

		// Open the source file.
		sourceFile, err := localfs.OpenFile(sourceFileRef)
		if err != nil {
			return fmt.Errorf("unable to open the source file: %w", err)
		}
		defer sourceFile.Close()

		// Record the source path and file size for event logging.
		sourceFilePath = sourceFile.Path()
		if fi, err := sourceFile.System().Stat(); err == nil {
			fileSize = fi.Size()
		}

		// Open the destination file.
		destFile, err := destDir.System().Create(destFileRef.FilePath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		// Record the destination path for event logging.
		sourceFilePath = sourceFile.Path()

		// Copy file data.
		if _, err := io.Copy(destFile, sourceFile.System()); err != nil {
			return err
		}

		// Copy the file modification date.
		sourceFileInfo, err := sourceFile.System().Stat()
		if err != nil {
			return err
		}
		if modTime := sourceFileInfo.ModTime(); !modTime.IsZero() {
			if err := filetime.SetFileModificationTime(destFile, modTime); err != nil {
				return fmt.Errorf("failed to set file modification time: %w", err)
			}
		}
		return nil
	}()

	// Record the time that the file copy stopped.
	stopped := time.Now()

	// Record the file copy.
	engine.events.Record(lbdeployevent.FileCopy{
		Deployment:         engine.deployment.ID,
		Flow:               engine.flow.ID,
		ActionIndex:        engine.action.Index,
		ActionType:         engine.action.Definition.Type,
		SourceID:           sourceFileID,
		SourcePath:         sourceFilePath,
		DestinationID:      destFileID,
		DestinationPath:    destFilePath,
		DestinationExisted: destFileExisted,
		FileSize:           fileSize,
		Started:            started,
		Stopped:            stopped,
		Err:                err,
	})

	return nil
}

// DeleteFile performs a file delete operation.
func (engine *fileEngine) DeleteFile(ctx context.Context) error {
	// Prepare a local file system resolver.
	resolver := localfs.NewResolver(engine.deployment.Resources.FileSystem)

	// Find the relevant file within the deployment.
	fileID := engine.action.Definition.DestinationFile
	fileRef, err := resolver.ResolveFile(fileID)
	if err != nil {
		return fmt.Errorf("file: %w", err)
	}

	// Make sure that the file is not in a protected location.
	if fileRef.Root.Protected {
		return fmt.Errorf("the file is located in the \"%s\" root, which is protected", fileRef.Root.ID)
	}

	// Record the time that the file deletion started.
	started := time.Now()

	var (
		filePath    string
		fileSize    int64
		fileExisted bool
	)
	err = func() error {
		// Open the root above the destination file.
		fileDir, err := localfs.OpenDir(fileRef.Dir())
		if err != nil {
			if os.IsNotExist(err) {
				return nil // The parent directory does not exist.
			}
			return fmt.Errorf("unable to open the file's directory: %w", err)
		}
		defer fileDir.Close()

		// Record the file path for event logging.
		{
			localized, err := filepath.Localize(fileRef.FilePath)
			if err == nil {
				filePath = filepath.Join(fileDir.Path(), localized)
			}
		}

		// If there is not an existing file, or if the path points to
		// something other than a regular file, stop.
		fi, err := fileDir.System().Stat(fileRef.FilePath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // The file does not exist.
			}
			return fmt.Errorf("unable to evaluate the file to be deleted: %w", err)
		} else if !fi.Mode().IsRegular() {
			return errors.New("the file path exists but is not a regular file")
		}

		// Record that the file exixted.
		fileExisted = true

		// Delete the file.
		return fileDir.System().Remove(fileRef.FilePath)
	}()

	// Record the time that the file deletion stopped.
	stopped := time.Now()

	// Record the file deletion.
	engine.events.Record(lbdeployevent.FileDelete{
		Deployment:  engine.deployment.ID,
		Flow:        engine.flow.ID,
		ActionIndex: engine.action.Index,
		ActionType:  engine.action.Definition.Type,
		FileID:      fileID,
		FilePath:    filePath,
		FileSize:    fileSize,
		FileExisted: fileExisted,
		Started:     started,
		Stopped:     stopped,
		Err:         err,
	})

	return nil
}

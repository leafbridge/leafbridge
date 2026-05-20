package lbengine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbdeployevent"
	"github.com/leafbridge/leafbridge/core/lbevent"
	"github.com/leafbridge/leafbridge/platform/windows/filetime"
	"github.com/leafbridge/leafbridge/platform/windows/localfs"
)

// packageSourceFile is an interface implemented by package source files,
// such as [stagingfs.PackgeFile] and [tempfs.ExtractedFile].
type packageSourceFile interface {
	LocalizedPath() string
	System() *os.File
	Close() error
}

// fileEngine handles file system operations within a deployment.
type fileEngine struct {
	invocation lbdeploy.Invocation
	deployment lbdeploy.Deployment
	flow       flowData
	action     actionData
	events     lbevent.Recorder
	state      *engineState
}

// CopyFile performs a file copy operation.
func (engine *fileEngine) CopyFile(ctx context.Context) error {
	// Interpret action data.
	action, ok := engine.action.Definition.(lbdeploy.CopyFileAction)
	if !ok {
		return fmt.Errorf("unable to copy file: the action is of type \"%s\"", engine.action.Definition.Type())
	}

	// Prepare a local file system resolver.
	resolver := localfs.NewResolver(engine.deployment.Resources.FileSystem)

	// Find the relevant source file within the deployment.
	sourceFileID := action.SourceFile
	sourceFileRef, err := resolver.ResolveFile(sourceFileID)
	if err != nil {
		return fmt.Errorf("unable to copy file from \"%s\" to \"%s\": %w", action.SourceFile, action.DestinationFile, err)
	}

	// Find the relevant destination file within the deployment.
	destFileID := action.DestinationFile
	destFileRef, err := resolver.ResolveFile(destFileID)
	if err != nil {
		return fmt.Errorf("unable to copy file from \"%s\" to \"%s\": %w", action.SourceFile, action.DestinationFile, err)
	}

	// Record the expected source and destination paths for event logging.
	sourceFilePath := sourceFileRef.LocalizedPath()
	destFilePath := destFileRef.LocalizedPath()

	// Record the time that the file copy started.
	started := time.Now()

	var (
		destFileExisted bool
		fileSize        int64
	)
	err = func() error {
		// Make sure that the destination file is not in a protected location.
		if destFileRef.Root.Protected {
			return fmt.Errorf("the \"%s\" destination file is located in the \"%s\" root, which is protected", destFileID, destFileRef.Root.ID)
		}

		// Open the root above the destination file.
		destDir, err := localfs.OpenDir(destFileRef.Dir())
		if err != nil {
			return fmt.Errorf("unable to open the destination directory: %w", err)
		}
		defer destDir.Close()

		// Record the actual destination path for event logging.
		destFilePath = filepath.Join(destDir.LocalizedPath(), destFileRef.Resource.LocalizedPath)

		// If there is an existing file, stop.
		fi, err := destDir.Stat(destFileRef.Resource.LocalizedPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
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

		// Retrieve information about the source file.
		sourceFileInfo, err := sourceFile.Stat()
		if err != nil {
			return err
		}

		// Record the actual source path and file size for event logging.
		sourceFilePath = sourceFile.LocalizedPath()
		fileSize = sourceFileInfo.Size()

		// Open the destination file.
		destFile, err := destDir.Create(destFileRef.Resource.LocalizedPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		// Copy file data.
		if _, err := io.Copy(destFile, sourceFile.System()); err != nil {
			return err
		}

		// Copy the file modification date.
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
		Invocation:         engine.invocation.ID,
		Deployment:         engine.deployment.ID,
		Flow:               engine.flow.ID,
		ActionIndex:        engine.action.Index,
		ActionType:         engine.action.Definition.Type(),
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

	return err
}

// CopyPackageFile performs a file copy operation.
func (engine *fileEngine) CopyPackageFile(ctx context.Context, source packageSourceFile) error {
	// Interpret action data.
	action, ok := engine.action.Definition.(lbdeploy.CopyPackageFileAction)
	if !ok {
		return fmt.Errorf("unable to copy package file: the action is of type \"%s\"", engine.action.Definition.Type())
	}

	// Prepare a local file system resolver.
	resolver := localfs.NewResolver(engine.deployment.Resources.FileSystem)

	// Find the relevant destination file within the deployment.
	destFileID := action.DestinationFile
	destFileRef, err := resolver.ResolveFile(destFileID)
	if err != nil {
		return fmt.Errorf("unable to copy package file from \"%s\" to \"%s\": %w", action.SourceName(), action.DestinationFile, err)
	}

	// Record the expected source and destination paths for event logging.
	sourceFilePath := source.LocalizedPath()
	destFilePath := destFileRef.LocalizedPath()

	// Record the time that the file copy started.
	started := time.Now()

	var (
		destFileExisted bool
		fileSize        int64
	)
	err = func() error {
		// Make sure that the destination file is not in a protected location.
		if destFileRef.Root.Protected {
			return fmt.Errorf("the \"%s\" destination file is located in the \"%s\" root, which is protected", destFileID, destFileRef.Root.ID)
		}

		// Open the root above the destination file.
		destDir, err := localfs.OpenDir(destFileRef.Dir())
		if err != nil {
			return fmt.Errorf("unable to open the destination directory: %w", err)
		}
		defer destDir.Close()

		// Record the actual destination path for event logging.
		destFilePath = filepath.Join(destDir.LocalizedPath(), destFileRef.Resource.LocalizedPath)

		// If there is an existing file, stop.
		fi, err := destDir.Stat(destFileRef.Resource.LocalizedPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
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

		// Get a reference to the source file.
		sourceFile := source.System()

		// Retrieve information about the source file.
		sourceFileInfo, err := sourceFile.Stat()
		if err != nil {
			return err
		}

		// Record the actual source path and file size for event logging.
		sourceFilePath = source.LocalizedPath()
		fileSize = sourceFileInfo.Size()

		// Open the destination file.
		destFile, err := destDir.Create(destFileRef.Resource.LocalizedPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		// Seek to the start of the source file.
		if _, err := sourceFile.Seek(0, io.SeekStart); err != nil {
			return err
		}

		// Copy file data.
		if _, err := io.Copy(destFile, sourceFile); err != nil {
			return err
		}

		// Copy the file modification date.
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
	engine.events.Record(lbdeployevent.PackageFileCopy{
		Invocation:         engine.invocation.ID,
		Deployment:         engine.deployment.ID,
		Flow:               engine.flow.ID,
		ActionIndex:        engine.action.Index,
		ActionType:         engine.action.Definition.Type(),
		SourcePackage:      action.Package,
		SourcePackageFile:  action.SourceFile,
		SourcePath:         sourceFilePath,
		DestinationID:      destFileID,
		DestinationPath:    destFilePath,
		DestinationExisted: destFileExisted,
		FileSize:           fileSize,
		Started:            started,
		Stopped:            stopped,
		Err:                err,
	})

	return err
}

// DeleteFile performs a file deletion operation.
func (engine *fileEngine) DeleteFile(ctx context.Context) error {
	// Interpret action data.
	action, ok := engine.action.Definition.(lbdeploy.DeleteFileAction)
	if !ok {
		return fmt.Errorf("unable to delete file: the action is of type \"%s\"", engine.action.Definition.Type())
	}

	// Prepare a local file system resolver.
	resolver := localfs.NewResolver(engine.deployment.Resources.FileSystem)

	// Find the relevant file within the deployment.
	fileID := action.File
	fileRef, err := resolver.ResolveFile(fileID)
	if err != nil {
		return fmt.Errorf("unable to delete \"%s\" file: %w", fileID, err)
	}

	// Record the expected file path for event logging.
	filePath := fileRef.LocalizedPath()

	// Record the time that the file deletion started.
	started := time.Now()

	var (
		fileSize    int64
		fileExisted bool
	)
	err = func() error {
		// Make sure that the file is not in a protected location.
		if fileRef.Root.Protected {
			return fmt.Errorf("the \"%s\" file is located in the \"%s\" root, which is protected", fileID, fileRef.Root.ID)
		}

		// Open the root above the destination file.
		fileDir, err := localfs.OpenDir(fileRef.Dir())
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil // The parent directory does not exist.
			}
			return fmt.Errorf("unable to open the file's directory: %w", err)
		}
		defer fileDir.Close()

		// Record the actual file path for event logging.
		filePath = filepath.Join(fileDir.LocalizedPath(), fileRef.Resource.LocalizedPath)

		// If there isn't an existing file, or if the path points to
		// something other than a regular file, stop.
		fi, err := fileDir.Stat(fileRef.Resource.LocalizedPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil // The file does not exist.
			}
			return fmt.Errorf("unable to evaluate the file to be deleted: %w", err)
		} else if !fi.Mode().IsRegular() {
			return errors.New("the file path exists but is not a regular file")
		}

		// Record that the file existed.
		fileExisted = true

		// Delete the file.
		return fileDir.Remove(fileRef.Resource.LocalizedPath)
	}()

	// Record the time that the file deletion stopped.
	stopped := time.Now()

	// Record the file deletion.
	engine.events.Record(lbdeployevent.FileDelete{
		Invocation:  engine.invocation.ID,
		Deployment:  engine.deployment.ID,
		Flow:        engine.flow.ID,
		ActionIndex: engine.action.Index,
		ActionType:  engine.action.Definition.Type(),
		FileID:      fileID,
		FilePath:    filePath,
		FileSize:    fileSize,
		FileExisted: fileExisted,
		Started:     started,
		Stopped:     stopped,
		Err:         err,
	})

	return err
}

// DeleteDirectory performs a directory deletion operation.
func (engine *fileEngine) DeleteDirectory(ctx context.Context) error {
	// Interpret action data.
	action, ok := engine.action.Definition.(lbdeploy.DeleteDirectoryAction)
	if !ok {
		return fmt.Errorf("unable to delete directory: the action is of type \"%s\"", engine.action.Definition.Type())
	}

	// Prepare a local file system resolver.
	resolver := localfs.NewResolver(engine.deployment.Resources.FileSystem)

	// Find the relevant directory within the deployment.
	dirID := action.Dir
	dirRef, err := resolver.ResolveDirectory(dirID)
	if err != nil {
		return fmt.Errorf("unable to delete \"%s\" directory: %w", dirID, err)
	}

	// Record the expected directory path for event logging.
	dirPath := dirRef.LocalizedPath()

	// Record the time that the directory deletion started.
	started := time.Now()

	var (
		dirExisted bool
	)
	err = func() error {
		// Make sure that the directory is not in a protected location.
		if dirRef.Root.Protected {
			return fmt.Errorf("the \"%s\" directory is located in the \"%s\" root, which is protected", dirID, dirRef.Root.ID)
		}

		// Get a reference to the parent above the destination directory.
		parentRef, err := dirRef.Parent()
		if err != nil {
			return err
		}

		// Get the resource for the destination directory.
		dirResource, err := dirRef.Resource()
		if err != nil {
			return err
		}

		// Open the parent above the destination directory.
		parentDir, err := localfs.OpenDir(parentRef)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil // The parent directory does not exist.
			}
			return fmt.Errorf("unable to open the directory's parent: %w", err)
		}
		defer parentDir.Close()

		// Record the actual directory path for event logging.
		dirPath = filepath.Join(parentDir.LocalizedPath(), dirResource.LocalizedPath)

		// If there isn't an existing directory, or if the path points to
		// something other than a directory, stop.
		fi, err := parentDir.Stat(dirResource.LocalizedPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil // The directory does not exist.
			}
			return fmt.Errorf("unable to evaluate the directory to be deleted: %w", err)
		} else if !fi.Mode().IsDir() {
			return errors.New("the directory path exists but is not a directory")
		}

		// Record that the directory existed.
		dirExisted = true

		// TODO: Enumerate the directory and all of its descendents, then
		// record the total number of descendent folders and files in the
		// event.

		// Delete the directory.
		if action.DeleteNonEmpty {
			return parentDir.RemoveAll(dirResource.LocalizedPath)
		}
		return parentDir.Remove(dirResource.LocalizedPath)
	}()

	// Record the time that the file deletion stopped.
	stopped := time.Now()

	// Record the file deletion.
	engine.events.Record(lbdeployevent.DirectoryDelete{
		Invocation:       engine.invocation.ID,
		Deployment:       engine.deployment.ID,
		Flow:             engine.flow.ID,
		ActionIndex:      engine.action.Index,
		ActionType:       engine.action.Definition.Type(),
		DirectoryID:      dirID,
		DirectoryPath:    dirPath,
		DirectoryExisted: dirExisted,
		Started:          started,
		Stopped:          stopped,
		Err:              err,
	})

	return err
}

package lbengine

import (
	"archive/zip"
	"context"
	"fmt"
	"path"
	"time"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbdeployevent"
	"github.com/leafbridge/leafbridge/core/lbevent"
	"github.com/leafbridge/leafbridge/platform/windows/stagingfs"
	"github.com/leafbridge/leafbridge/platform/windows/tempfs"
)

// extractionEngine manages the extraction of files and directories from
// archives.
type extractionEngine struct {
	deployment lbdeploy.Deployment
	flow       flowData
	action     actionData
	events     lbevent.Recorder
	state      *engineState
}

func (engine *extractionEngine) ExtractPackage(ctx context.Context, source stagingfs.PackageFile, destination tempfs.ExtractionDir) error {
	// Record the time that the extraction started.
	started := time.Now()

	// Get the current size of the file.
	fi, err := source.Stat()
	if err != nil {
		return err
	}

	// Prepare a ZIP file reader.
	reader, err := zip.NewReader(source, fi.Size())
	if err != nil {
		return err
	}

	// Collect statistics for the archive.
	var sourceStats lbdeployevent.ExtractionStats
	for _, zipFile := range reader.File {
		fi := zipFile.FileInfo()
		if fi.IsDir() {
			sourceStats.Directories++
		} else {
			sourceStats.Files++
			sourceStats.TotalBytes += fi.Size()
		}
		// FIXME: Include parent directories in file paths, which
		// propbably requires building a map of all directories
		// encountered.
	}

	// Record the start of the extraction.
	engine.events.Record(lbdeployevent.ExtractionStarted{
		Deployment:      engine.deployment.ID,
		Flow:            engine.flow.ID,
		ActionIndex:     engine.action.Index,
		ActionType:      engine.action.Definition.Type,
		SourcePath:      source.Path,
		DestinationPath: destination.Path(),
		SourceStats:     sourceStats,
	})

	// Process each file and directory in the archive.
	var destinationStats lbdeployevent.ExtractionStats
	err = func() error {
		for i, zipFile := range reader.File {
			if err := ctx.Err(); err != nil {
				return err
			}

			// Record the start of the extraction of this file.
			fileStarted := time.Now()

			// Collect information from the zip file.
			fileInfo := zipFile.FileInfo()

			// Attempt to extract the file.
			err := func() error {
				// If this is a directory, make sure it exists.
				if fileInfo.IsDir() {
					if err := destination.MkdirAll(zipFile.Name); err != nil {
						return fmt.Errorf("failed to create parent directory: %w", err)
					}
					destinationStats.Directories++
					return nil
				}

				// FIXME: Include parent directories in file paths, which
				// propbably requires building a map of all directories
				// encountered.

				// If this is a file, make sure the directory it goes in exists.
				if zipDir := path.Dir(zipFile.Name); zipDir != "" && zipDir != "." {
					if err := destination.MkdirAll(zipDir); err != nil {
						return fmt.Errorf("failed to create parent directory: %w", err)
					}
				}

				// Open the file.
				fileReader, err := zipFile.Open()
				if err != nil {
					return fmt.Errorf("failed to open file within the zip archive: %w", err)
				}
				defer fileReader.Close()

				// Write the file to the directory, preserving its
				// modification time.
				written, err := destination.WriteFile(zipFile.Name, newReaderWithContext(ctx, fileReader), zipFile.Modified)
				if err != nil {
					return fmt.Errorf("failed to write file to its destination: %w", err)
				}

				// Update statistics.
				destinationStats.Files++
				destinationStats.TotalBytes += written

				return nil
			}()

			// Record the time that the extraction of this file stopped.
			fileStopped := time.Now()

			// Record the extraction of the file.
			engine.events.Record(lbdeployevent.ExtractedFile{
				Deployment: engine.deployment.ID,
				Flow:       engine.flow.ID,
				Action:     engine.action.Definition.Type,
				FileNumber: i,
				Path:       zipFile.Name,
				FileSize:   fileInfo.Size(),
				Started:    fileStarted,
				Stopped:    fileStopped,
				Err:        err,
			})

			// If the extraction of this file failed, stop the extraction
			// process.
			if err != nil {
				return err
			}
		}
		return nil
	}()

	// Record the time that the extraction stopped.
	stopped := time.Now()

	// Record the end of the extraction.
	engine.events.Record(lbdeployevent.ExtractionStopped{
		Deployment:       engine.deployment.ID,
		Flow:             engine.flow.ID,
		ActionIndex:      engine.action.Index,
		ActionType:       engine.action.Definition.Type,
		SourcePath:       source.Path,
		DestinationPath:  destination.Path(),
		SourceStats:      sourceStats,
		DestinationStats: destinationStats,
		Started:          started,
		Stopped:          stopped,
		Err:              err,
	})

	return err
}

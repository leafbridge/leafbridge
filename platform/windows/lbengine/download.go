package lbengine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbdeployevent"
	"github.com/leafbridge/leafbridge/core/lbevent"
	"github.com/leafbridge/leafbridge/platform/windows/stagingfs"
)

// downloadEngine manages the download and verification of files.
type downloadEngine struct {
	deployment lbdeploy.Deployment
	flow       flowData
	action     actionData
	events     lbevent.Recorder
	state      *engineState
}

// DownloadAndVerifyPackage will attempt to download and verify a package
// file. It uses the provided open package file to read and write data.
//
// If the file already contains the expected data, the download will be
// skipped.
//
// If the file was partially downloaded, the download will be resumed.
func (engine *downloadEngine) DownloadAndVerifyPackage(ctx context.Context, pkg packageData, file stagingfs.PackageFile) error {
	// Prepare a verifier for the package.
	verifier, err := NewFileVerifier(pkg.Definition.Attributes.Hashes.Types()...)
	if err != nil {
		return fmt.Errorf("failed to prepare a file content verifier for package \"%s\": %w", pkg.ID, err)
	}
	if len(verifier.HashTypes()) == 0 {
		return errors.New("packages must provide at least one file hash for verification")
	}

	// Move to the beginning of the file.
	file.Seek(0, io.SeekStart)

	// Read any existing file content into the verifier.
	// This effectively seeks to the end of the file.
	if _, err := verifier.ReadFrom(newReaderWithContext(ctx, file)); err != nil {
		return fmt.Errorf("failed to verify existing file content for package \"%s\": %w", pkg.ID, err)
	}

	// If the file has already been filled with the expected number of
	// bytes, or if it is larger than expected, treat it as a completed
	// download and go immediately to the verification process.
	if existingFileAttributes := verifier.State(); existingFileAttributes.Size >= pkg.Definition.Attributes.Size {
		// Record the file verification result.
		engine.events.Record(lbdeployevent.FileVerification{
			Deployment:  engine.deployment.ID,
			Flow:        engine.flow.ID,
			ActionIndex: engine.action.Index,
			ActionType:  engine.action.Definition.Type,
			FileName:    file.Name,
			Path:        file.Path,
			Expected:    pkg.Definition.Attributes,
			Actual:      existingFileAttributes,
		})

		// Verify the existing file by testing whether its attributes match
		// what was expected.
		if lbdeploy.EqualFileAttributes(pkg.Definition.Attributes, existingFileAttributes) {
			// The file attributes match what was expected.
			// Verification is complete and we're done.
			return nil
		}

		// The file failed verification. Truncate it and try again.
		var reason lbdeployevent.DownloadResetReason
		if existingFileAttributes.Size > pkg.Definition.Attributes.Size {
			reason = lbdeployevent.ExistingFileTooLarge
		} else {
			reason = lbdeployevent.ExistingFileVerificationFailed
		}
		if err := engine.resetFileDownload(lbdeploy.PackageSource{}, file, verifier, reason); err != nil {
			return err
		}
	}

	// Verify that at least one source has been specified.
	if len(pkg.Definition.Sources) == 0 {
		return errors.New("no sources were provided for the package")
	}

	// Start or resume the download. Attempt the download up to two times.
	for attempt := 0; attempt < 2; attempt++ {
		var (
			errs   []error
			source lbdeploy.PackageSource
		)
		for _, candidate := range pkg.Definition.Sources {
			err := engine.downloadPackageFromSource(ctx, candidate, file, verifier)
			if err == nil {
				// The download completed successfully.
				source = candidate
				break
			}
			errs = append(errs, err)
		}

		// If the download failed, so we stop.
		if err := errors.Join(errs...); err != nil {
			return err
		}

		// The download was completed.
		//
		// Ask the verifier for the downloaded file's attributes.
		downloadedFileAttributes := verifier.State()

		// Record the file verification result.
		engine.events.Record(lbdeployevent.FileVerification{
			Deployment:  engine.deployment.ID,
			Flow:        engine.flow.ID,
			ActionIndex: engine.action.Index,
			ActionType:  engine.action.Definition.Type,
			Source:      source,
			FileName:    file.Name,
			Path:        file.Path,
			Expected:    pkg.Definition.Attributes,
			Actual:      downloadedFileAttributes,
		})

		// Verify the downloaded file by testing whether its attributes match
		// what was expected.
		if lbdeploy.EqualFileAttributes(pkg.Definition.Attributes, downloadedFileAttributes) {
			// The file attributes match what was expected.
			// Verification is complete and we're done.
			return nil
		}

		// The file failed verification. Truncate it and try again.
		if attempt == 0 {
			if err := engine.resetFileDownload(source, file, verifier, lbdeployevent.DownloadedFileVerificationFailed); err != nil {
				return err
			}
		}
	}

	// We've exhausted the maximum number of retries, but still failed to
	// produce a downloaded package with the expected file attributes.
	return errors.New("the downloaded package did not pass its file verification checks")
}

func (engine *downloadEngine) downloadPackageFromSource(ctx context.Context, source lbdeploy.PackageSource, file stagingfs.PackageFile, verifier *FileVerifier) (err error) {
	if source.Type != lbdeploy.PackageSourceHTTP {
		return fmt.Errorf("unrecognized package source type: %s", source.Type)
	}

	// Start at an offset when resuming downloads.
	offset := verifier.Size()

	// Prepare an HTTP request. If offset is greater than zero, include a
	// range header.
	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return err
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	// Make the HTTP request.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Record the time that the download started.
	started := time.Now()

	// Examine the status code of the response.
	switch resp.StatusCode {
	case http.StatusOK:
		if offset > 0 {
			offset = 0
			if err := engine.resetFileDownload(source, file, verifier, lbdeployevent.HTTPServerDoesNotSupportResume); err != nil {
				return err
			}
		}
	case http.StatusPartialContent:
		// This indicates that the range header was accepted and the download
		// can be resumed.
	default:
		return fmt.Errorf("the server returned an unexpected status code: %s", resp.Status)
	}

	// Record the start of the download.
	engine.events.Record(lbdeployevent.DownloadStarted{
		Deployment:  engine.deployment.ID,
		Flow:        engine.flow.ID,
		ActionIndex: engine.action.Index,
		ActionType:  engine.action.Definition.Type,
		Source:      source,
		FileName:    file.Name,
		Path:        file.Path,
		Offset:      offset,
	})

	// Download the file, writing to both the file and the verifier.
	var buf [262144]byte // 256 KB
	var downloaded int64
	err = func() error {
		for {
			if err := ctx.Err(); err != nil {
				return err
			}

			chunk, err := resp.Body.Read(buf[:])
			if chunk > 0 {
				downloaded += int64(chunk)
				if _, err := file.Write(buf[:chunk]); err != nil {
					return err
				}
				if _, err := verifier.Write(buf[:chunk]); err != nil {
					return err
				}
			}

			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		}
	}()

	// Record the time that the download stopped.
	stopped := time.Now()

	// Record the end of the download.
	engine.events.Record(lbdeployevent.DownloadStopped{
		Deployment:  engine.deployment.ID,
		Flow:        engine.flow.ID,
		ActionIndex: engine.action.Index,
		ActionType:  engine.action.Definition.Type,
		Source:      source,
		FileName:    file.Name,
		Path:        file.Path,
		Downloaded:  downloaded,
		FileSize:    offset + downloaded,
		Started:     started,
		Stopped:     stopped,
		Err:         err,
	})

	return err
}

func (engine *downloadEngine) resetFileDownload(source lbdeploy.PackageSource, file stagingfs.PackageFile, verifier *FileVerifier, reason lbdeployevent.DownloadResetReason) error {
	// Record the reset of the download.
	engine.events.Record(lbdeployevent.DownloadReset{
		Deployment:  engine.deployment.ID,
		Flow:        engine.flow.ID,
		ActionIndex: engine.action.Index,
		ActionType:  engine.action.Definition.Type,
		Source:      source,
		FileName:    file.Name,
		Path:        file.Path,
		Reason:      reason,
	})

	// Seek to the beginning of the file.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// Truncate the file.
	if err := file.Truncate(0); err != nil {
		return err
	}

	// Reset the file verifier.
	verifier.Reset()

	return nil
}

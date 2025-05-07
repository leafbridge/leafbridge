package filetime

import (
	"os"
	"syscall"
	"time"

	"github.com/gentlemanautomaton/volmgmt/fileapi"
)

// SetFileModificationTime attempts to set the file modification time for
// the open file.
func SetFileModificationTime(file *os.File, modified time.Time) error {
	update := fileapi.BasicInfo{
		LastWriteTime: modified, // The last time data was written to the file.
		ChangeTime:    modified, // The last time file attributes were changed.
	}

	return fileapi.SetFileInformationByHandle(syscall.Handle(file.Fd()), update)
}

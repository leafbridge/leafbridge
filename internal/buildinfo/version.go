package buildinfo

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// Version holds a version number string in the form Major.Minor.Patch.Build,
// as in "1.2.3.4".
type Version string

// GetVersion does its best to create a version number from the given build
// info.
//
// It returns the version number of the main package's module, if it is
// avaliable and it is not a pseudoversion.
//
// If a module version number is not available, it creates a version from the
// most recent commit timestamp, or from the current time as a last resort.
// In such a case, the version will be in the format returned by
// [VersionForTime].
func GetVersion(info *debug.BuildInfo) Version {
	if v := info.Main.Version; v != "" && semver.IsValid(v) && !module.IsPseudoVersion(v) {
		return Version(strings.TrimPrefix(semver.Canonical(v), "v"))
	}

	commit := ParseCommit(info.Settings)

	timestamp := commit.Time.UTC()
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	return VersionForTime(timestamp)
}

// VersionForTime creates a version number from the given time. The returned
// version will be in the format "0.0.YYYY.MMDDT", where T is a value between
// 0 and 9 that increments as time passes through a day.
//
// For example, a timestamp for 2025-12-12T23:45:00Z would return a [Version]
// of "0.0.2025.12129".
func VersionForTime(t time.Time) Version {
	return Version(fmt.Sprintf("0.0.%04d.%02d%02d%d",
		t.Year(),
		t.Month(),
		t.Day(),
		tenthHour(t),
	))
}

func tenthHour(t time.Time) int {
	return ((t.Hour()*60 + t.Minute()) * 10) / 1440
}

// Major returns the major number from v.
func (v Version) Major() int {
	return getVersionSegment(v, 0)
}

// Minor returns the minor number from v.
func (v Version) Minor() int {
	return getVersionSegment(v, 1)
}

// Patch returns the patch number from v.
func (v Version) Patch() int {
	return getVersionSegment(v, 2)
}

// Build returns the build number from v.
func (v Version) Build() int {
	return getVersionSegment(v, 3)
}

func getVersionSegment(v Version, index int) int {
	parts := strings.Split(string(v), ".")
	if len(parts) <= index {
		return 0
	}

	value, err := strconv.ParseInt(parts[index], 10, 32)
	if err != nil {
		return 0
	}

	return int(value)
}

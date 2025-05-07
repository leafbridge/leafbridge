package buildinfo

import (
	"runtime/debug"
	"strconv"
	"time"
)

// Commit stores information about the version control system commit that
// was used to build the program.
type Commit struct {
	Time     time.Time
	Revision string
	Modified bool
}

// ParseCommit looks for commit information within the given build settings.
func ParseCommit(settings []debug.BuildSetting) Commit {
	var commit Commit

	for _, setting := range settings {
		if setting.Key == "vcs.time" && setting.Value != "" {
			commit.Time, _ = time.Parse(time.RFC3339, setting.Value)
		}
		if setting.Key == "vcs.revision" && setting.Value != "" {
			commit.Revision = setting.Value
		}
		if setting.Key == "vcs.modified" && setting.Value != "" {
			commit.Modified, _ = strconv.ParseBool(setting.Value)
		}
	}

	return commit
}

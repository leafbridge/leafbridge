package lbengine

import (
	"fmt"

	"github.com/gentlemanautomaton/winproc"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// NumberOfRunningProcesses returns the number of processes running on the
// local system that match the given criteria.
func NumberOfRunningProcesses(match lbdeploy.ProcessMatch) (n int, err error) {
	filter, err := buildProcessFilter(match)
	if err != nil {
		return 0, err
	}

	procs, err := winproc.List(winproc.Include(filter))
	if err != nil {
		return 0, err
	}

	return len(procs), nil
}

// buildProcessFilter prepares a Windows process filter for the given
// criteria.
func buildProcessFilter(match lbdeploy.ProcessMatch) (winproc.Filter, error) {
	if len(match.Any) > 0 {
		var filters []winproc.Filter
		for i, submatch := range match.Any {
			subfilter, err := buildProcessFilter(submatch)
			if err != nil {
				return nil, fmt.Errorf("Match Any [%d]: %w", i, err)
			}
			filters = append(filters, subfilter)
		}
		return winproc.MatchAny(filters...), nil
	}

	if len(match.All) > 0 {
		var filters []winproc.Filter
		for i, submatch := range match.All {
			subfilter, err := buildProcessFilter(submatch)
			if err != nil {
				return nil, fmt.Errorf("Match All [%d]: %w", i, err)
			}
			filters = append(filters, subfilter)
		}
		return winproc.MatchAll(filters...), nil
	}

	switch match.Attribute {
	case lbdeploy.ProcessName:
		switch match.Type {
		case lbdeploy.MatchEquals:
			return winproc.EqualsName(match.Value), nil
		case lbdeploy.MatchContains:
			return winproc.ContainsName(match.Value), nil
		case "":
			return nil, fmt.Errorf("a process match type was not provided")
		default:
			return nil, fmt.Errorf("the process match type \"%s\" is not recognized", match.Type)
		}
	case "":
		return nil, fmt.Errorf("a process attribute was not provided")
	default:
		return nil, fmt.Errorf("the process attribute \"%s\" is not recognized", match.Attribute)
	}
}

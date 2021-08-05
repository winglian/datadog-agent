package so

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strconv"
)

// AllLibraries represents a filter that matches all shared libraries
var AllLibraries = regexp.MustCompile(`\.so($|\.)`)

// Find returns the host-resolved paths of all shared libraries matching the given filter
// It does so by iterating over all /proc/<PID>/maps and /proc/<PID>/mountinfo files in the host
func Find(procRoot string, filter *regexp.Regexp) []Library {
	finder := newFinder(procRoot)
	return finder.Find(filter)
}

// FromPID returns all shared libraries matching the given filter that are mapped into memory by a given PID
func FromPID(procRoot string, pid int32, filter *regexp.Regexp) []Library {
	pidPath := filepath.Join(procRoot, strconv.Itoa(int(pid)))
	buffer := bufio.NewReader(nil)
	libs := getSharedLibraries(pidPath, buffer, filter)
	if len(libs) == 0 {
		return nil
	}

	pathResolver := newPathResolver(procRoot, buffer)
	mountInfo := getMountInfo(pidPath, buffer)
	var result []Library
	for _, lib := range libs {
		if hostPath := pathResolver.Resolve(lib, mountInfo); hostPath != "" {
			result = append(result, Library{
				PIDPath: pidPath,
				LibPath: hostPath,
			})
		}
	}
	return result
}

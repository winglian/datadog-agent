package so

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
)

type finder struct {
	procRoot     string
	pathResolver *pathResolver
	buffer       *bufio.Reader
}

func newFinder(procRoot string) *finder {
	buffer := bufio.NewReader(nil)
	return &finder{
		procRoot:     procRoot,
		pathResolver: newPathResolver(procRoot, buffer),
		buffer:       buffer,
	}
}

func (f *finder) Find(filter *regexp.Regexp) []*ByPID {
	resultByPID := make(map[string]*ByPID)
	resolved := make(map[key]string)
	iteratePIDS(f.procRoot, func(pidPath string, info os.FileInfo, mntNS ns) {
		matches := getSharedLibraries(pidPath, f.buffer, filter)
		if len(matches) == 0 {
			return
		}

		byPID := &ByPID{PIDPath: pidPath, Libraries: make([]string, 0, len(matches))}
		var mountInfo *mountInfo
		for _, lib := range matches {
			pathKey := key{mntNS, lib}
			resolvedPath, ok := resolved[pathKey]
			if ok {
				byPID.Libraries = append(byPID.Libraries, resolvedPath)
				continue
			}

			if mountInfo == nil {
				mountInfo = getMountInfo(pidPath, f.buffer)
			}
			if mountInfo == nil {
				return
			}

			if resolvedPath := f.pathResolver.Resolve(lib, mountInfo); resolvedPath != "" {
				byPID.Libraries = append(byPID.Libraries, resolvedPath)
				resolved[pathKey] = resolvedPath
			}
		}
		resultByPID[pidPath] = byPID
	})

	result := make([]*ByPID, len(resultByPID))
	for _, r := range resultByPID {
		result = append(result, r)
	}

	return result
}

func iteratePIDS(procRoot string, fn callback) {
	w := newWalker(procRoot, fn)
	filepath.Walk(procRoot, filepath.WalkFunc(w.walk))
}

// key is used to keep track of which libraries have been resolved
type key struct {
	ns   ns
	name string
}

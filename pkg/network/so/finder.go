package so

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
)

type Library struct {
	PIDPath string
	LibPath string
}

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

func (f *finder) Find(filter *regexp.Regexp) []Library {
	var result []Library
	resolved := make(map[key]string)

	iteratePIDS(f.procRoot, func(pidPath string, info os.FileInfo, mntNS ns) {
		var mountInfo *mountInfo
		matches := getSharedLibraries(pidPath, f.buffer, filter)
		for _, lib := range matches {
			pathKey := key{mntNS, lib}
			resolvedPath, ok := resolved[pathKey]
			if ok {
				result = append(result, Library{
					PIDPath: pidPath,
					LibPath: resolvedPath,
				})
				continue
			}

			if mountInfo == nil {
				mountInfo = getMountInfo(pidPath, f.buffer)
			}
			if mountInfo == nil {
				return
			}

			if resolvedPath := f.pathResolver.Resolve(lib, mountInfo); resolvedPath != "" {
				result = append(result, Library{
					PIDPath: pidPath,
					LibPath: resolvedPath,
				})
				resolved[pathKey] = resolvedPath
			}
		}
	})
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

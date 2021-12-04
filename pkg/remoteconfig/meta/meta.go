package meta

import _ "embed"

var (
	//go:embed 1.director.json
	rootDirector1 []byte

	//go:embed 1.config.json
	rootConfig1 []byte
)

type EmbeddedRoot []byte
type EmbeddedRoots map[uint64]EmbeddedRoot

var rootsDirector = EmbeddedRoots{
	1: rootDirector1,
}

var rootsConfig = EmbeddedRoots{
	1: rootConfig1,
}

func RootsDirector() EmbeddedRoots {
	return rootsDirector
}

func RootsConfig() EmbeddedRoots {
	return rootsConfig
}

func (roots EmbeddedRoots) Last() EmbeddedRoot {
	return roots[uint64(len(roots))]
}

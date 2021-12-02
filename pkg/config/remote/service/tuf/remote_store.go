package tuf

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/config/remote/service/meta"
	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/theupdateframework/go-tuf/client"
)

type version uint64
type role string
type path string

const (
	roleRoot      role = "root"
	roleTargets   role = "targets"
	roleSnapshot  role = "snapshot"
	roleTimestamp role = "timestamp"
)

// remoteStore implements go-tuf's RemoteStore
// Its goal is to serve TUF metadata updates comming to the backend in a way go-tuf understands
// See https://pkg.go.dev/github.com/theupdateframework/go-tuf@v0.0.0-20211130162850-52193a283c30/client#RemoteStore
type remoteStore struct {
	metas   map[role]map[version][]byte
	targets map[path][]byte
}

func newRemoteStore() remoteStore {
	return remoteStore{
		metas: make(map[role]map[version][]byte),
	}
}

func (s *remoteStore) resetTargets() {
	s.targets = make(map[path][]byte)
}

func (s *remoteStore) resetRole(r role) {
	s.metas[r] = make(map[version][]byte)
}

func (s *remoteStore) latestVersion(r role) version {
	latestVersion := version(0)
	for v, _ := range s.metas[r] {
		if v > latestVersion {
			latestVersion = v
		}
	}
	return latestVersion
}

// GetMeta implements go-tuf's RemoteStore.GetMeta
// See https://pkg.go.dev/github.com/theupdateframework/go-tuf@v0.0.0-20211130162850-52193a283c30/client#RemoteStore
func (s *remoteStore) GetMeta(path string) (io.ReadCloser, int64, error) {
	metaPath, err := parseMetaPath(path)
	if err != nil {
		return nil, 0, err
	}
	roleVersions, roleFound := s.metas[metaPath.role]
	if !roleFound {
		return nil, 0, client.ErrNotFound{File: path}
	}
	version := metaPath.version
	if !metaPath.versionSet {
		if metaPath.role != roleTimestamp {
			return nil, 0, client.ErrNotFound{File: path}
		}
		version = s.latestVersion(metaPath.role)
	}
	requestedVersion, versionFound := roleVersions[version]
	if !versionFound {
		return nil, 0, client.ErrNotFound{File: path}
	}
	return ioutil.NopCloser(bytes.NewReader(requestedVersion)), int64(len(requestedVersion)), nil
}

// GetMeta implements go-tuf's RemoteStore.GetTarget
// See https://pkg.go.dev/github.com/theupdateframework/go-tuf@v0.0.0-20211130162850-52193a283c30/client#RemoteStore
func (s *remoteStore) GetTarget(targetPath string) (stream io.ReadCloser, size int64, err error) {
	target, found := s.targets[path(targetPath)]
	if !found {
		return nil, 0, client.ErrNotFound{File: targetPath}
	}
	return ioutil.NopCloser(bytes.NewReader(target)), int64(len(target)), nil

}

type remoteStoreDirector struct {
	remoteStore
}

func newRemoteStoreDirector() *remoteStoreDirector {
	s := newRemoteStore()
	s.metas[roleRoot][1] = getDirectorRoot()
	return &remoteStoreDirector{remoteStore: s}
}

func (sd *remoteStoreDirector) update(update *pbgo.LatestConfigsResponse) {
	if update == nil {
		return
	}
	sd.resetTargets()
	for _, target := range update.TargetFiles {
		sd.targets[path(target.Path)] = target.Raw
	}
	if update.DirectorMetas == nil {
		return
	}
	metas := update.DirectorMetas
	for _, root := range metas.Roots {
		sd.metas[roleRoot][version(root.Version)] = root.Raw
	}
	if metas.Timestamp != nil {
		sd.resetRole(roleTimestamp)
		sd.metas[roleTimestamp][version(metas.Timestamp.Version)] = metas.Timestamp.Raw
	}
	if metas.Snapshot != nil {
		sd.resetRole(roleSnapshot)
		sd.metas[roleSnapshot][version(metas.Snapshot.Version)] = metas.Snapshot.Raw
	}
	if metas.Targets != nil {
		sd.resetRole(roleTargets)
		sd.metas[roleTargets][version(metas.Targets.Version)] = metas.Targets.Raw
	}
}

type remoteStoreConfig struct {
	remoteStore
}

func newRemoteStoreConfig() *remoteStoreConfig {
	s := newRemoteStore()
	s.metas[roleRoot][1] = getConfigRoot()
	return &remoteStoreConfig{remoteStore: s}
}

func (sc *remoteStoreConfig) update(update *pbgo.LatestConfigsResponse) {
	if update == nil {
		return
	}
	sc.resetTargets()
	for _, target := range update.TargetFiles {
		sc.targets[path(target.Path)] = target.Raw
	}
	if update.ConfigMetas == nil {
		return
	}
	metas := update.ConfigMetas
	for _, root := range metas.Roots {
		sc.metas[roleRoot][version(root.Version)] = root.Raw
	}
	for _, delegatedMeta := range metas.DelegatedTargets {
		role := role(delegatedMeta.Role)
		sc.resetRole(role)
		sc.metas[role][version(delegatedMeta.Version)] = delegatedMeta.Raw
	}
	if metas.Timestamp != nil {
		sc.resetRole(roleTimestamp)
		sc.metas[roleTimestamp][version(metas.Timestamp.Version)] = metas.Timestamp.Raw
	}
	if metas.Snapshot != nil {
		sc.resetRole(roleSnapshot)
		sc.metas[roleSnapshot][version(metas.Snapshot.Version)] = metas.Snapshot.Raw
	}
	if metas.TopTargets != nil {
		sc.resetRole(roleTargets)
		sc.metas[roleTargets][version(metas.TopTargets.Version)] = metas.TopTargets.Raw
	}
}

// TODO: clean
func getDirectorRoot() []byte {
	if directorRoot := config.Datadog.GetString("remote_configuration.director_root"); directorRoot != "" {
		return []byte(directorRoot)
	}
	return meta.RootDirector
}

// TODO: clean
func getConfigRoot() []byte {
	if configRoot := config.Datadog.GetString("remote_configuration.config_root"); configRoot != "" {
		return []byte(configRoot)
	}
	return meta.RootConfig
}

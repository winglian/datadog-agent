package uptane

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/theupdateframework/go-tuf/client"
)

type partialRemoteStore struct {
	remote *pbgo.ClientGetConfigsResponse
}

// GetMeta implements go-tuf's RemoteStore.GetMeta
func (s *partialRemoteStore) GetMeta(path string) (io.ReadCloser, int64, error) {
	metaPath, err := parseMetaPath(path)
	if err != nil {
		return nil, 0, err
	}
	switch metaPath.role {
	case roleRoot:
		if !metaPath.versionSet {
			return nil, 0, client.ErrNotFound{File: path}
		}
		for _, root := range s.remote.Roots {
			if root.Version == metaPath.version {
				return ioutil.NopCloser(bytes.NewReader(root.Raw)), int64(len(root.Raw)), nil
			}
		}
		return nil, 0, client.ErrNotFound{File: path}
	}
	return nil, 0, client.ErrNotFound{File: path}
}

// GetMeta implements go-tuf's RemoteStore.GetTarget
func (s *partialRemoteStore) GetTarget(targetPath string) (stream io.ReadCloser, size int64, err error) {
	for _, target := range s.remote.ConfigFiles {
		if target.Path == targetPath {
			return ioutil.NopCloser(bytes.NewReader(target.Raw)), int64(len(target.Raw)), nil
		}
	}
	return nil, 0, client.ErrNotFound{File: targetPath}
}

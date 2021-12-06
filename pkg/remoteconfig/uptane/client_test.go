package uptane

import (
	"crypto/sha256"
	"encoding/json"
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/meta"
	"github.com/stretchr/testify/assert"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/sign"
)

func TestClientState(t *testing.T) {
	testRepository1 := newTestRepository(1)
	config.Datadog.Set("remote_configuration.director_root", testRepository1.directorRoot)
	config.Datadog.Set("remote_configuration.config_root", testRepository1.configRoot)

	db := getTestDB()
	client, err := NewClient(db, "testcachekey", 2)
	assert.NoError(t, err)

	expectedDefaultState := State{
		ConfigSnapshotVersion: 0,
		ConfigRootVersion:     meta.RootsConfig().LastVersion(),
		DirectorRootVersion:   meta.RootsDirector().LastVersion(),
	}
	clientState, err := client.State()
	assert.NoError(t, err)
	assert.Equal(t, expectedDefaultState, clientState)
	_, err = client.TargetsMeta()
	assert.Error(t, err)

	err = client.Update(testRepository1.toUpdate())
	assert.NoError(t, err)
}

func TestClientVerifyTUF(t *testing.T) {

}

func TestClientVerifyUptane(t *testing.T) {

}

func TestClientVerifyOrgID(t *testing.T) {

}

func generateKey() *sign.PrivateKey {
	key, _ := sign.GenerateEd25519Key()
	return key
}

type testRepositories struct {
	configTimestampKey   *sign.PrivateKey
	configTargetsKey     *sign.PrivateKey
	configSnapshotKey    *sign.PrivateKey
	configRootKey        *sign.PrivateKey
	directorTimestampKey *sign.PrivateKey
	directorTargetsKey   *sign.PrivateKey
	directorSnapshotKey  *sign.PrivateKey
	directorRootKey      *sign.PrivateKey

	configTimestampVersion   int
	configTargetsVersion     int
	configSnapshotVersion    int
	configRootVersion        int
	directorTimestampVersion int
	directorTargetsVersion   int
	directorSnapshotVersion  int
	directorRootVersion      int

	configTimestamp   []byte
	configTargets     []byte
	configSnapshot    []byte
	configRoot        []byte
	directorTimestamp []byte
	directorTargets   []byte
	directorSnapshot  []byte
	directorRoot      []byte
}

func newTestRepository(version int) testRepositories {
	repos := testRepositories{
		configTimestampKey:   generateKey(),
		configTargetsKey:     generateKey(),
		configSnapshotKey:    generateKey(),
		configRootKey:        generateKey(),
		directorTimestampKey: generateKey(),
		directorTargetsKey:   generateKey(),
		directorSnapshotKey:  generateKey(),
		directorRootKey:      generateKey(),
	}
	repos.configRootVersion = version
	repos.configTimestampVersion = 10 + version
	repos.configTargetsVersion = 100 + version
	repos.configSnapshotVersion = 1000 + version
	repos.directorRootVersion = version
	repos.directorTimestampVersion = 20 + version
	repos.directorTargetsVersion = 200 + version
	repos.directorSnapshotVersion = 2000 + version
	repos.configRoot = generateRoot(repos.configRootKey, version, repos.configTimestampKey, repos.configTargetsKey, repos.configSnapshotKey)
	repos.configTargets = generateTargets(repos.configTargetsKey, 100+version)
	repos.configSnapshot = generateSnapshot(repos.configSnapshotKey, 1000+version, repos.configTargetsVersion)
	repos.configTimestamp = generateTimestamp(repos.configTimestampKey, 10+version, repos.configSnapshotVersion, repos.configSnapshot)
	repos.directorRoot = generateRoot(repos.directorRootKey, version, repos.directorTimestampKey, repos.directorTargetsKey, repos.directorSnapshotKey)
	repos.directorTargets = generateTargets(repos.directorTargetsKey, 200+version)
	repos.directorSnapshot = generateSnapshot(repos.directorSnapshotKey, 2000+version, repos.directorTargetsVersion)
	repos.directorTimestamp = generateTimestamp(repos.directorTimestampKey, 20+version, repos.directorSnapshotVersion, repos.directorSnapshot)
	return repos
}

func (r testRepositories) toUpdate() *pbgo.LatestConfigsResponse {
	return &pbgo.LatestConfigsResponse{
		ConfigMetas: &pbgo.ConfigMetas{
			Roots:      []*pbgo.TopMeta{{Version: uint64(r.configRootVersion), Raw: r.configRoot}},
			Timestamp:  &pbgo.TopMeta{Version: uint64(r.configTimestampVersion), Raw: r.configTimestamp},
			Snapshot:   &pbgo.TopMeta{Version: uint64(r.configSnapshotVersion), Raw: r.configSnapshot},
			TopTargets: &pbgo.TopMeta{Version: uint64(r.configTargetsVersion), Raw: r.configTargets},
		},
		DirectorMetas: &pbgo.DirectorMetas{
			Roots:     []*pbgo.TopMeta{{Version: uint64(r.directorRootVersion), Raw: r.directorRoot}},
			Timestamp: &pbgo.TopMeta{Version: uint64(r.directorTimestampVersion), Raw: r.directorTimestamp},
			Snapshot:  &pbgo.TopMeta{Version: uint64(r.directorSnapshotVersion), Raw: r.directorSnapshot},
			Targets:   &pbgo.TopMeta{Version: uint64(r.directorTargetsVersion), Raw: r.directorTargets},
		},
	}
}

func generateRoot(key *sign.PrivateKey, version int, timestampKey *sign.PrivateKey, targetsKey *sign.PrivateKey, snapshotKey *sign.PrivateKey) []byte {
	root := data.NewRoot()
	root.Version = version
	root.Expires = time.Now().Add(1 * time.Hour)
	root.AddKey(key.PublicData())
	root.AddKey(timestampKey.PublicData())
	root.AddKey(targetsKey.PublicData())
	root.AddKey(snapshotKey.PublicData())
	root.Roles["root"] = &data.Role{
		KeyIDs:    key.PublicData().IDs(),
		Threshold: 1,
	}
	root.Roles["timestamp"] = &data.Role{
		KeyIDs:    timestampKey.PublicData().IDs(),
		Threshold: 1,
	}
	root.Roles["targets"] = &data.Role{
		KeyIDs:    targetsKey.PublicData().IDs(),
		Threshold: 1,
	}
	root.Roles["snapshot"] = &data.Role{
		KeyIDs:    snapshotKey.PublicData().IDs(),
		Threshold: 1,
	}
	signedRoot, _ := sign.Marshal(&root, key.Signer())
	serializedRoot, _ := json.Marshal(signedRoot)
	return serializedRoot
}

func generateTimestamp(key *sign.PrivateKey, version int, snapshotVersion int, snapshot []byte) []byte {
	meta := data.NewTimestamp()
	meta.Expires = time.Now().Add(1 * time.Hour)
	meta.Version = version
	meta.Meta["snapshot.json"] = data.TimestampFileMeta{Version: snapshotVersion, FileMeta: data.FileMeta{Length: int64(len(snapshot)), Hashes: data.Hashes{
		"sha256": hashSha256(snapshot),
	}}}
	signed, _ := sign.Marshal(&meta, key.Signer())
	serialized, _ := json.Marshal(signed)
	return serialized
}

func generateTargets(key *sign.PrivateKey, version int) []byte {
	meta := data.NewTargets()
	meta.Expires = time.Now().Add(1 * time.Hour)
	meta.Version = version
	signed, _ := sign.Marshal(&meta, key.Signer())
	serialized, _ := json.Marshal(signed)
	return serialized
}

func generateSnapshot(key *sign.PrivateKey, version int, targetsVersion int) []byte {
	meta := data.NewSnapshot()
	meta.Expires = time.Now().Add(1 * time.Hour)
	meta.Version = version
	meta.Meta["targets.json"] = data.SnapshotFileMeta{Version: targetsVersion}

	signed, _ := sign.Marshal(&meta, key.Signer())
	serialized, _ := json.Marshal(signed)
	return serialized
}

func hashSha256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

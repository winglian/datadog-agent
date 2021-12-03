package uptane

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/config/remote/store"
	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/theupdateframework/go-tuf/client"
)

type Client struct {
	orgIDTargetPrefix string

	configLocalStore  *localStoreConfig
	configRemoteStore *remoteStoreConfig
	configTUFClient   *client.Client

	directorLocalStore  *localStoreDirector
	directorRemoteStore *remoteStoreDirector
	directorTUFClient   *client.Client
}

func NewClient(orgID int, localStore *store.Store) (*Client, error) {
	c := &Client{
		orgIDTargetPrefix:   fmt.Sprintf("%d/", orgID),
		configLocalStore:    newLocalStoreConfig(localStore),
		configRemoteStore:   newRemoteStoreConfig(),
		directorLocalStore:  newLocalStoreDirector(localStore),
		directorRemoteStore: newRemoteStoreDirector(),
	}
	c.configTUFClient = client.NewClient(c.configLocalStore, c.configRemoteStore)
	c.directorTUFClient = client.NewClient(c.directorLocalStore, c.directorRemoteStore)
	return c, nil
}

func (c *Client) Update(response *pbgo.LatestConfigsResponse) error {
	err := c.updateRepos(response)
	if err != nil {
		return err
	}
	err = c.verifyOrgID()
	if err != nil {
		return err
	}
	return c.verifyUptane()
}

func (c *Client) updateRepos(response *pbgo.LatestConfigsResponse) error {
	c.directorRemoteStore.update(response)
	c.configRemoteStore.update(response)
	_, err := c.directorTUFClient.Update()
	if err != nil {
		return err
	}
	_, err = c.configTUFClient.Update()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) verifyOrgID() error {
	directorTargets, err := c.directorTUFClient.Targets()
	if err != nil {
		return err
	}
	for targetPath := range directorTargets {
		if !strings.HasPrefix(targetPath, c.orgIDTargetPrefix) {
			return fmt.Errorf("director target '%s' does not have the correct orgID", targetPath)
		}
	}
	return nil
}

func (c *Client) verifyUptane() error {
	directorTargets, err := c.directorTUFClient.Targets()
	if err != nil {
		return err
	}
	for targetPath, targetMeta := range directorTargets {
		configTargetMeta, err := c.configTUFClient.Target(targetPath)
		if err != nil {
			return fmt.Errorf("failed to find target '%s' in config repository", targetPath)
		}
		if configTargetMeta.Length != targetMeta.Length {
			return fmt.Errorf("target '%s' has size %d in directory repository and %d in config repository", targetPath, configTargetMeta.Length, targetMeta.Length)
		}
		for kind, directorHash := range targetMeta.Hashes {
			configHash, found := configTargetMeta.Hashes[kind]
			if !found {
				return fmt.Errorf("hash '%s' found in directory repository and not in config repository", directorHash)
			}
			if !bytes.Equal([]byte(directorHash), []byte(configHash)) {
				return fmt.Errorf("directory hash '%s' is not equal to config repository '%s'", string(directorHash), string(configHash))
			}
		}
	}
	return nil
}

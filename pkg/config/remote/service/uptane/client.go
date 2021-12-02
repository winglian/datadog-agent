package uptane

import (
	"bytes"
	"fmt"

	"github.com/DataDog/datadog-agent/pkg/config/remote/store"
	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/theupdateframework/go-tuf/client"
)

type Client struct {
	configLocalStore  *localStoreConfig
	configRemoteStore *remoteStoreConfig
	configTUFClient   *client.Client

	directorLocalStore  *localStoreDirector
	directorRemoteStore *remoteStoreDirector
	directorTUFClient   *client.Client
}

func NewClient(localStore *store.Store) (*Client, error) {
	c := &Client{
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
	err = c.verifyUptane(response)
	return nil
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

func (c *Client) verifyUptane(response *pbgo.LatestConfigsResponse) error {
	for _, target := range response.TargetFiles {
		name := tuf.TrimHash(target.Path)

		log.Debugf("Considering director target %s", name)
		directorTarget, err := s.director.Target(name)
		if err != nil {
			return fmt.Errorf("failed to find target '%s' in director repository", name)
		}

		configTarget, err := s.config.Target(name)
		if err != nil {
			return fmt.Errorf("failed to find target '%s' in config repository", name)
		}

		if configTarget.Length != directorTarget.Length {
			return fmt.Errorf("target '%s' has size %d in directory repository and %d in config repository", name, configTarget.Length, directorTarget.Length)
		}

		for kind, directorHash := range directorTarget.Hashes {
			configHash, found := configTarget.Hashes[kind]
			if !found {
				return fmt.Errorf("hash '%s' found in directory repository and not in config repository", directorHash)
			}

			if !bytes.Equal([]byte(directorHash), []byte(configHash)) {
				return fmt.Errorf("directory hash '%s' is not equal to config repository '%s'", string(directorHash), string(configHash))
			}
		}
	}
}

func (c *Client) verifyExtra(response *pbgo.LatestConfigsResponse) error {

}

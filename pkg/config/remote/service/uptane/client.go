package uptane

import (
	"encoding/json"
	"fmt"

	"github.com/DataDog/datadog-agent/pkg/config/remote/store"
	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/theupdateframework/go-tuf/client"
)

type Client struct {
	orgID int

	configLocalStore  *localStoreConfig
	configRemoteStore *remoteStoreConfig
	configTUFClient   *client.Client

	directorLocalStore  *localStoreDirector
	directorRemoteStore *remoteStoreDirector
	directorTUFClient   *client.Client
}

func NewClient(orgID int, localStore *store.Store) (*Client, error) {
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
	err := c.verifyOrgID(response)
	if err != nil {
		return err
	}
	err = c.updateRepos(response)
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

func (c *Client) verifyOrgID(response *pbgo.LatestConfigsResponse) error {
	c.directorTUFClient.Targets()

	if response.DirectorMetas.Targets == nil {
		return fmt.Errorf("director target %s has no custom field", name)
	}

	var custom targetCustomOrgID
	if err := json.Unmarshal([]byte(*directorTarget.Custom), &custom); err != nil {
		return fmt.Errorf("failed to decode target custom for %s: %w", name, err)
	}

	if custom.OrgID != s.orgID {
		return fmt.Errorf("unexpected custom organization id: %s", custom.OrgID)
	}
}

// func (c *Client) verifyUptane() error {
// 	for _, target := range  {
// 		name := trimTargetPathHash(target.Path)

// 		directorTarget, err := c.directorTUFClient.Target(name)
// 		if err != nil {
// 			return fmt.Errorf("failed to find target '%s' in director repository", name)
// 		}

// 		configTarget, err := c.configTUFClient.Target(name)
// 		if err != nil {
// 			return fmt.Errorf("failed to find target '%s' in config repository", name)
// 		}

// 		if configTarget.Length != directorTarget.Length {
// 			return fmt.Errorf("target '%s' has size %d in directory repository and %d in config repository", name, configTarget.Length, directorTarget.Length)
// 		}

// 		for kind, directorHash := range directorTarget.Hashes {
// 			configHash, found := configTarget.Hashes[kind]
// 			if !found {
// 				return fmt.Errorf("hash '%s' found in directory repository and not in config repository", directorHash)
// 			}

// 			if !bytes.Equal([]byte(directorHash), []byte(configHash)) {
// 				return fmt.Errorf("directory hash '%s' is not equal to config repository '%s'", string(directorHash), string(configHash))
// 			}
// 		}
// 	}
// 	return nil
// }

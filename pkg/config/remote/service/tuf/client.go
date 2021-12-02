package tuf

import (
	"github.com/DataDog/datadog-agent/pkg/config/remote/store"
	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
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
	// c.remote.update(response)
	// _, err := c.Client.Update()
	// TODO: update + full verify
	return nil
}

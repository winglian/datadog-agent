package service

import (
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
)

type client struct {
	expireAt time.Time
	pbClient *pbgo.Client
}

func (c *client) expired() bool {
	return time.Now().After(c.expireAt)
}

type clients struct {
	sync.Mutex

	clientsTTL time.Duration
	clients    map[string]*client
}

func newClients(clientsTTL time.Duration) *clients {
	return &clients{
		clientsTTL: clientsTTL,
		clients:    make(map[string]*client),
	}
}

// seen marks the given client as active
func (c *clients) seen(pbClient *pbgo.Client) {
	c.Lock()
	defer c.Unlock()
	c.clients[pbClient.Id] = &client{
		expireAt: time.Now().Add(c.clientsTTL),
		pbClient: pbClient,
	}
}

// activeClients returns the list of active clients
func (c *clients) activeClients() []*pbgo.Client {
	c.Lock()
	defer c.Unlock()
	var activeClients []*pbgo.Client
	for id, client := range c.clients {
		if client.expired() {
			delete(c.clients, id)
			continue
		}
		activeClients = append(activeClients, client.pbClient)
	}
	return activeClients
}

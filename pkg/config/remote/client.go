package config

import (
	"context"
	"fmt"
	"time"

	"github.com/DataDog/datadog-agent/pkg/api/security"
	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/DataDog/datadog-agent/pkg/util/grpc"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	ctx          context.Context
	facts        Facts
	pollInterval time.Duration
	grpc         pbgo.AgentSecureClient
	close        func()
}

type Facts struct {
	ID      string
	Name    string
	Version string
}

func NewClient(ctx context.Context, facts Facts) (*Client, error) {
	token, err := security.FetchAuthToken()
	if err != nil {
		return nil, errors.Wrap(err, "could not acquire agent auth token")
	}
	ctx, close := context.WithCancel(ctx)
	md := metadata.MD{
		"authorization": []string{fmt.Sprintf("Bearer %s", token)},
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	grpcClient, err := grpc.GetDDAgentSecureClient(ctx)
	if err != nil {
		close()
		return nil, err
	}
	c := &Client{
		ctx:   ctx,
		facts: facts,
		grpc:  grpcClient,
		close: close,
	}
	go c.pollLoop()
	return c, nil
}

func (c *Client) Close() {
	c.close()
}

func (c *Client) pollLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(c.pollInterval):
			err := c.poll()
			if err != nil {
				log.Errorf("could not poll remote-config agent service: %v", err)
			}
		}
	}
}

func (c *Client) poll() error {
	c.grpc.ClientGetConfigs(c.ctx, &pbgo.ClientGetConfigsRequest{
		Client: &pbgo.Client{
			Id:      c.facts.ID,
			Name:    c.facts.Name,
			Version: c.facts.Version,
			State:   &pbgo.ClientState{},
		},
	})
}

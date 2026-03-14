// Package gcp provides a Google Cloud Platform client using Application Default Credentials.
package gcp

import (
	"context"
	"fmt"
	"sync"

	crm "google.golang.org/api/cloudresourcemanager/v3"
	iam "google.golang.org/api/iam/v1"
	oauth2api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	pubsubapi "google.golang.org/api/pubsub/v1"
	serviceusage "google.golang.org/api/serviceusage/v1"
)

// Client implements ports.GCPClient using Google's official Go SDK with ADC.
type Client struct {
	ctx  context.Context
	opts []option.ClientOption

	mu           sync.Mutex
	crmService   *crm.Service
	suService    *serviceusage.Service
	iamService   *iam.Service
	psService    *pubsubapi.Service
	oauthService *oauth2api.Service
}

// NewClient creates a new GCP client using Application Default Credentials.
func NewClient(ctx context.Context) (*Client, error) {
	return &Client{ctx: ctx}, nil
}

// NewClientWithOptions creates a new GCP client with custom options (for testing).
func NewClientWithOptions(ctx context.Context, opts ...option.ClientOption) (*Client, error) {
	return &Client{ctx: ctx, opts: opts}, nil
}

func (c *Client) resourceManager() (*crm.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.crmService == nil {
		svc, err := crm.NewService(c.ctx, c.opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create Resource Manager client: %w", err)
		}
		c.crmService = svc
	}
	return c.crmService, nil
}

func (c *Client) serviceUsage() (*serviceusage.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.suService == nil {
		svc, err := serviceusage.NewService(c.ctx, c.opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create Service Usage client: %w", err)
		}
		c.suService = svc
	}
	return c.suService, nil
}

func (c *Client) iamSvc() (*iam.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.iamService == nil {
		svc, err := iam.NewService(c.ctx, c.opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create IAM client: %w", err)
		}
		c.iamService = svc
	}
	return c.iamService, nil
}

func (c *Client) pubsub() (*pubsubapi.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.psService == nil {
		svc, err := pubsubapi.NewService(c.ctx, c.opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create Pub/Sub client: %w", err)
		}
		c.psService = svc
	}
	return c.psService, nil
}

func (c *Client) oauth2() (*oauth2api.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.oauthService == nil {
		svc, err := oauth2api.NewService(c.ctx, c.opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OAuth2 client: %w", err)
		}
		c.oauthService = svc
	}
	return c.oauthService, nil
}

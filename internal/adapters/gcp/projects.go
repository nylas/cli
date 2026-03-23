package gcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nylas/cli/internal/domain"
	crm "google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/googleapi"
	serviceusage "google.golang.org/api/serviceusage/v1"
)

// CheckAuth verifies ADC works and returns the authenticated email.
func (c *Client) CheckAuth(ctx context.Context) (string, error) {
	svc, err := c.oauth2()
	if err != nil {
		return "", fmt.Errorf("authentication check failed: %w", err)
	}
	info, err := svc.Userinfo.Get().Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("authentication check failed: %w", err)
	}
	return info.Email, nil
}

// ListProjects lists the user's accessible GCP projects.
func (c *Client) ListProjects(ctx context.Context) ([]domain.GCPProject, error) {
	svc, err := c.resourceManager()
	if err != nil {
		return nil, err
	}

	var projects []domain.GCPProject
	err = svc.Projects.Search().Context(ctx).Pages(ctx, func(resp *crm.SearchProjectsResponse) error {
		for _, p := range resp.Projects {
			if p.State == "ACTIVE" {
				projects = append(projects, domain.GCPProject{
					ProjectID:   p.ProjectId,
					DisplayName: p.DisplayName,
					State:       p.State,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	return projects, nil
}

// CreateProject creates a new GCP project.
func (c *Client) CreateProject(ctx context.Context, projectID, displayName string) error {
	svc, err := c.resourceManager()
	if err != nil {
		return err
	}

	op, err := svc.Projects.Create(&crm.Project{
		ProjectId:   projectID,
		DisplayName: displayName,
	}).Context(ctx).Do()

	if isConflict(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return c.pollCRMOperation(ctx, svc, op.Name)
}

// GetProject checks if a project exists.
func (c *Client) GetProject(ctx context.Context, projectID string) error {
	svc, err := c.resourceManager()
	if err != nil {
		return err
	}

	_, err = svc.Projects.Get("projects/" + projectID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}
	return nil
}

// BatchEnableAPIs enables multiple APIs for a project.
func (c *Client) BatchEnableAPIs(ctx context.Context, projectID string, apis []string) error {
	svc, err := c.serviceUsage()
	if err != nil {
		return err
	}

	op, err := svc.Services.BatchEnable("projects/"+projectID, &serviceusage.BatchEnableServicesRequest{
		ServiceIds: apis,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to enable APIs: %w", err)
	}

	return c.pollSUOperation(ctx, svc, op.Name)
}

func (c *Client) pollCRMOperation(ctx context.Context, svc *crm.Service, opName string) error {
	for {
		op, err := svc.Operations.Get(opName).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to poll operation: %w", err)
		}
		if op.Done {
			if op.Error != nil {
				return fmt.Errorf("operation failed: %s", op.Error.Message)
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func (c *Client) pollSUOperation(ctx context.Context, svc *serviceusage.Service, opName string) error {
	for {
		op, err := svc.Operations.Get(opName).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to poll operation: %w", err)
		}
		if op.Done {
			if op.Error != nil {
				return fmt.Errorf("operation failed: %s", op.Error.Message)
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func isConflict(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*googleapi.Error); ok {
		return apiErr.Code == http.StatusConflict
	}
	return false
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*googleapi.Error); ok {
		return apiErr.Code == http.StatusNotFound
	}
	return false
}

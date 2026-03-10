package gcp

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/ports"
	crm "google.golang.org/api/cloudresourcemanager/v3"
	iam "google.golang.org/api/iam/v1"
)

// GetIAMPolicy retrieves the IAM policy for a project.
func (c *Client) GetIAMPolicy(ctx context.Context, projectID string) (*ports.IAMPolicy, error) {
	svc, err := c.resourceManager()
	if err != nil {
		return nil, err
	}

	resp, err := svc.Projects.GetIamPolicy("projects/"+projectID, &crm.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM policy: %w", err)
	}

	policy := &ports.IAMPolicy{Etag: resp.Etag}
	for _, b := range resp.Bindings {
		policy.Bindings = append(policy.Bindings, &ports.IAMBinding{
			Role:    b.Role,
			Members: b.Members,
		})
	}
	return policy, nil
}

// SetIAMPolicy sets the IAM policy for a project.
func (c *Client) SetIAMPolicy(ctx context.Context, projectID string, policy *ports.IAMPolicy) error {
	svc, err := c.resourceManager()
	if err != nil {
		return err
	}

	var bindings []*crm.Binding
	for _, b := range policy.Bindings {
		bindings = append(bindings, &crm.Binding{
			Role:    b.Role,
			Members: b.Members,
		})
	}

	_, err = svc.Projects.SetIamPolicy("projects/"+projectID, &crm.SetIamPolicyRequest{
		Policy: &crm.Policy{
			Bindings: bindings,
			Etag:     policy.Etag,
		},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to set IAM policy: %w", err)
	}
	return nil
}

// CreateServiceAccount creates a service account in a project.
func (c *Client) CreateServiceAccount(ctx context.Context, projectID, accountID, displayName string) (string, error) {
	svc, err := c.iamSvc()
	if err != nil {
		return "", err
	}

	sa, err := svc.Projects.ServiceAccounts.Create("projects/"+projectID, &iam.CreateServiceAccountRequest{
		AccountId: accountID,
		ServiceAccount: &iam.ServiceAccount{
			DisplayName: displayName,
		},
	}).Context(ctx).Do()

	if isConflict(err) {
		email := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", accountID, projectID)
		return email, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to create service account: %w", err)
	}
	return sa.Email, nil
}

// ServiceAccountExists checks if a service account exists.
func (c *Client) ServiceAccountExists(ctx context.Context, projectID, email string) bool {
	svc, err := c.iamSvc()
	if err != nil {
		return false
	}

	_, err = svc.Projects.ServiceAccounts.Get("projects/" + projectID + "/serviceAccounts/" + email).Context(ctx).Do()
	return err == nil
}

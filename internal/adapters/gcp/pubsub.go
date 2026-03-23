package gcp

import (
	"context"
	"fmt"
	"slices"

	pubsubapi "google.golang.org/api/pubsub/v1"
)

// CreateTopic creates a Pub/Sub topic.
func (c *Client) CreateTopic(ctx context.Context, projectID, topicName string) error {
	svc, err := c.pubsub()
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("projects/%s/topics/%s", projectID, topicName)
	_, err = svc.Projects.Topics.Create(topic, &pubsubapi.Topic{}).Context(ctx).Do()
	if isConflict(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}
	return nil
}

// TopicExists checks if a Pub/Sub topic exists.
func (c *Client) TopicExists(ctx context.Context, projectID, topicName string) bool {
	svc, err := c.pubsub()
	if err != nil {
		return false
	}

	topic := fmt.Sprintf("projects/%s/topics/%s", projectID, topicName)
	_, err = svc.Projects.Topics.Get(topic).Context(ctx).Do()
	return !isNotFound(err) && err == nil
}

// SetTopicIAMPolicy sets IAM policy on a Pub/Sub topic.
func (c *Client) SetTopicIAMPolicy(ctx context.Context, projectID, topicName, member, role string) error {
	svc, err := c.pubsub()
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("projects/%s/topics/%s", projectID, topicName)

	// Get current policy
	policy, err := svc.Projects.Topics.GetIamPolicy(topic).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get topic IAM policy: %w", err)
	}

	// Check if binding already exists
	for _, b := range policy.Bindings {
		if b.Role == role {
			if slices.Contains(b.Members, member) {
				return nil // Already set
			}
			b.Members = append(b.Members, member)
			_, err = svc.Projects.Topics.SetIamPolicy(topic, &pubsubapi.SetIamPolicyRequest{
				Policy: policy,
			}).Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("failed to set topic IAM policy: %w", err)
			}
			return nil
		}
	}

	// Add new binding
	policy.Bindings = append(policy.Bindings, &pubsubapi.Binding{
		Role:    role,
		Members: []string{member},
	})

	_, err = svc.Projects.Topics.SetIamPolicy(topic, &pubsubapi.SetIamPolicyRequest{
		Policy: policy,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to set topic IAM policy: %w", err)
	}
	return nil
}

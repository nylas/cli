package webhook

import (
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

var pubSubColumns = []ports.Column{
	{Header: "ID", Field: "ID", Width: -1},
	{Header: "Description", Field: "Description", Width: 28},
	{Header: "Topic", Field: "Topic", Width: 40},
	{Header: "Status", Field: "Status", Width: 10},
}

func parseAndValidateTriggers(values []string) ([]string, error) {
	var triggers []string
	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			trigger := strings.TrimSpace(part)
			if trigger != "" {
				triggers = append(triggers, trigger)
			}
		}
	}

	if len(triggers) == 0 {
		return nil, common.NewUserError(
			"at least one trigger type is required",
			"Use --triggers and run 'nylas webhook triggers' to see available types",
		)
	}

	validTriggers := domain.AllTriggerTypes()
	for _, trigger := range triggers {
		if err := common.ValidateOneOf("trigger type", trigger, validTriggers); err != nil {
			return nil, common.NewUserError(
				fmt.Sprintf("invalid trigger type: %s", trigger),
				"Run 'nylas webhook triggers' to see available trigger types",
			)
		}
	}

	return triggers, nil
}

func validatePubSubTopic(topic string) error {
	if err := common.ValidateRequiredFlag("--topic", topic); err != nil {
		return err
	}
	if !strings.HasPrefix(topic, "projects/") || !strings.Contains(topic, "/topics/") {
		return common.NewUserError(
			"invalid Pub/Sub topic",
			"Use the full topic path: projects/<PROJECT_ID>/topics/<TOPIC_ID>",
		)
	}
	return nil
}

// printPubSubChannel prints a Pub/Sub channel. When showSecrets is true the
// encryption key is printed in full; otherwise it is masked. Callers should
// pass true only at creation time, when the user needs to copy the key once.
func printPubSubChannel(channel *domain.PubSubChannel, showSecrets bool) {
	fmt.Printf("ID:           %s\n", channel.ID)
	fmt.Printf("Description:  %s\n", channel.Description)
	fmt.Printf("Topic:        %s\n", channel.Topic)
	if channel.Status != "" {
		fmt.Printf("Status:       %s\n", channel.Status)
	}
	if channel.EncryptionKey != "" {
		if showSecrets {
			fmt.Printf("Encryption Key: %s\n", channel.EncryptionKey)
		} else {
			fmt.Printf("Encryption Key: %s\n", maskSecret(channel.EncryptionKey))
		}
	}
	if len(channel.TriggerTypes) > 0 {
		fmt.Println("\nTrigger Types:")
		for _, trigger := range channel.TriggerTypes {
			fmt.Printf("  • %s\n", trigger)
		}
	}
	if len(channel.NotificationEmailAddresses) > 0 {
		fmt.Println("\nNotification Emails:")
		for _, email := range channel.NotificationEmailAddresses {
			fmt.Printf("  • %s\n", email)
		}
	}
	if !channel.CreatedAt.IsZero() || !channel.UpdatedAt.IsZero() {
		fmt.Println("\nTimestamps:")
		if !channel.CreatedAt.IsZero() {
			fmt.Printf("  Created:  %s\n", channel.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		if !channel.UpdatedAt.IsZero() {
			fmt.Printf("  Updated:  %s\n", channel.UpdatedAt.Format("2006-01-02 15:04:05"))
		}
	}
}

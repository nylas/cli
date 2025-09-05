package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nylas/cli/pkg/util"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func getToken(t string) (string, error) {
	key, err := keyring.Get("nylas-cli-"+t, "default-user")
	return key, err
}

func handleError(Body io.ReadCloser) {
	err := Body.Close()
	if err != nil {
		fmt.Printf("An error occurred while waiting for a response: %v\n", err)
		os.Exit(1)
	}
}

var tunnelUrl string
var appID string
var webhookBase = &cobra.Command{
	Use:   "webhook",
	Short: "Manages various parts of a Nylas webhook",
	Long:  "Manages various parts of a Nylas webhook. Currently only supports tunneling a webhook connection.",
}
var webhook = &cobra.Command{
	Use:   "tunnel",
	Short: "Creates a connection to Nylas's webhook server and receives events locally",
	Long:  "Creates a streaming connection to Nylas's webhook server to receive events from locally. Capable of forwarding these events to a specified URL if the --forward flag is defined.",
	Run: func(cmd *cobra.Command, args []string) {
		c := &http.Client{
			Timeout: time.Hour,
		}
		// Nylas webhook challenge
		if tunnelUrl != "" {
			challenge := uuid.New().String()
			req, err := http.NewRequest("GET", strings.TrimSuffix(tunnelUrl, "/")+"?challenge="+challenge, nil)
			if err != nil {
				fmt.Printf("Error occurred while sending challenge request: %v\n", err)
				os.Exit(1)
			}
			resp, err := c.Do(req)
			if err != nil {
				fmt.Printf("Error occurred while sending challenge request: %v\n", err)
				os.Exit(1)
			}
			defer handleError(resp.Body)

			if resp.StatusCode != 200 {
				fmt.Printf("Tunnel URL returned non-200 status: %d", resp.StatusCode)
				os.Exit(1)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading challenge response: %v\n", err)
				os.Exit(1)
			}
			if string(body) != challenge {
				fmt.Printf("Returned value (%s) does not match challenge string (%s)", string(body), challenge)
				os.Exit(1)
			}
		}

		key, err := getToken("api-key")
		if err != nil {
			fmt.Printf("Error occurred while retrieving app ID: %v", err)
			os.Exit(1)
		}

		debugf("Using region: %s", region)
		req, err := http.NewRequest("GET", util.RegionConfig[region].StreamEndpointURL, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+key)

		// Only support Application ID header in dev region
		if region == "dev" && appID != "" {
			req.Header.Set("X-Nylas-Application-Id", appID)
		}

		resp, err := c.Do(req)
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			os.Exit(1)
		}
		defer handleError(resp.Body)

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Server returned non-200 status: %d\n", resp.StatusCode)
			os.Exit(1)
		}

		reader := bufio.NewReader(resp.Body)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Println("Server closed the connection.")
					break
				}
				fmt.Printf("Error reading from stream: %v\n", err)
				os.Exit(1)
			}

			// Remove leading/trailing whitespace
			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "data: ") { // Webhook events
				data := strings.TrimPrefix(line, "data: ")
				if tunnelUrl != "" {
					// Forward the data to the tunnel URL
					req, err = http.NewRequestWithContext(ctx, http.MethodPost, tunnelUrl, strings.NewReader(data))
					if err != nil {
						fmt.Printf("Error forwarding webhook event: %v\n", err)
						os.Exit(1)
					}
					req.Header.Set("Content-Type", "application/json")

					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						fmt.Printf("Error forwarding webhook event: %v\n", err)
						os.Exit(1)
					}
					defer handleError(resp.Body)

					if resp.StatusCode != http.StatusOK {
						fmt.Printf("Tunnel returned non-200 status: %d\n", resp.StatusCode)
					}
				} else {
					fmt.Printf("Received message: %s\n", data)
				}
			} else if strings.HasPrefix(line, ":") { // Comments
				if line != ":heartbeat" { // Ignore heartbeats
					fmt.Println("Received comment: " + line[1:])
				}
			}
		}
	},
}

func init() {
	webhook.Flags().StringVar(&tunnelUrl, "forward", "", "The locally hosted URL (http://localhost:PORT) to forward webhook messages to")
	webhook.Flags().StringVar(&appID, "app-id", "", "Nylas Application ID to send as X-Nylas-Application-Id header (only used with --region=dev)")
	webhookBase.AddCommand(webhook)
	root.AddCommand(webhookBase)
}

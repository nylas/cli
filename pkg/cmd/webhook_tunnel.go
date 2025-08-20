package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"nylas-cli/pkg/util"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func getToken(t string) (string, error) {
	key, err := keyring.Get("nylas-cli-"+t, "default-user")
	return key, err
}

var tunnelUrl string
var webhook = &cobra.Command{
	Use:   "webhook tunnel",
	Short: "Creates a connection to Nylas's webhook server and receives events locally",
	Long:  "Creates a streaming connection to Nylas's webhook server to receive events from locally. Capable of forwarding these events to a specified URL if the --forward flag is defined.",
	Run: func(cmd *cobra.Command, args []string) {
		c := &http.Client{
			Timeout: 3.6e+12, // 1 hour timeout
		}
		// Nylas webhook challenge
		if tunnelUrl != "" {
			challenge := uuid.New().String()
			req, err := http.NewRequest("GET", strings.TrimSuffix(tunnelUrl, "/")+"?challenge="+challenge, nil)
			if err != nil {
				log.Fatal("Error occurred while sending challenge request: ", err)
			}
			resp, err := c.Do(req)
			if err != nil {
				log.Fatal("Error occurred while sending challenge request: ", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				log.Fatalf("Tunnel URL returned non-200 status: %d", resp.StatusCode)
			}
			body, err := io.ReadAll(resp.Body)
			if string(body) != challenge {
				log.Fatalf("Returned value (%s) does not match challenge string (%s)", string(body), challenge)
			}
		}

		key, err := getToken("api-key")
		if err != nil {
			log.Fatalf("Error occurred while retrieving app ID: %v", err)
		}

		req, err := http.NewRequest("GET", util.RegionConfig["us"].StreamEndpointURL, nil)
		if err != nil {
			log.Fatalf("Error creating request: %v", err)
		}
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+key)

		resp, err := c.Do(req)
		if err != nil {
			log.Fatalf("Error making request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Server returned non-200 status: %d", resp.StatusCode)
		}

		reader := bufio.NewReader(resp.Body)

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Println("Server closed the connection.")
					break
				}
				log.Fatalf("Error reading from stream: %v", err)
			}

			// Remove leading/trailing whitespace
			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "data: ") { // Webhook events
				data := strings.TrimPrefix(line, "data: ")
				if tunnelUrl != "" {
					// Forward the data to the tunnel URL
					http.Post(tunnelUrl, "application/json", strings.NewReader(data))
				} else {
					fmt.Printf("Received message: %s\n", data)
				}
			} else if strings.HasPrefix(line, ":") { // Comments
				fmt.Println("Received comment: " + line[1:])
			}
		}
	},
}

func init() {
	webhook.Flags().StringVar(&tunnelUrl, "forward", "", "The locally hosted URL (http://localhost:PORT) to forward webhook messages to")
	root.AddCommand(webhook)
}

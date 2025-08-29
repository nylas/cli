package cmd

import (
	"fmt"
	"os"

	"github.com/nylas/cli/pkg/client"
	"github.com/nylas/cli/pkg/util"

	"syscall"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

func saveToken(t string, token string) error {
	return keyring.Set("nylas-cli-"+t, "default-user", token)
}

var initialize = &cobra.Command{
	Use:   "init",
	Short: "Authenticate the CLI using a Nylas API key",
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := ""

		if len(args) != 1 {
			fmt.Println("Visit the dashboard at https://dashboard-v3.nylas.com and create a new API key if you do not have one.")

			fmt.Print("Enter your API key: ")
			bytes, err := term.ReadPassword(syscall.Stdin)
			if err != nil {
				fmt.Printf("An error occured while reading API key: %v\n", err)
				os.Exit(1)
			}
			apiKey = string(bytes)
		} else {
			apiKey = args[0]
		}

		debugf("Using region: %s", region)
		nylasAPI := client.CreateNylasAPIClient(util.RegionConfig[region].NylasAPIURL)
		_, err := nylasAPI.GetApplication(apiKey)

		if err != nil {
			fmt.Println("Could not initialize the app with credentials provided. Please check your API key and try again.")
			fmt.Println(err)
			os.Exit(1)
		} else {
			if err := saveToken("api-key", apiKey); err != nil {
				fmt.Printf("An error occurred while trying to save the API key: %v\n", err)
				os.Exit(1)
				return
			}

			fmt.Println("Successfully saved your application credentials.")
		}
	},
}

func init() {
	root.AddCommand(initialize)
}

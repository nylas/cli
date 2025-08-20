package cmd

import (
	"fmt"
	"log"
	"nylas-cli/pkg/client"
	"nylas-cli/pkg/util"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
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
			fmt.Scanln(&apiKey)
		} else {
			apiKey = args[0]
		}

		nylasApi := client.CreateNylasAPIClient(util.RegionConfig["us"].NylasAPIURL)
		_, appErr := nylasApi.GetApplication(apiKey)

		if appErr != nil {
			fmt.Println("<red>Could not initialize the app with credentials provided. Please check your API key and try again.<red>")
			fmt.Println(appErr)
		} else {
			if err := saveToken("api-key", apiKey); err != nil {
				log.Fatal(err)
				return
			}

			fmt.Println("<green>Successfully saved your application credentials.<green>")
		}
	},
}

func init() {
	root.AddCommand(initialize)
}

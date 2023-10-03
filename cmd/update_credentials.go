package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/kontik-pk/goph-keeper/internal"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

// updateCredentialsCmd represents the updateCredentials command
var updateCredentialsCmd = &cobra.Command{
	Use:     "update-credentials",
	Short:   "Update user credentials for provided login.",
	Example: "goph-keeper update-credentials --user <user-name> --login <saved-login> --password <new-password>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatalf("Some error occured. Err: %s", err)
		}
		var cfg internal.Params
		if err := envconfig.Process("", &cfg); err != nil {
			log.Fatalf("error while loading envs: %s\n", err)
		}

		userName, _ := cmd.Flags().GetString("user")
		login, _ := cmd.Flags().GetString("login")
		password, _ := cmd.Flags().GetString("password")
		metadata, _ := cmd.Flags().GetString("metadata")
		requestCredentials := internal.Credentials{
			UserName: userName,
			Login:    &login,
			Password: &password,
			Metadata: &metadata,
		}
		body, err := json.Marshal(requestCredentials)
		if err != nil {
			log.Fatalf(err.Error())
		}
		resp, err := resty.New().R().
			SetHeader("Content-type", "application/json").
			SetBody(body).
			Post(fmt.Sprintf("http://%s:%s/update/credentials", cfg.ApplicationHost, cfg.ApplicationPort))
		if err != nil {
			log.Printf(err.Error())
		}
		if resp.StatusCode() != http.StatusOK {
			log.Printf("status code is not OK: %s\n", resp.Status())
		}
		log.Println(resp.String())
	},
}

func init() {
	rootCmd.AddCommand(updateCredentialsCmd)
	updateCredentialsCmd.Flags().String("user", "", "user name")
	updateCredentialsCmd.Flags().String("login", "", "user login")
	updateCredentialsCmd.Flags().String("password", "", "user password")
	updateCredentialsCmd.Flags().String("metadata", "", "metadata")
	updateCredentialsCmd.MarkFlagRequired("user")
	updateCredentialsCmd.MarkFlagRequired("login")
	updateCredentialsCmd.MarkFlagRequired("password")
}

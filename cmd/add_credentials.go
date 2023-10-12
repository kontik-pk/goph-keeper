package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/kontik-pk/goph-keeper/internal"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

// addCredentialsCmd represents the add-credentials command
var addCredentialsCmd = &cobra.Command{
	Use:   "add-credentials",
	Short: "Add a pair of login/password to goph-keeper.",
	Long: `Add a pair of login/password to goph-keeper database for
long-term storage. Only authorized users can use this command. The password is stored in the database in encrypted form.`,
	Example: "goph-keeper add-credentials --user <user-name> --login <user-login> --password <password to store> --metadata <some description>",

	Run: func(cmd *cobra.Command, args []string) {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatalf("error while getting envs: %s", err)
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
		}
		if metadata != "" {
			requestCredentials.Metadata = &metadata
		}
		body, err := json.Marshal(requestCredentials)
		if err != nil {
			log.Fatalf(err.Error())
		}
		resp, err := resty.New().R().
			SetHeader("Content-type", "application/json").
			SetBody(body).
			Post(fmt.Sprintf("http://%s:%s/save/credentials", cfg.ApplicationHost, cfg.ApplicationPort))
		if err != nil {
			log.Printf(err.Error())
		}
		if resp.StatusCode() != http.StatusOK {
			log.Printf("status code is not OK: %s\n", resp.Status())
		}
		fmt.Println(resp.String())
	},
}

func init() {
	rootCmd.AddCommand(addCredentialsCmd)
	addCredentialsCmd.Flags().String("user", "", "user name")
	addCredentialsCmd.Flags().String("login", "", "user login")
	addCredentialsCmd.Flags().String("password", "", "user password")
	addCredentialsCmd.Flags().String("metadata", "", "metadata")
	addCredentialsCmd.MarkFlagRequired("user")
	addCredentialsCmd.MarkFlagRequired("login")
	addCredentialsCmd.MarkFlagRequired("password")
}

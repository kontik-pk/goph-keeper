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

// getCredentialsCmd represents the get-credentials command
var getCredentialsCmd = &cobra.Command{
	Use:   "get-credentials",
	Short: "Get a pair of login/password for specified user",
	Long: `Get a pair of login/password for specified user from goph-keeper storage. 
Only authorized users can use this command`,
	Example: "goph-keeper get-credentials --user user_name",
	Run: func(cmd *cobra.Command, args []string) {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatalf("error while getting envs: %s", err)
		}

		var cfg internal.Params
		if err := envconfig.Process("", &cfg); err != nil {
			log.Fatalf("error while loading envs: %s\n", err)
		}

		userName, _ := cmd.Flags().GetString("user")
		userLogin, _ := cmd.Flags().GetString("login")
		requestUserCredentials := internal.Credentials{
			UserName: userName,
		}
		if userLogin != "" {
			requestUserCredentials.Login = &userLogin
		}
		body, err := json.Marshal(requestUserCredentials)
		if err != nil {
			log.Fatalln(err.Error())
		}

		resp, err := resty.New().R().
			SetHeader("Content-type", "application/json").
			SetBody(body).
			Post(fmt.Sprintf("http://%s:%s/get/credentials", cfg.ApplicationHost, cfg.ApplicationPort))
		if err != nil {
			log.Printf(err.Error())
		}
		if resp.StatusCode() != http.StatusOK {
			log.Printf("status code is not OK: %s\n", resp.Status())
		}
		log.Printf(resp.String())
	},
}

func init() {
	rootCmd.AddCommand(getCredentialsCmd)
	getCredentialsCmd.Flags().String("user", "", "user name")
	getCredentialsCmd.Flags().String("login", "", "user login")
	getCredentialsCmd.MarkFlagRequired("user")
}

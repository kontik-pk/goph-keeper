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
	"os"

	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the goph-keeper system",
	Long: `Login to the goph-keeper system with specified login and password. 
Only registered users can run this command`,
	Example: "goph-keeper login --login <user-system-login> --password <user-system-password>`",
	Run: func(cmd *cobra.Command, args []string) {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatalf("Some error occured. Err: %s", err)
		}

		var cfg internal.Params
		if err := envconfig.Process("", &cfg); err != nil {
			log.Printf("error while loading envs: %s\n", err)
			os.Exit(1)
		}

		login, _ := cmd.Flags().GetString("login")
		password, _ := cmd.Flags().GetString("password")
		userCreds := internal.User{
			Login:    login,
			Password: password,
		}

		body, err := json.Marshal(userCreds)
		if err != nil {
			log.Printf(err.Error())
			os.Exit(1)
		}

		resp, err := resty.New().R().
			SetHeader("Content-type", "application/json").
			SetBody(body).
			Post(fmt.Sprintf("http://%s:%s/auth/login", cfg.ApplicationHost, cfg.ApplicationPort))
		if err != nil {
			log.Printf(err.Error())
		}
		if resp.StatusCode() != http.StatusOK {
			log.Printf("status code is not OK: %s\n", resp.Status())
			fmt.Println(resp.String())
			return
		}
		fmt.Printf("user %q was successfully logined in goph-keeper", login)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().String("login", "", "user login")
	loginCmd.Flags().String("password", "", "user password")
	loginCmd.MarkFlagRequired("login")
	loginCmd.MarkFlagRequired("password")
}

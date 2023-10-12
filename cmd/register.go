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

// registerCmd represents the register command
var registerCmd = &cobra.Command{
	Use:     "register",
	Short:   "Register in the goph-keeper system.",
	Long:    `Register in the goph-keeper system with provided login and password`,
	Example: "goph-keeper register --login <user-system-login> --password <user-system-password>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatalf("error while getting envs: %s", err)
		}

		var cfg internal.Params
		if err := envconfig.Process("", &cfg); err != nil {
			log.Fatalf("error while loading envs: %s\n", err)
		}

		login, _ := cmd.Flags().GetString("login")
		password, _ := cmd.Flags().GetString("password")
		userCreds := internal.User{
			Login:    login,
			Password: password,
		}

		body, err := json.Marshal(userCreds)
		if err != nil {
			log.Fatalf(err.Error())
		}

		resp, err := resty.New().R().
			SetHeader("Content-type", "application/json").
			SetBody(body).
			Post(fmt.Sprintf("http://%s:%s/auth/register", cfg.ApplicationHost, cfg.ApplicationPort))
		if err != nil {
			log.Printf(err.Error())
		}
		if resp.StatusCode() != http.StatusOK {
			log.Printf("status code is not OK: %s\n", resp.Status())
			fmt.Println(resp.String())
			return
		}
		fmt.Printf("user %q was successfully registered in goph-keeper", login)
	},
}

func init() {
	rootCmd.AddCommand(registerCmd)
	registerCmd.Flags().String("login", "", "user login")
	registerCmd.Flags().String("password", "", "user password")
}

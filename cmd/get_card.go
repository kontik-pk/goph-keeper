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

// getCardCmd represents the getCard command
var getCardCmd = &cobra.Command{
	Use:     "get-card",
	Short:   "Get card info from goph-keeper storage",
	Example: "goph-keeper  get-card --user <user-name> --number <card number>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatalf("Some error occured. Err: %s", err)
		}

		var cfg internal.Params
		if err := envconfig.Process("", &cfg); err != nil {
			log.Printf("error while loading envs: %s\n", err)
			os.Exit(1)
		}

		userName, _ := cmd.Flags().GetString("user")
		bank, _ := cmd.Flags().GetString("bank")
		number, _ := cmd.Flags().GetString("number")
		requestCard := internal.Card{
			UserName: userName,
		}
		if bank != "" {
			requestCard.BankName = &bank
		}
		if number != "" {
			requestCard.Number = &number
		}
		body, err := json.Marshal(requestCard)
		if err != nil {
			log.Fatalln(err.Error())
		}

		resp, err := resty.New().R().
			SetHeader("Content-type", "application/json").
			SetBody(body).
			Post(fmt.Sprintf("http://%s:%s/get/card", cfg.ApplicationHost, cfg.ApplicationPort))
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
	rootCmd.AddCommand(getCardCmd)
	getCardCmd.Flags().String("user", "", "user name")
	getCardCmd.Flags().String("bank", "", "bank")
	getCardCmd.Flags().String("number", "", "number")
	getCardCmd.MarkFlagRequired("user")
}

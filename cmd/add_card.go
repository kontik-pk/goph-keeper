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

// addCardCmd represents the addCard command
var addCardCmd = &cobra.Command{
	Use:   "add-card",
	Short: "Add bank card info to goph-keeper.",
	Long: `Add bank card info (bank name, card number, cv, password and metadata) to goph-keeper database for
long-term storage. Only authorized users can use this command. Password and cv are stored in the database in the encrypted form.`,
	Example: "goph-keeper  add-card --user user-name --bank alpha --number 1111222233334444 --cv 123 --password 1243",
	Run: func(cmd *cobra.Command, args []string) {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatalf("error occured while loading envs from file: %s", err)
		}
		var cfg internal.Params
		if err := envconfig.Process("", &cfg); err != nil {
			log.Fatalf("error while processing envs: %s\n", err)
		}

		userName, _ := cmd.Flags().GetString("user")
		bank, _ := cmd.Flags().GetString("bank")
		number, _ := cmd.Flags().GetString("number")
		cv, _ := cmd.Flags().GetString("cv")
		password, _ := cmd.Flags().GetString("password")
		metadata, _ := cmd.Flags().GetString("metadata")

		if bank == "" || userName == "" || number == "" || cv == "" || password == "" {
			log.Fatalln("user name, bank name, card number, cv and password should not be empty")
		}
		if len(number) != 16 {
			log.Fatalln("the identification number of the plastic card must consist of 16 digits.")
		}
		if len(cv) != 3 {
			log.Fatalln("the cv code of the plastic card must consist of 3 digits.")
		}
		requestCard := internal.Card{
			UserName: userName,
			BankName: &bank,
			Number:   &number,
			CV:       &cv,
			Password: &password,
		}
		if metadata != "" {
			requestCard.Metadata = &metadata
		}
		body, err := json.Marshal(requestCard)
		if err != nil {
			log.Fatalf(err.Error())
		}
		resp, err := resty.New().R().
			SetHeader("Content-type", "application/json").
			SetBody(body).
			Post(fmt.Sprintf("http://%s:%s/save/card", cfg.ApplicationHost, cfg.ApplicationPort))
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
	rootCmd.AddCommand(addCardCmd)
	addCardCmd.Flags().String("user", "", "user name")
	addCardCmd.Flags().String("bank", "", "bank")
	addCardCmd.Flags().String("number", "", "card number")
	addCardCmd.Flags().String("cv", "", "card cv")
	addCardCmd.Flags().String("password", "", "card password")
	addCardCmd.Flags().String("metadata", "", "metadata")
	addCardCmd.MarkFlagRequired("user")
	addCardCmd.MarkFlagRequired("bank")
	addCardCmd.MarkFlagRequired("number")
	addCardCmd.MarkFlagRequired("cv")
	addCardCmd.MarkFlagRequired("password")
}

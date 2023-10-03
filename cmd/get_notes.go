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

// getNotesCmd represents the getNotes command
var getNotesCmd = &cobra.Command{
	Use:     "get-note",
	Short:   "Get user's notes from goph-keeper",
	Example: "goph-keeper get-note --user <user-name>",
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
		title, _ := cmd.Flags().GetString("title")
		requestNotes := internal.Note{
			UserName: userName,
		}
		if title != "" {
			requestNotes.Title = &title
		}
		body, err := json.Marshal(requestNotes)
		if err != nil {
			log.Fatalln(err.Error())
		}

		resp, err := resty.New().R().
			SetHeader("Content-type", "application/json").
			SetBody(body).
			Post(fmt.Sprintf("http://%s:%s/get/note", cfg.ApplicationHost, cfg.ApplicationPort))
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
	rootCmd.AddCommand(getNotesCmd)
	getNotesCmd.Flags().String("user", "", "user name")
	getNotesCmd.Flags().String("title", "", "title of the note")
	getNotesCmd.MarkFlagRequired("user")
}

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

// deleteNotesCmd represents the deleteNotes command
var deleteNotesCmd = &cobra.Command{
	Use:     "delete-note",
	Short:   "Delete user's notes from goph-keeper storage",
	Example: "goph-keeper delete-note --user <user-name> --title <note title>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatalf("error while getting envs: %s", err)
		}
		var cfg internal.Params
		if err := envconfig.Process("", &cfg); err != nil {
			log.Fatalf("error while loading envs: %s\n", err)
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
			Post(fmt.Sprintf("http://%s:%s/delete/note", cfg.ApplicationHost, cfg.ApplicationPort))
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
	rootCmd.AddCommand(deleteNotesCmd)
	deleteNotesCmd.Flags().String("user", "", "user name")
	deleteNotesCmd.Flags().String("title", "", "title of the note")
	deleteNotesCmd.MarkFlagRequired("user")
}

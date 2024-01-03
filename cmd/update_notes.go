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

// updateNotesCmd represents the updateNotes command
var updateNotesCmd = &cobra.Command{
	Use:     "update-note",
	Short:   "Update user notes.",
	Example: "goph-keeper update-notes --user <user-name> --title <note-title> --content <new-content>",
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
		content, _ := cmd.Flags().GetString("content")
		metadata, _ := cmd.Flags().GetString("metadata")
		requestNote := internal.Note{
			UserName: userName,
			Title:    &title,
			Content:  &content,
			Metadata: &metadata,
		}
		body, err := json.Marshal(requestNote)
		if err != nil {
			log.Fatalf(err.Error())
		}
		resp, err := resty.New().R().
			SetHeader("Content-type", "application/json").
			SetBody(body).
			Post(fmt.Sprintf("http://%s:%s/update/note", cfg.ApplicationHost, cfg.ApplicationPort))
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
	rootCmd.AddCommand(updateNotesCmd)
	updateNotesCmd.Flags().String("user", "", "user name")
	updateNotesCmd.Flags().String("title", "", "title of the note")
	updateNotesCmd.Flags().String("content", "", "new note's content")
	updateNotesCmd.Flags().String("metadata", "", "metadata")
	updateNotesCmd.MarkFlagRequired("user")
	updateNotesCmd.MarkFlagRequired("title")
	updateNotesCmd.MarkFlagRequired("content")
}

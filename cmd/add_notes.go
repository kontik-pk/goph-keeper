/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
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

// addNotesCmd represents the add-notes command
var addNotesCmd = &cobra.Command{
	Use:   "add-note",
	Short: "Add user's note to goph-keeper storage.",
	Long: `Add user's note to goph-keeper database for long-term storage.
Only authorized users can use this command. The note content is stored in the database in encrypted form.`,
	Example: "goph-keeper add-note --user <user-name> --title <note title> --content <note content> --metadata <note metadata>",
	Run: func(cmd *cobra.Command, args []string) {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatalf("Some error occured. Err: %s", err)
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
		}
		if metadata != "" {
			requestNote.Metadata = &metadata
		}
		body, err := json.Marshal(requestNote)
		if err != nil {
			log.Fatalf(err.Error())
		}
		resp, err := resty.New().R().
			SetHeader("Content-type", "application/json").
			SetBody(body).
			Post(fmt.Sprintf("http://%s:%s/save/note", cfg.ApplicationHost, cfg.ApplicationPort))
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
	rootCmd.AddCommand(addNotesCmd)
	addNotesCmd.Flags().String("user", "", "user name")
	addNotesCmd.Flags().String("title", "", "user login")
	addNotesCmd.Flags().String("content", "", "user password")
	addNotesCmd.Flags().String("metadata", "", "metadata")
	addNotesCmd.MarkFlagRequired("user")
	addNotesCmd.MarkFlagRequired("title")
	addNotesCmd.MarkFlagRequired("content")
}

// Package main provides the CLI entry point for the application.
// It uses Cobra to provide subcommands for running the server and generating
// OpenAPI specifications.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "app",
	Short: "Application CLI",
	Long:  "CLI for running the application server and generating OpenAPI specifications.",
}

func init() {
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(openapiCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

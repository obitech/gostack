package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/example/app/internal/api"
)

var outputFile string

var openapiCmd = &cobra.Command{
	Use:   "generate-openapi",
	Short: "Generate the OpenAPI specification",
	Long:  "Generates the OpenAPI specification from the API routes and outputs it as YAML.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateOpenAPI()
	},
}

func init() {
	openapiCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file path (default: stdout)")
}

func generateOpenAPI() error {
	yaml, err := api.GenerateOpenAPIYAML()
	if err != nil {
		return fmt.Errorf("generating OpenAPI spec: %w", err)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, yaml, 0o644); err != nil {
			return fmt.Errorf("writing to file %s: %w", outputFile, err)
		}
		return nil
	}

	fmt.Print(string(yaml))
	return nil
}

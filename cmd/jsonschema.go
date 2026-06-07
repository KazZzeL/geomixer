package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"

	"github.com/KazZzeL/geomixer/internal/config"
	"github.com/KazZzeL/geomixer/internal/utils/filer"
)

var (
	schemaOutputDir string
	schemaFileName  string
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Generate JSON Schema for configuration file",
	RunE:  runSchema,
}

func init() {
	schemaCmd.Flags().
		StringVarP(&schemaOutputDir, "output", "o", "./jsonschema", "Generated schema dir (default: ./jsonschema)")
	schemaCmd.Flags().
		StringVarP(&schemaFileName, "filename", "f", "geomixer.schema.json", "Generated schema filename (default: geomixer.schema.json)")
}

func runSchema(_ *cobra.Command, _ []string) error {
	schema := jsonschema.Reflect(&config.Config{})
	schema.ID = jsonschema.ID(
		"https://github.com/KazZzeL/geomixer/jsonschema/geomixer.schema.json",
	)

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	if err := filer.WriteFile(schemaOutputDir, schemaFileName, data); err != nil {
		return fmt.Errorf("write schema: %w", err)
	}

	return nil
}

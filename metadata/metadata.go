// Package metadata provides build-time information for CLI applications.
//
// Services set their build variables via ldflags at build time:
//
//	go build -ldflags "\
//	  -X github.com/dioad/cli/metadata.Version=$(VERSION) \
//	  -X github.com/dioad/cli/metadata.Commit=$(COMMIT) \
//	  -X github.com/dioad/cli/metadata.Date=$(DATE)" \
//	  ./...
//
// Then pass a BuildInfo to NewVersionCommand:
//
//	cmd.AddCommand(
//	    metadata.NewVersionCommand(OrgName, AppName, metadata.BuildInfo{
//	        Version: metadata.Version,
//	        Commit:  metadata.Commit,
//	        Date:    metadata.Date,
//	    }),
//	)
package metadata

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the application version, set via ldflags at build time.
var Version = "local"

// Commit is the git commit SHA, set via ldflags at build time.
var Commit = "XX"

// Date is the build date, set via ldflags at build time.
var Date = "1970-01-01"

// BuildInfo holds version information populated at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// NewVersionCommand returns a "version" cobra.Command that prints build information.
// orgName and appName are used in the plain-text output line.
//
// Unlike commands built with cli.NewCommand, this command wires RunE directly and
// does not call InitConfig, so it produces no side effects such as reading config
// files, configuring logging, or starting a background WatchConfig goroutine.
func NewVersionCommand(orgName, appName string, info BuildInfo) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information (use --json for detailed output)",
		Long: `Show version information.

By default, this prints a single human-readable line with the application version.
Use --json to print detailed version information (including version, commit, and date)
as a JSON object.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if jsonOutput {
				details := map[string]string{
					"version": info.Version,
					"commit":  info.Commit,
					"date":    info.Date,
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(details)
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s\n", orgName, appName, info.Version)
			return err
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output version information as JSON")

	return cmd
}

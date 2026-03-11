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
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dioad/cli"
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

type versionConfig struct {
	JSON bool `mapstructure:"json"`
}

// NewVersionCommand returns a "version" cobra.Command that prints build information.
// orgName and appName are used in the plain-text output line.
func NewVersionCommand(orgName, appName string, info BuildInfo) *cobra.Command {
	cfg := &versionConfig{}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Shows version information",
		Long:  `Shows detailed version information.`,
		Args:  cobra.NoArgs,
	}

	exec := func(_ context.Context, cfg *versionConfig) error {
		if cfg.JSON {
			details := map[string]string{
				"version": info.Version,
				"commit":  info.Commit,
				"date":    info.Date,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(details)
		}

		fmt.Printf("%s %s %v\n", orgName, appName, info.Version)
		return nil
	}

	return cli.NewCommand(cmd, exec, cfg)
}

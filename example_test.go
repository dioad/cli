package cli_test

import (
	"context"
	"fmt"

	"github.com/dioad/cli"
	"github.com/spf13/cobra"
)

// ExampleNewCommand demonstrates creating a type-safe command with configuration.
func ExampleNewCommand() {
	type Config struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	}

	cfg := &Config{
		Name: "example-service",
		Port: 8080,
	}

	cmd := cli.NewCommand(
		&cobra.Command{
			Use:   "serve",
			Short: "Start the service",
		},
		func(ctx context.Context, c *Config) error {
			fmt.Printf("Service %s listening on port %d\n", c.Name, c.Port)
			return nil
		},
		cfg,
		cli.WithConfigFlag("config.yaml"),
	)

	if cmd != nil {
		fmt.Println("Command created successfully")
	}
	// Output: Command created successfully
}

// ExampleWithConfigFlag demonstrates adding a config file flag to a command.
func ExampleWithConfigFlag() {
	cmd := &cobra.Command{Use: "serve"}
	opt := cli.WithConfigFlag("~/.app/config.yaml")
	opt(cmd)

	configFlag := cmd.Flag("config")
	if configFlag != nil {
		fmt.Printf("Config flag default: %s\n", configFlag.DefValue)
	}
	// Output: Config flag default: ~/.app/config.yaml
}

// ExampleDefaultConfigPath demonstrates getting the default config path.
func ExampleDefaultConfigPath() {
	path, err := cli.DefaultConfigPath("myorg", "myapp")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if path != "" {
		fmt.Println("Config path obtained successfully")
	}
	// Output: Config path obtained successfully
}

// ExampleDefaultPersistencePath demonstrates getting the persistence path.
func ExampleDefaultPersistencePath() {
	path, err := cli.DefaultPersistencePath("myorg", "myapp")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if path != "" {
		fmt.Println("Persistence path obtained successfully")
	}
	// Output: Persistence path obtained successfully
}

// ExampleDefaultConfigFile demonstrates getting a default config file path.
func ExampleDefaultConfigFile() {
	file, err := cli.DefaultConfigFile("myorg", "myapp", "config")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if file != "" {
		fmt.Println("Config file path obtained successfully")
	}
	// Output: Config file path obtained successfully
}

// ExampleContext demonstrates creating a context with application metadata.
func ExampleContext() {
	ctx := cli.Context(
		context.Background(),
		cli.SetOrgName("myorg"),
		cli.SetAppName("myapp"),
	)

	if ctx != nil {
		fmt.Println("Context created with metadata")
	}
	// Output: Context created with metadata
}

// ExampleSetOrgName demonstrates setting organization name in context.
func ExampleSetOrgName() {
	opt := cli.SetOrgName("acme-corp")
	ctx := opt(context.Background())

	if ctx != nil {
		fmt.Println("Organization name set in context")
	}
	// Output: Organization name set in context
}

// ExampleSetAppName demonstrates setting app name in context.
func ExampleSetAppName() {
	opt := cli.SetAppName("myservice")
	ctx := opt(context.Background())

	if ctx != nil {
		fmt.Println("App name set in context")
	}
	// Output: App name set in context
}

// ExampleIsDocker demonstrates Docker detection.
func ExampleIsDocker() {
	inDocker := cli.IsDocker()
	if inDocker {
		fmt.Println("Running in Docker")
	} else {
		fmt.Println("Running on host machine")
	}
	// Output: Running on host machine
}

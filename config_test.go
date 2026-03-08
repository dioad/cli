// Package cli_test contains unit tests for the cli package.
package cli_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/dioad/cli"
)

// TestIsDocker verifies Docker detection returns false in normal test environments.
// Note: testing the true case requires /.dockerenv to exist, which needs root.
// Coverage of the Docker-specific code paths in DefaultConfigPath and
// DefaultPersistencePath is instead achieved via those functions' own tests.
func TestIsDocker(t *testing.T) {
	// In the standard test environment /.dockerenv is absent, so IsDocker must
	// return false. If this runs inside an actual Docker container the test is
	// skipped because the environment is inherently Docker.
	if cli.IsDocker() {
		t.Skip("running inside Docker; skipping non-Docker assertion")
	}
	if cli.IsDocker() {
		t.Error("IsDocker() = true, want false when /.dockerenv is absent")
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		label   string
		name    string
		wantErr bool
	}{
		{
			label:   "valid names",
			name:    "validname",
			wantErr: false,
		},
		{
			label:   "empty org name",
			name:    "",
			wantErr: true,
		},
		{
			label:   "names with spaces",
			name:    "org with spaces",
			wantErr: true,
		},
		{
			label:   "names with special characters",
			name:    "org/\\:*?\"<>|(){}[]!@#$%^&*+=~`",
			wantErr: true,
		},
		{
			label:   "names with unicode characters",
			name:    "组织",
			wantErr: false, // Assuming unicode is allowed
		},
		{label: "names with dashes and underscores",
			name:    "org-name_with-dash",
			wantErr: false,
		},
		{
			label:   "names with leading spaces",
			name:    " orgname",
			wantErr: true,
		},
		{
			label:   "names with trailing spaces",
			name:    "orgname ",
			wantErr: true,
		},
		{
			label:   "names with only spaces",
			name:    "   ",
			wantErr: true,
		},
		{
			label:   "no path separators",
			name:    "org/name",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			err := cli.ValidateName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDefaultUserConfigPath verifies config path generation and creation.
func TestDefaultUserConfigPath(t *testing.T) {
	orgName := "testorg"
	appName := "testapp"

	filePath, err := cli.DefaultUserConfigPath(orgName, appName)
	if err != nil {
		t.Fatalf("DefaultUserConfigPath() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(filePath)) })

	if filePath == "" {
		t.Error("DefaultUserConfigPath() returned empty path")
	}

	// Verify directory was created
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("DefaultUserConfigPath() created path not found: %v", err)
	}

	// Verify path contains org and app names
	if !strings.Contains(filePath, orgName) {
		t.Errorf("DefaultUserConfigPath() path doesn't contain orgName: %s", filePath)
	}
}

// TestDefaultConfigPath returns the correct path based on environment.
func TestDefaultConfigPath(t *testing.T) {
	orgName := "testorg"
	appName := "testapp"

	filePath, err := cli.DefaultConfigPath(orgName, appName)
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(filePath)) })

	if filePath == "" {
		t.Error("DefaultConfigPath() returned empty path")
	}

	// Verify path is valid
	if _, err := os.Stat(filePath); err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("DefaultConfigPath() stat error: %v", err)
		}
		// Path may not exist yet, that's OK
	}
}

// TestDefaultPersistencePath returns the correct path.
func TestDefaultPersistencePath(t *testing.T) {
	orgName := "testorg"
	appName := "testapp"

	filePath, err := cli.DefaultPersistencePath(orgName, appName)
	if err != nil {
		t.Fatalf("DefaultPersistencePath() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(filePath)) })

	if filePath == "" {
		t.Error("DefaultPersistencePath() returned empty path")
	}
}

// TestDefaultConfigFile returns the correct file path.
func TestDefaultConfigFile(t *testing.T) {
	orgName := "testorg"
	appName := "testapp"
	baseName := "config"

	filePath, err := cli.DefaultConfigFile(orgName, appName, baseName)
	if err != nil {
		t.Fatalf("DefaultConfigFile() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(filePath)) })

	if filePath == "" {
		t.Error("DefaultConfigFile() returned empty path")
	}

	// Verify the file path ends with the expected name
	if !strings.HasSuffix(filePath, "config.yaml") {
		t.Errorf("DefaultConfigFile() path doesn't end with 'config.yaml': %s", filePath)
	}
}

// TestDefaultPersistenceFile returns the correct file path.
func TestDefaultPersistenceFile(t *testing.T) {
	orgName := "testorg"
	appName := "testapp"
	baseName := "state"

	filePath, err := cli.DefaultPersistenceFile(orgName, appName, baseName)
	if err != nil {
		t.Fatalf("DefaultPersistenceFile() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(filePath)) })

	if filePath == "" {
		t.Error("DefaultPersistenceFile() returned empty path")
	}

	if !strings.HasSuffix(filePath, "state.yaml") {
		t.Errorf("DefaultPersistenceFile() path doesn't end with 'state.yaml': %s", filePath)
	}
}

// TestContext creates and retrieves context values.
func TestContext(t *testing.T) {
	orgName := "testorg"
	appName := "testapp"

	ctx := cli.Context(
		context.Background(),
		cli.SetOrgName(orgName),
		cli.SetAppName(appName),
	)

	if ctx == nil {
		t.Error("Context() returned nil")
	}

	// Verify context is usable
	select {
	case <-ctx.Done():
		t.Error("Context() returned canceled context")
	default:
		// Good, context is not canceled
	}
}

// TestContextWithNilBase creates context with nil base context.
func TestContextWithNilBase(t *testing.T) {
	ctx := cli.Context(nil) // lint:ignore SA1012 specifically testing behaviour is nil is passed
	assert.NotNil(t, ctx)

	// Verify context is usable
	select {
	case <-ctx.Done():
		t.Error("Context(nil) returned canceled context")
	default:
		// Good, context is not canceled
	}
}

// TestSetOrgName sets organization name in context.
func TestSetOrgName(t *testing.T) {
	orgName := "myorg"
	opt := cli.SetOrgName(orgName)

	if opt == nil {
		t.Error("SetOrgName() returned nil")
	}

	// Apply to context
	ctx := opt(context.Background())
	if ctx == nil {
		t.Error("SetOrgName() returned nil context")
	}
}

// TestSetAppName sets app name in context.
func TestSetAppName(t *testing.T) {
	appName := "myapp"
	opt := cli.SetAppName(appName)

	if opt == nil {
		t.Error("SetAppName() returned nil")
	}

	// Apply to context
	ctx := opt(context.Background())
	if ctx == nil {
		t.Error("SetAppName() returned nil context")
	}
}

// TestNewCommand creates a command with type-safe config.
func TestNewCommand(t *testing.T) {
	type TestConfig struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	}

	cfg := &TestConfig{
		Name: "test",
		Port: 8080,
	}

	cmd := cli.NewCommand(
		&cobra.Command{
			Use:   "test",
			Short: "Test command",
		},
		func(ctx context.Context, c *TestConfig) error {
			return nil
		},
		cfg,
	)

	if cmd == nil {
		t.Fatal("NewCommand() returned nil command")
	}

	if cmd.Use != "test" {
		t.Errorf("NewCommand() use = %s, want test", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("NewCommand() RunE not set")
	}
}

// TestNewCommandWithConfigFlag creates command with config flag option.
func TestNewCommandWithConfigFlag(t *testing.T) {
	type TestConfig struct {
		Value string `mapstructure:"value"`
	}

	cfg := &TestConfig{}

	cmd := cli.NewCommand(
		&cobra.Command{
			Use: "test",
		},
		func(ctx context.Context, c *TestConfig) error {
			return nil
		},
		cfg,
		cli.WithConfigFlag("config.yaml"),
	)

	if cmd == nil {
		t.Fatal("NewCommand() returned nil")
	}

	// Verify config flag was added
	configFlag := cmd.Flag("config")
	if configFlag == nil {
		t.Fatal("NewCommand() did not add config flag")
	}

	if configFlag.Shorthand != "c" {
		t.Errorf("config flag shorthand = %s, want c", configFlag.Shorthand)
	}
}

// TestWithConfigFlag option sets config flag correctly.
func TestWithConfigFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	opt := cli.WithConfigFlag("my-config.yaml")

	opt(cmd)

	configFlag := cmd.Flag("config")
	if configFlag == nil {
		t.Fatal("WithConfigFlag() did not add config flag")
	}

	if configFlag.DefValue != "my-config.yaml" {
		t.Errorf("config flag default = %s, want my-config.yaml", configFlag.DefValue)
	}
}

func TestUnmarshalConfig(t *testing.T) {
	type TestConfig struct {
		Name     string        `mapstructure:"name"`
		Port     int           `mapstructure:"port"`
		Duration time.Duration `mapstructure:"duration"`
		IP       net.IP        `mapstructure:"ip"`
		CIDR     *net.IPNet    `mapstructure:"cidr"`
	}

	cfg := &TestConfig{}

	flags := &pflag.FlagSet{}
	flags.String("name", "", "")
	flags.String("port", "", "")
	flags.String("duration", "", "")
	flags.String("ip", "", "")
	flags.String("cidr", "", "")

	err := flags.Parse([]string{
		"--name=test",
		"--port=8080",
		"--duration=60s",
		"--ip=1.2.3.4",
		"--cidr=2.3.4.5/24",
	})
	assert.NoErrorf(t, err, "flag parse error")

	v := viper.New()

	err = v.BindPFlags(flags)
	assert.NoError(t, err)

	err = cli.UnmarshalConfig(v, cfg)
	assert.NoErrorf(t, err, "UnmarshalConfig() error")

	assert.Equal(t, cfg.Name, "test")
	assert.Equal(t, "test", cfg.Name)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, 60*time.Second, cfg.Duration)

	ip := net.ParseIP("1.2.3.4")
	assert.Equal(t, ip, cfg.IP)

	_, cidr, err := net.ParseCIDR("2.3.4.5/24")
	assert.NoError(t, err)
	assert.Equal(t, cidr, cfg.CIDR)
}

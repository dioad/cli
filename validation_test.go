package cli_test

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dioad/cli"
	"github.com/dioad/config/configcheck"
)

type strictTLS struct {
	CertFile string `mapstructure:"cert-file"`
}

type strictTunnel struct {
	Name      string     `mapstructure:"name"`
	TLSServer *strictTLS `mapstructure:"tls-server"`
}

type strictMail struct {
	Domain string `mapstructure:"domain"`
}

// strictFile is the type of the whole config file in these tests.
type strictFile struct {
	cli.CommonConfig `flag:"-" mapstructure:",squash"`

	Listen  string         `mapstructure:"listen"`
	Port    int            `mapstructure:"port"`
	Tunnels []strictTunnel `flag:"-" mapstructure:"tunnels"`
	Mail    strictMail     `flag:"-" mapstructure:"mail"`
}

// mailOnly is a command config that reads just part of a strictFile.
type mailOnly struct {
	Mail strictMail `flag:"-" mapstructure:"mail"`
}

// validatedFile is a strictFile that checks itself.
type validatedFile struct {
	strictFile `flag:"-" mapstructure:",squash"`
}

func (c *validatedFile) Validate() error {
	collector := configcheck.NewCollector()
	if c.Mail.Domain == "" {
		collector.Field("mail").Field("domain").Errorf("is required")
	}
	return collector.Err()
}

// runWithConfigFile runs a command whose config type is T against a config
// file holding content, plus any extra command-line args, and returns the
// config the run function received (nil if it was not called).
func runWithConfigFile[T any](t *testing.T, content string, args []string, opts ...cli.CommandOpt) (*T, error) {
	t.Helper()

	var got *T
	cmd := cli.NewCommand(
		&cobra.Command{Use: "app", SilenceUsage: true, SilenceErrors: true},
		func(_ context.Context, c *T) error {
			got = c
			return nil
		},
		new(T),
		append([]cli.CommandOpt{cli.WithConfigFlag(writeYAML(t, content))}, opts...)...,
	)
	cmd.SetArgs(args)

	ctx := cli.Context(context.Background(), cli.SetOrgName("testorg"), cli.SetAppName("testapp"))
	return got, cmd.ExecuteContext(ctx)
}

func requireFindings(t *testing.T, err error) []configcheck.Finding {
	t.Helper()
	require.Error(t, err)
	var ce *configcheck.Error
	require.ErrorAs(t, err, &ce)
	return ce.Findings
}

func TestWithStrictConfigFile_RejectsUnknownKeysWithTheirFullPath(t *testing.T) {
	got, err := runWithConfigFile[strictFile](t, `
log:
  levle: debug
status-address: ":8080"
tunnels:
  - name: web
  - name: mail
    tls-server-config: {}
  - name: api
    tls-server:
      auto-cert-config: {}
`, nil, cli.WithStrictConfigFile[strictFile]())

	assert.Nil(t, got, "the run function must not be called with an invalid config file")
	assert.Equal(t, []configcheck.Finding{
		{Path: "log.levle", Message: "unknown key"},
		{Path: "status-address", Message: "unknown key"},
		{Path: "tunnels[1].tls-server-config", Message: "unknown key"},
		{Path: "tunnels[2].tls-server.auto-cert-config", Message: "unknown key"},
	}, requireFindings(t, err))
	assert.Contains(t, err.Error(), "invalid configuration in ")
	assert.Contains(t, err.Error(), "\nlog.levle: unknown key\n")
}

func TestWithStrictConfigFile_AcceptsAValidFileAndIgnoresBoundFlags(t *testing.T) {
	got, err := runWithConfigFile[strictFile](t, `
log:
  level: debug
port: 9090
tunnels:
  - name: web
    tls-server:
      cert-file: web.pem
`, []string{"--listen", ":443"}, cli.WithStrictConfigFile[strictFile]())

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "debug", got.Logging.Level)
	assert.Equal(t, ":443", got.Listen)
	assert.Equal(t, 9090, got.Port)
	assert.Equal(t, "web.pem", got.Tunnels[0].TLSServer.CertFile)
}

func TestWithStrictConfigFile_ReportsTypeErrorsWithTheirPath(t *testing.T) {
	_, err := runWithConfigFile[strictFile](t, "port: eighty\n", nil, cli.WithStrictConfigFile[strictFile]())

	findings := requireFindings(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, "port", findings[0].Path)
}

func TestWithStrictConfigFile_ChecksAPartialCommandAgainstTheFileType(t *testing.T) {
	const fileForMail = `
port: 9090
tunnels:
  - name: web
mail:
  domain: example.com
`

	got, err := runWithConfigFile[mailOnly](t, fileForMail, nil, cli.WithStrictConfigFile[strictFile]())
	require.NoError(t, err, "keys the partial command does not use are still valid file keys")
	assert.Equal(t, "example.com", got.Mail.Domain)

	_, err = runWithConfigFile[mailOnly](t, fileForMail+"tunnel-defaults:\n  tags: [a]\n", nil, cli.WithStrictConfigFile[strictFile]())
	assert.Equal(t, []configcheck.Finding{{Path: "tunnel-defaults", Message: "unknown key"}}, requireFindings(t, err))
}

func TestWithStrictConfigFile_DoesNotReportUnknownKeysWithNoValue(t *testing.T) {
	// viper drops null and empty-map values when it reads the file, so the
	// loader never sees these keys and they carry no settings to lose.
	_, err := runWithConfigFile[strictFile](t, "port: 1\nempty: {}\nnull-value:\n", nil, cli.WithStrictConfigFile[strictFile]())

	assert.NoError(t, err)
}

func TestNewCommand_IgnoresUnknownKeysWithoutStrictConfigFile(t *testing.T) {
	got, err := runWithConfigFile[strictFile](t, "status-address: \":8080\"\nport: 9090\n", nil)

	require.NoError(t, err)
	assert.Equal(t, 9090, got.Port)
}

func TestWithConfigValidation_ReportsValidationAndUnknownKeysTogether(t *testing.T) {
	got, err := runWithConfigFile[validatedFile](t, "status-address: \":8080\"\n", nil,
		cli.WithStrictConfigFile[validatedFile](), cli.WithConfigValidation())

	assert.Nil(t, got)
	assert.Equal(t, []configcheck.Finding{
		{Path: "status-address", Message: "unknown key"},
		{Path: "mail.domain", Message: "is required"},
	}, requireFindings(t, err))
}

func TestWithConfigValidation_PassesAValidConfigToTheRunFunction(t *testing.T) {
	got, err := runWithConfigFile[validatedFile](t, "mail:\n  domain: example.com\n", nil, cli.WithConfigValidation())

	require.NoError(t, err)
	assert.Equal(t, "example.com", got.Mail.Domain)
}

func TestNewCommand_DoesNotCallValidateWithoutConfigValidation(t *testing.T) {
	got, err := runWithConfigFile[validatedFile](t, "port: 1\n", nil)

	require.NoError(t, err, "Validate would reject the missing mail.domain")
	assert.Equal(t, 1, got.Port)
}

func TestDecodeConfigFile(t *testing.T) {
	t.Parallel()

	var cfg strictFile
	err := cli.DecodeConfigFile(writeYAML(t, "port: 9090\nmail:\n  domian: x\n"), &cfg)

	assert.Equal(t, []configcheck.Finding{{Path: "mail.domian", Message: "unknown key"}}, requireFindings(t, err))
	assert.Equal(t, 9090, cfg.Port)
}

func TestDecodeConfigFile_MissingFileIsNotAFinding(t *testing.T) {
	t.Parallel()

	var cfg strictFile
	err := cli.DecodeConfigFile(t.TempDir()+"/missing.yaml", &cfg)

	require.Error(t, err)
	var ce *configcheck.Error
	assert.NotErrorAs(t, err, &ce)
}

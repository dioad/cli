# Config keys for Dioad services

Dioad services share one vocabulary for config keys, so an operator who knows
one service can read another's config. `dioad/connect`'s servers set the
convention; this page records it. The same keys are the flag and environment
names: see [Keys, flags and environment variables](#keys-flags-and-environment-variables).

A runnable service that follows these rules, with `config validate`,
`config example` and `config schema`, is in
[`examples/config-command`](../examples/config-command/README.md).

## Vocabulary

| Setting | Key | Notes |
|---|---|---|
| Logging | `log.*` | `log.level`, `log.file`, `log.use-local-time`, and so on. Provided by `cli.CommonConfig`. |
| Management listener | `management.*` | A nested block with its own `listen-address` and `tls-server`, for health, metrics and status. |
| TLS for a listener | `tls-server` | Not `tls-server-config`: the type says it is config. |
| TLS for an outbound connection | `tls-client` | Not `tls-client-config`. |
| Time to keep serving after a stop request | `drain-delay` | One key, a duration (`5s`). A load balancer sees the service not ready for this long before it stops. |
| Listening without authentication | `allow-unauthenticated`, and `allow-unauthenticated-non-loopback` | The second, narrower key lets a listener that is not on loopback run without authentication. Without it, such a listener is rejected unless the TLS config requires client certificates. Set it only when the network in front of the listener is trusted. |

`connect` implements `tls-server`, `tls-client`, `drain-delay`, `management.*`
and `allow-unauthenticated-non-loopback`. It has no plain
`allow-unauthenticated` key: the agent's management listener is controlled by
`local-auth-disabled`. A service that needs the broader setting should use
`allow-unauthenticated` and say what it covers.

## Keys, flags and environment variables

A setting has one name, derived from its config key:

| Config key | Flag | Environment variable (`<APP>` is the app name) |
|---|---|---|
| `log.level` | `--log-level` | `<APP>_LOG_LEVEL` |
| `management.listen-address` | `--management-listen-address` | `<APP>_MANAGEMENT_LISTEN_ADDRESS` |
| `cors-allowed-origins` | `--cors-allowed-origins` | `<APP>_CORS_ALLOWED_ORIGINS` |

`cli.NewCommand` names a flag after its field's `mapstructure` key path, and
binds it to that key, so the flag, the file and the environment all set the
same value. A struct squashed into its parent (`mapstructure:",squash"`) adds
no prefix. To choose a flag's name yourself, give the field a `flag:"name"`
tag; its children are then prefixed with it.

Flags generated under older names, such as `--logging-level`, are still
accepted as hidden, deprecated aliases.

## Writing a config type

- Give every field a `mapstructure` key in kebab-case and a `desc` tag. The
  `desc` is the flag's help text, the comment in `config example` and the
  description in `config schema`.
- Implement `cli.Validator` (`Validate() error`), reporting problems with
  `configcheck.Collector` so each names its key path.
- Add `cli.NewConfigCommand(kind, newConfig)` to the service's root command, so
  operators can check a file with `config validate` before starting the service.
- Guard the help text in a test: `require.Empty(t, cli.UndocumentedFlags(root))`.

## Renaming a key

Renaming a key breaks existing files, and the strict decode rejects the old
name. Until `cli` offers a key alias mechanism, rename a key in a release that
says so, and have `Validate` report the old key's presence with the new name.

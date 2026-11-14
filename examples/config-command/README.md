# config-command

A small service command with the three `config` subcommands that
`cli.NewConfigCommand` provides. Its `run` command prints a greeting and exits,
so the example starts no long-lived process.

```console
$ go run ./examples/config-command config example
# what to say
greeting: hello
# how many times to say it
repeat: 1
# how long to keep running after a stop request
drain-delay: 5s
management:
    # address of the management listener
    listen-address: 127.0.0.1:4444
```

`config validate` checks a file without running the service, and reports every
problem together. For a file with a misspelt key and a value `Validate` rejects:

```console
$ cat greeter.yaml
greeting: hi
repeat: 0
management:
  listen-adress: ":1"

$ go run ./examples/config-command config validate --config greeter.yaml
greeter.yaml:
  management.listen-adress: unknown key
  repeat must be at least 1, got 0
Error: greeter.yaml: 2 configuration problem(s) found
```

`config schema` prints a JSON Schema for editors and validators.

The flags follow the config keys: `management.listen-address` is
`--management-listen-address` on `run`. See
[Config keys for Dioad services](../../doc/config-keys.md) for the vocabulary.

`main_test.go` runs each of these commands.

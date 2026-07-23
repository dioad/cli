# Architecture Review

## Executive Summary

This is a small (4-package, ~110-symbol) library wrapping Cobra + Viper + zerolog for
building CLI applications. The code is clean, well-commented, and reasonably tested, but
one real correctness bug ships silently (`FatalError` never fires), and the recurring
architectural theme is **coupling to process-global singletons** — the global `viper`
package, the global `zerolog/log.Logger`, and an advertised-but-inert config-watcher
goroutine — which together undermine composability, testability, and the "hot-reload"
and "structured logging" features the package claims to provide. Fixing `FatalError` is
the priority; the singleton coupling is a design decision worth revisiting deliberately
rather than patching piecemeal.

## Findings

### `FatalError` never logs or exits — the event is built but never finalized

- **File(s):** `logging/logging.go:149-151`
- **Dimension(s):** Correctness
- **Priority:** High
- **Description:** `FatalError` is implemented as:
  ```go
  func FatalError(err error) {
      log.Fatal().Err(err)
  }
  ```
  zerolog's own documentation states an `Event` is "finalized by the `Msg` or `Msgf`
  method" — until one of those (or `Send`) is called, the event is never written, and
  critically, zerolog's `os.Exit(1)` side effect for `Fatal()` only fires from that same
  finalization path. Without `.Msg("")`/`.Send()`, `FatalError(err)` is a complete no-op:
  it neither logs the error nor terminates the process, despite its name and its exported
  contract. `gograph query FatalError` confirms this function has no production callers
  today, but it is an exported API surface that any consumer of this library may already
  be relying on (or will rely on) to terminate on unrecoverable errors — silently doing
  nothing is the worst failure mode for a function named `FatalError`. Notably, the
  existing test (`logging/logging_test.go:182`, `TestFatalError`) documents that it
  "can't easily test FatalError as it calls log.Fatal which exits" — the test author's
  own assumption confirms the bug went unnoticed because it looks correct at a glance.
- **Recommended fix:** Add `.Msg(err.Error())` (or `.Send()`) to finalize the event:
  ```go
  func FatalError(err error) {
      log.Fatal().Err(err).Msg(err.Error())
  }
  ```
  Then add a real test using `os.Exit` interception (e.g. via a subprocess test with
  `exec.Command` re-invoking the test binary, the standard Go pattern for testing
  functions that call `os.Exit`), replacing the current placebo test.
- **Status:** Resolved in `a7e66d6`
- **Complexity delta:** 0 -> 0 (no branching added; `logging.go`'s other functions
  unaffected. New subprocess test verified to fail against the pre-fix code.)

### `InitConfig` advertises "hot-reloading" but never re-applies reloaded config, and leaks a watcher goroutine per call

- **File(s):** `config.go:130-219` (`InitConfig`, `WithoutWatchConfig`)
- **Dimension(s):** Correctness, Architecture
- **Priority:** Medium-High
- **Description:** `InitConfig`'s doc comment promises "Configuration hot-reloading via
  Viper watchers," and `options.watchConfig` gates a call to `v.WatchConfig()`. However,
  `v.WatchConfig()` only updates Viper's *internal* key-value store when the file
  changes — `gograph query OnConfigChange` returns no results anywhere in the repo, so
  no `v.OnConfigChange(func(fsnotify.Event){...})` callback is ever registered to
  re-`Unmarshal` the change into the caller's `cfg`/`CommonConfig` structs. The net
  effect: the advertised "hot-reload" feature changes nothing the caller can observe.
  Compounding this, `InitConfig` does not return the `*viper.Viper` instance it
  constructs, so even a future caller wanting to add their own `OnConfigChange` handler
  has no handle to do so, and the `WatchConfig`-spawned fsnotify watcher goroutine (one
  per `InitConfig` call) can never be stopped — each call to `InitConfig` with the
  default options leaks one background goroutine for the lifetime of the process.
- **Recommended fix:** Either (a) implement the feature properly — register
  `v.OnConfigChange` to re-run `UnmarshalConfig(v, cfg)` (and reconfigure logging if
  `c.Logging` changed) on file-change events, and return the `*viper.Viper` (or a small
  wrapper with a `Stop()`/`Close()`) so callers can manage the watcher's lifecycle; or
  (b) if hot-reload isn't actually a supported feature, remove `v.WatchConfig()` and the
  `WithoutWatchConfig` option entirely and correct the doc comment. Leaving the
  half-built version in place is the worst of both — it reads as a feature but silently
  isn't one.
- **Status:** Resolved in `a059e26`. Option (a) was rejected during implementation
  design review: registering `v.OnConfigChange` to re-`Unmarshal` into the caller's
  `cfg` writes that struct from the fsnotify goroutine while the caller/`execFunc` reads
  it from the main goroutine — an unsynchronized read/write the project's own
  `go test -race` gate would (correctly) flag. Safe delivery needs the caller to
  participate (mutex, atomic snapshot, getter), which is an API change and therefore out
  of scope for a non-breaking fix. Went with a variant of option (b): removed the
  `v.WatchConfig()` call from `InitConfig` (the exported `WithoutWatchConfig` option is
  kept, now documented as a deprecated no-op, since removing an exported symbol would be
  a breaking change) and corrected `InitConfig`'s doc comment. Added
  `TestInitConfigDoesNotLeakWatcherGoroutine`, verified to fail against the pre-fix code
  (32 live goroutines vs. an expected ≤4 after 10 calls) and pass after.
- **Complexity delta:** `InitConfig`: 15 -> 13 (one branch removed); all other
  functions in `config.go` unchanged.

### Two parallel, inconsistent config-initialization paths: global Viper singleton vs. per-call instance

- **File(s):** `config.go:37-93` (`InitViperConfig`, `InitViperConfigWithFlagSet`) vs. `config.go:138-219` (`InitConfig`)
- **Dimension(s):** Inconsistencies, Architecture, Maintainability
- **Priority:** Medium
- **Description:** The package ships two independent config-loading implementations
  with different behavior and different coupling models:
  - `InitViperConfig` / `InitViperConfigWithFlagSet` operate entirely on the **global**
    `viper` package-level singleton (`gograph query "viper."` shows every one of
    `viper.BindPFlags`, `viper.SetConfigName`, `viper.AddConfigPath`, `viper.AutomaticEnv`,
    etc. called only from `InitViperConfigWithFlagSet`). Any process that calls this
    twice (e.g. two subcommands, or a test suite) mutates shared global state; the
    package's own test (`initconfig_test.go:177-178`) documents this explicitly: *"Note:
    InitViperConfigWithFlagSet uses the global Viper instance. These tests must not run
    in parallel to avoid interference."* — a direct violation of this project's own
    testing standard ("Run in Parallel: tests must be fully independent").
  - `InitConfig` instead constructs a local `viper.New()` per call and is fully
    parallel-safe and composable — the design the library should be standardizing on.
  - The two paths also disagree on environment variable key replacement:
    `InitViperConfigWithFlagSet` replaces only `-` (`strings.NewReplacer("-", "_")`,
    `config.go:77`), while `InitConfig` replaces both `-` and `.`
    (`strings.NewReplacer("-", "_", ".", "_")`, `config.go:152`) — so a nested config key
    like `log.level` maps to `LOG_LEVEL` under one path and is left unmapped under the
    other.
  - **Compounding this, the package's own doc comments contradict each other on
    precedence.** `doc.go:53-61` states the merge order (highest to lowest, "last wins")
    as: file at `/etc/...` → file at `$HOME/.config/...` → CLI flags → env vars →
    `--config` file. `InitViperConfig`'s doc comment (`config.go:44-49`) states the
    opposite ordering — flags highest, then `--config` file, then env vars, then
    standard-location files lowest. Both cannot be correct, and neither matches Viper's
    actual fixed precedence order (explicit `Set` > flag > env > config file > default),
    so a consumer reading either doc comment will form a wrong mental model of which
    source wins.
- **Recommended fix:** Deprecate `InitViperConfig`/`InitViperConfigWithFlagSet` in favor
  of `InitConfig`'s local-instance pattern (or refactor them to build a local
  `viper.New()` too), unify the env-key replacer behavior between the two paths, and
  rewrite `doc.go`'s and `InitViperConfig`'s precedence descriptions to match Viper's
  actual documented precedence — verify against the real behavior with a table test
  rather than prose.
- **Status:** Resolved in `5bf716a`. Kept both deprecated functions as-is behaviorally
  (no change to the global-singleton mechanics or the env-key replacer difference —
  deprecated code shouldn't get further behavior changes, only documentation pointing
  callers away from it) and added standard `// Deprecated:` godoc lines to both,
  referencing `InitConfig`. Rewrote `doc.go`'s "Configuration Loading Order" section and
  `InitViperConfig`'s doc comment to state the same, correct precedence order (verified
  against Viper's official docs via Context7): command-line flags > environment
  variables > config file > defaults.
- **Complexity delta:** No change (doc-comment-only edit); `InitViperConfigWithFlagSet`
  remains at 6, `InitViperConfig` remains untracked by gocognit (below its reporting
  threshold, unaffected).

### `Context()`/`SetOrgName`/`SetAppName` establish a context-propagation pattern, but the configured logger is never attached to it

- **File(s):** `config.go:391-438` (`Context`, `SetOrgName`, `SetAppName`), `logging/logging.go:119-146` (`ConfigureLogOutput`)
- **Dimension(s):** Architecture, Maintainability
- **Priority:** Medium
- **Description:** The library provides a clean, hexagonal-style context-propagation
  convention for org/app metadata (`cli.Context`, `cli.SetOrgName`, `cli.SetAppName`,
  consumed later by `CobraRunE` via `OrgNameFromContext`/`AppNameFromContext`), and
  `CobraRunE`'s `execFunc` signature is `func(context.Context, *T) error` — signaling
  that downstream application code is expected to work through `context.Context`. But
  `ConfigureCmdLogger`/`ConfigureLogOutput` only ever mutate the global
  `github.com/rs/zerolog/log.Logger` — there is no equivalent `SetLogger`/context
  injection to make the configured logger retrievable via `zerolog.Ctx(ctx)` from within
  `execFunc`. Every consumer is therefore forced to depend on the global `log.Logger`
  inside their command bodies even though the library already hands them a
  `context.Context` built for exactly this kind of propagation. This is a missed
  opportunity in the library's own abstraction, not a bug — but it means the "structured
  logging" feature this package advertises doesn't compose with the "context management"
  feature it also advertises.
- **Recommended fix:** After `ConfigureLogOutput` builds `log.Logger`, also inject it
  into the context passed to `execFunc` (e.g. have `CobraRunE` call
  `log.Logger.WithContext(cmd.Context())` before invoking `execFunc`, and consider
  setting `zerolog.DefaultContextLogger = &log.Logger` at the same point) so command
  bodies can use `zerolog.Ctx(ctx)` instead of the global logger.
- **Status:** Resolved in `e8217e3`. `CobraRunE` now does
  `ctx := log.Logger.WithContext(cmd.Context())` and passes that to `execFunc`. Did
  **not** also set `zerolog.DefaultContextLogger` — per this repo's own zerolog
  conventions, that's set once at the entry point (`main`/`Exec`), not inside a
  per-command `RunE` that can run repeatedly. Purely additive; no signature changes.
  Added `TestCobraRunEInjectsLoggerIntoContext`, verified to fail (reports
  `zerolog.Disabled`) against the pre-fix code.
- **Complexity delta:** `CobraRunE`: 7 -> 7 (no new branching, straight-line context
  wrap).

### Inconsistent test assertion style: raw `t.Error`/`t.Fatalf` alongside testify in the same package

- **File(s):** `logging/logging_test.go` (entirely raw), `config_test.go` (`TestIsDocker`, `TestValidateName`, `TestDefaultUserConfigPath`, etc. — raw), vs. `integration_test.go`, `initconfig_test.go`, `commandparts_test.go` (consistently testify)
- **Dimension(s):** Inconsistencies, Modern Practices
- **Priority:** Low-Medium
- **Description:** The project's own testing standard mandates testify (`assert`/
  `require`) for all assertions, and `logging/example_test.go`, `logging/logfile_test.go`,
  `integration_test.go`, and `initconfig_test.go` all follow it. But `logging/logging_test.go`
  uses raw `if x != y { t.Errorf(...) }` throughout (e.g. lines 23-37, 84-90, 102-137,
  192-221), and `config_test.go` mixes both styles in the same file (`TestIsDocker`,
  `TestValidateName` at lines 25-101 are raw; `TestContextWithNilBase`,
  `TestUnmarshalConfig` at lines 238-249, 395-416 use testify). This makes failure
  messages inconsistent in quality (raw `t.Errorf` calls here don't produce the
  richer diffs testify provides) and signals the standard isn't enforced uniformly.
- **Recommended fix:** Convert the remaining raw assertions in `logging/logging_test.go`
  and the older parts of `config_test.go` to `assert`/`require`, consistent with the rest
  of the suite. Low risk, mechanical change — a good candidate for a single follow-up
  commit per file.
- **Status:** Resolved in `e89380d`. Converted all raw `t.Error`/`t.Errorf`/`t.Fatalf`
  checks in both files to `assert`/`require`. No behavior change; a couple of
  incidental simplifications fell out naturally (e.g. `TestIsDocker`'s redundant
  double-check of the same condition collapsed to one skip-guard plus one assertion).
- **Complexity delta:** Every touched function decreased or stayed the same; none
  increased. Notably `TestConfigureLogLevel`: 6 -> 4, `TestDefaultConfigPath`: 5 -> 1,
  `TestValidateName`: 4 -> 1 (via a branch-free `assert.Equal(t, tt.wantErr, err != nil)`
  instead of an if/else, after an initial if/else draft measured 4 -> 5 and was
  reworked).

### Minor: redundant `log.Logger` construction in `ConfigureLogOutput`

- **File(s):** `logging/logging.go:119-146`
- **Dimension(s):** Maintainability
- **Priority:** Low
- **Description:** `log.Logger` is constructed twice — once at line 129 to guarantee a
  writable logger exists while resolving file output, and again unconditionally at line
  141 with whatever `logOutput` ended up being. This is intentional per the inline
  comment ("Setup logging to stdout by default so we have somewhere to log any errors
  configuring logging") and not a bug, but it's easy to misread as duplicated/dead code
  during future edits.
- **Recommended fix:** Optional — a one-line comment above line 141 clarifying "this
  replaces the temporary stdout logger above once file output (if any) is resolved"
  would remove the ambiguity for future readers. Not worth a structural change.
- **Status:** Resolved in `03cb631`. Added the clarifying comment; no logic change.
- **Complexity delta:** `ConfigureLogOutput`: 5 -> 5 (comment-only).

## Priority Table

| Priority | Title | File(s) |
|----------|-------|---------|
| High | `FatalError` never logs or exits | `logging/logging.go` |
| Medium-High | `InitConfig` hot-reload is inert and leaks a watcher goroutine | `config.go` |
| Medium | Two parallel, inconsistent config-init paths (global vs. local Viper) | `config.go`, `doc.go` |
| Medium | Context propagation pattern doesn't extend to the configured logger | `config.go`, `logging/logging.go` |
| Low-Medium | Inconsistent test assertion style (raw vs. testify) | `logging/logging_test.go`, `config_test.go` |
| Low | Redundant `log.Logger` construction (intentional, under-commented) | `logging/logging.go` |

# AGENTS.md - Go Project Guidelines

This document outlines the essential checks that must pass before any task is considered complete and provides guidelines for generating idiomatic Go code within this project.

## Core Principles

### Definition of Done

Before a task is marked as complete, the following verification suite *must* execute successfully. This ensures code
quality, correctness, and adherence to project standards.

This command automates the following steps:

   documentation (MD, HTML, JSON).
2. **`go build .`**: Compiles the project to ensure syntactic correctness and dependency resolution.
3. **`go vet ./...`**: Reports suspicious constructs and potential bugs.
4. **`go test -race ./...`**: Runs unit tests with the data race detector enabled.

**Requirement:** All checks must pass without errors or warnings. Any issues must be addressed or explicitly justified.


### Idiomatic Go Coding Standards

Generated code should adhere to the following principles to maintain consistency, readability, and Go best practices. The goal is for generated code to be indistinguishable from hand-written idiomatic Go code.

1.  **Formatting (`gofmt`):**
    *   All generated code *must* be formatted using `gofmt`. This ensures consistent style across the entire codebase.

2.  **Naming Conventions:**
    *   **Public (Exported) Elements:** Use `CamelCase` (e.g., `MyStruct`, `MyFunction`).
    *   **Private (Unexported) Elements:** Use `camelCase` (e.g., `myStruct`, `myFunction`).
    *   **Acronyms:** Acronyms should be all uppercase in exported names (e.g., `HTTPClient`, `JSONMarshal`), and all lowercase in unexported names (e.g., `httpClient`, `jsonMarshal`).
    *   **Interface Names:** Short, descriptive, often ending with "er" (e.g., `Reader`, `Writer`, `Manager`).

3.  **Error Handling:**
    *   **Explicit Error Checks:** Always check for errors returned by functions. Do not ignore them.
    *   **Error Wrapping:** Use `fmt.Errorf("message: %w", err)` to wrap errors, preserving the original error context. This allows for programmatic inspection using `errors.Is` and `errors.As`.
    *   **Custom Error Types:** Define custom error types for specific, expected error conditions when it benefits API consumers (e.g., `ErrNotFound`, `ErrInvalidInput`).
    *   **Clear Error Messages:** Error messages should be informative and actionable for the user or developer.

4.  **Context (`context.Context`):**
    *   Functions that perform I/O operations, interact with external services, or are long-running/cancellable *must* accept `context.Context` as their first argument.
    *   Propagate the context through function calls.

5.  **Concurrency:**
    *   Use goroutines and channels for concurrent operations.
    *   Utilize `sync.WaitGroup` for waiting on multiple goroutines to complete.
    *   Avoid shared mutable state where possible. If necessary, protect with `sync.Mutex` or `sync.RWMutex`.
    *   Avoid global mutable state.

6.  **Imports:**
    *   Organize imports into standard Go library, then third-party libraries, then internal project packages, each in separate groups.
    *   Use blank imports (`_`) only for side effects (e.g., registering a `driver`).

7.  **Comments (`Godoc`):**
    *   All exported (public) types, functions, methods, and constants *must* have clear and concise Godoc comments.
    *   Comments should explain *what* the element does, *why* it exists, its parameters, and what it returns (if applicable).
    *   Internal (unexported) elements should have comments when their purpose is not immediately obvious from the code.

8.  **Package Design:**
    *   Packages should be small, focused, and have a single responsibility.
    *   Exported APIs should be minimal and clear, following the principle of least surprise.

9.  **Generics (Go 1.18+):**
    *   Use generics when they genuinely improve code reuse, type safety, and readability without adding unnecessary complexity. Avoid using generics for the sake of it.

10. **Security:**
    *   Generated code should follow secure coding practices, including input validation, avoiding hardcoded credentials, and using secure defaults for configurations (e.g., TLS).

Following these guidelines ensures that generated code is maintainable, understandable, and integrates seamlessly with the rest of the Go project.

### Patterns for Testability

A comprehensive testing strategy ensures code quality, correctness, and integration between backend and frontend
components. **Solutions should prefer testing via Go's native testing mechanisms** (`go test`) whenever possible.

#### Core Principle: Testability Through Architecture

**Poor Architecture** → Complex integration tests + mocks + external dependencies  
**Good Architecture** → Simple unit tests + clear boundaries + dependency injection

When a component cannot be easily unit tested:

1. **First:** Suggest architectural improvements (dependency injection, interface extraction, package separation).
2. **Second:** If architectural changes are outside scope, document the limitation and use integration tests.
3. **Never:** Accept unmaintainable test code as the solution.

#### Red Flags for Refactoring

If any of these are true, the package likely needs architectural improvements:

1. **"I need to mock many dependencies"** → **Refactor:** Use dependency injection via interfaces.
    * Instead of: `type Service struct { db *sql.DB; cache *redis.Client; logger Logger }`
    * Prefer: `type Service struct { store Storage; cache Cache; logger Logger }` where each is an interface.
    * Benefit: Tests can provide simple mock implementations.

2. **"The test requires running the full server"** → **Refactor:** Extract logic into a testable layer.
    * Instead of: Testing HTTP handlers directly with `httptest.Server`.
    * Prefer: Test business logic separately, then test HTTP layer with thin wrappers.
    * Benefit: 90% of code tested at unit level, only HTTP marshaling tested at integration level.

3. **"I can't test this without external services"** → **Refactor:** Use interfaces for external dependencies.
    * Instead of: `func (s *Service) ProcessTunnel(db *sql.DB, client *http.Client) error`
    * Prefer: `func (s *Service) ProcessTunnel(ctx context.Context) error` with injected dependencies set during
      construction.
    * Benefit: Tests inject mock services, production injects real ones.

4. **"The package is too complex to test"** → **Refactor:** Break into smaller packages.
    * Single Responsibility Principle: Each package should do one thing well.
    * Clear boundaries: Internal detail vs. exported interface.
    * Benefit: Each component testable independently.

#### Recommended Testing Structure

```
package/
  ├── interface.go         # Exported interfaces (what consumers depend on)
  ├── implementation.go    # Internal implementation
  ├── implementation_test.go  # Unit tests (>80% coverage)
  └── doc.go             # Package documentation

key_points:
  - Interface file defines contracts
  - Implementation is internal detail
  - Tests import only the package, create mocks from exported interfaces
  - External dependencies injected via constructor
```

#### Example: Refactoring for Testability

**Before (Hard to test):**

```go
type Handler struct {
db *sql.DB
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// Logic directly querying database
// Can't test without database
}
```

**After (Easy to test):**

```go
// interface.go - What consumers depend on
type TunnelStore interface {
GetTunnel(ctx context.Context, id string) (*Tunnel, error)
}

type TunnelHandler struct {
store TunnelStore
}

func (h *TunnelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// Calls store interface, doesn't know or care about implementation
}

// tunnelhandler_test.go - Easy to test
type mockStore struct { /* implements TunnelStore */ }

func TestTunnelHandler(t *testing.T) {
handler := &TunnelHandler{store: &mockStore{}}
// Test with mock, no database needed
}
```

## Verification Suites

### Unit Testing (Primary Testing Method)

Unit tests verify individual package behavior in isolation.

```bash
go test -race ./...
```

* **Requirement:** All tests must pass with race detector enabled.
* **Coverage:** Aim for >80% coverage on critical paths.
* **When to Run:** After any code change.
* **Best Practice:**
    * Test packages should import only the package under test and standard library/common utilities.
    * If a test needs to mock complex dependencies, refactor the package to use dependency injection.
    * If a test requires external services, use interfaces and testable constructors.


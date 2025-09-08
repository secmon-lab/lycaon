# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lycaon is a Slack-based incident management service that processes Slack messages, analyzes them using LLM (Gemini), and manages incident response through Slack integration.

## Architecture Reference

This project follows the architecture patterns from `tmp/warren/`. When implementing features:
- Directory structure should mirror warren's layout
- Package organization should follow warren's patterns
- Use warren's implementation as reference for similar functionality

## Core Dependencies

### Essential Go Packages
- **CLI Framework**: `github.com/urfave/cli/v3` - ALL CLI and environment variable handling
- **Logging**: `slog` with `github.com/m-mizutani/clog` for console output
- **Error Handling**: `github.com/m-mizutani/goerr/v2` - Always wrap errors with context
- **Testing**: `github.com/m-mizutani/gt` - Use Helper Driven Testing style
- **LLM**: `github.com/m-mizutani/gollem` - Direct usage, no wrapper needed
- **Slack SDK**: `github.com/slack-go/slack` - Use directly as interface

### Authentication & Storage
- **Auth**: Google Default Application Credential (no API keys in env)
- **Database**: Firestore for persistence, memory implementation for testing
- **Session**: session_id + session_secret in HTTPOnly cookies

## Common Development Commands

### Building and Running
```bash
go run . serve              # Start HTTP server
go build                     # Build binary
go test ./...               # Run all tests
go test ./pkg/path/...      # Run specific package tests
```

### Task Management
```bash
task                        # Run default tasks
task mock                   # Generate mocks using moq
```

## Project Structure

```
lycaon/
├── main.go                # Entry point
├── pkg/
│   ├── cli/              # CLI commands and config
│   ├── controller/       # HTTP/Slack handlers
│   ├── usecase/          # Business logic
│   ├── domain/           # Domain models and interfaces
│   ├── repository/       # Data persistence (Firestore/Memory)
│   └── utils/            # Utilities
└── frontend/             # TypeScript/React frontend
```

## Environment Variables

```bash
# Server
LYCAON_ADDR=:8080

# Slack
LYCAON_SLACK_CLIENT_ID=xxx
LYCAON_SLACK_CLIENT_SECRET=xxx  
LYCAON_SLACK_SIGNING_SECRET=xxx
LYCAON_SLACK_OAUTH_TOKEN=xxx

# Firestore
LYCAON_FIRESTORE_PROJECT=xxx
LYCAON_FIRESTORE_DATABASE=xxx

# Gemini (uses Google Default Application Credential)
LYCAON_GEMINI_PROJECT=xxx       # GCP Project ID
LYCAON_GEMINI_MODEL=gemini-1.5-flash

# Logging
LYCAON_LOG_LEVEL=info
```

## Implementation Rules

### Testing
- Test files must match source file names: `foo.go` → `foo_test.go`
- Use memory repository for repository tests, not mocks
- LLM mocks: use gollem's built-in mock implementation

### Repository Layer
- Firestore stores only Slack messages (no user data, no sessions)
- Memory implementation must match Firestore interface exactly
- Never use firestore tags on struct fields (`firestore:"fieldname"`). They cause bugs and must be avoided completely
- Do not use json tags on struct fields unless explicitly required for JSON output. Only add json tags when there's a clear requirement to output JSON

### Configuration
- Do not create unified Config struct
- Handle each config (Server, Slack, Firestore, etc.) individually in serve command
- All CLI options and environment variables must be handled through `github.com/urfave/cli/v3`
- Never use `os.Getenv()` directly except in tests. All environment variable access must go through cli/v3 flags
- Use cli/v3 flag definitions with `EnvVars` field for environment variable support

### Controller Layer Responsibilities
- Controllers must only route requests and call ONE usecase method per flow
- Controllers must not contain business logic, error formatting, or UI building
- All business logic, data validation, and error handling must be in usecase layer
- Controllers should only:
  1. Parse/validate request format (JSON, etc.)
  2. Extract parameters from request
  3. Call exactly ONE usecase method
  4. Return the result
- Example violation: Controller getting data, building UI blocks, handling errors separately
- Example correct: Controller calls `usecase.HandleCreateIncident()` and returns result

### Slack Integration
- Always verify Slack signatures (X-Slack-Signature)
- Handle challenge requests for URL verification
- Process events and interactions separately

## Spec-Driven Development

This project uses spec-driven development. Specifications are in `.cckiro/specs/`:
- `req.md`: Requirements document
- `design.md`: Technical design
- `impl.md`: Implementation plan

Always refer to these documents when implementing features.

## GitHub Actions

CI/CD workflows in `.github/workflows/`:
- `test.yml`: Run tests and checks
- `lint.yml`: golangci-lint
- `build.yml`: Build verification
- `frontend.yml`: Frontend build and checks

## Restrictions and Rules

### Error Handling

Using `github.com/m-mizutani/goerr/v2` for enhanced error handling:

#### Creating Errors
```go
// Create basic error
err := goerr.New("operation failed")

// Create error with context
err := goerr.New("user validation failed", 
    goerr.V("userID", userID),
    goerr.V("timestamp", time.Now()))
```

#### Wrapping Errors
```go
// Always wrap errors to preserve context and stack trace
if err := someOperation(); err != nil {
    return goerr.Wrap(err, "failed to process user data",
        goerr.V("userID", userID),
        goerr.V("operation", "validation"))
}
```

#### Adding Context with goerr.V()
- Use `goerr.V(key, value)` to add contextual information
- Helps with debugging and error tracking
- Include relevant IDs, parameters, and state information

#### Error Tags for Categorization
```go
// Define error tags as package variables
var ErrTagNotFound = goerr.NewTag("not_found")
var ErrTagValidation = goerr.NewTag("validation")

// Create tagged errors
err := goerr.New("user not found", 
    goerr.T(ErrTagNotFound),
    goerr.V("userID", userID))

// Check error tags
if goerr.HasTag(err, ErrTagNotFound) {
    // Handle not found scenario
}
```

#### Sentinel Errors
```go
// Define as package variables
var ErrNotFound = goerr.New("not found")

// Always wrap when returning
return goerr.Wrap(ErrNotFound, "failed to get user",
    goerr.V("userID", userID))

// Check using errors.Is()
if errors.Is(err, ErrNotFound) {
    // handle not found case
}
```

#### Best Practices
- Never compare errors using string matching
- Always add meaningful context with `goerr.V()`
- Use error tags for categorization and handling
- Use `%+v` format for printing errors with stack traces during debugging
- Wrap all external errors to add context and preserve stack traces

### Directory

- When you are mentioned about `tmp` directory, you SHOULD NOT see `/tmp`. You need to check `./tmp` directory from root of the repository.

### Exposure policy

In principle, do not trust developers who use this library from outside

- Do not export unnecessary methods, structs, and variables
- Assume that exposed items will be changed. Never expose fields that would be problematic if changed
- Use `export_test.go` for items that need to be exposed for testing purposes

### Check

When making changes, before finishing the task, always:
- Run `go vet ./...`, `go fmt ./...` to format the code
- Run `golangci-lint run ./...` to check lint error
- Run `gosec -exclude-generated -quiet ./...` to check security issue
- Run tests to ensure no impact on other code

### Language

All comment and character literal in source code must be in English

### Testing

- Test files should have `package {name}_test`. Do not use same package name
- Test file name convention is: `xyz.go` → `xyz_test.go`. Other test file names (e.g., `xyz_e2e_test.go`) are not allowed.
- Repository Tests Best Practices:
  - Always use random IDs (e.g., using `time.Now().UnixNano()`) to avoid test conflicts
  - Never use hardcoded IDs like "msg-001", "user-001" as they cause test failures when running in parallel
  - Always verify ALL fields of returned values, not just checking for nil/existence
  - Compare expected values properly - don't just check if something exists, verify it matches what was saved
  - For timestamp comparisons, use tolerance (e.g., `< time.Second`) to account for storage precision

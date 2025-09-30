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
LYCAON_GEMINI_PROJECT_ID=xxx       # GCP Project ID
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

### User Management
- **User ID Principle**: Always use Slack User ID as the primary User.ID field
- New users MUST be created with `User.ID = types.UserID(slackUserID)`
- Never use `types.NewUserID()` for user creation (it generates UUIDs)
- Firestore document ID for users is always the Slack User ID
- Existing UUID-based users in the database remain functional but new users follow Slack ID principle

### Configuration
- Do not create unified Config struct
- Handle each config (Server, Slack, Firestore, etc.) individually in serve command
- All CLI options and environment variables must be handled through `github.com/urfave/cli/v3`
- Never use `os.Getenv()` directly except in tests. All environment variable access must go through cli/v3 flags
- Use cli/v3 flag definitions with `EnvVars` field for environment variable support

### Controller Layer Responsibilities (Clean Architecture)

Controllers serve as the interface adapters between external requests and business logic. Their responsibilities are strictly limited:

#### Primary Responsibilities
- **Request Parsing**: Decompose Slack messages and extract necessary information
- **Data Preparation**: Organize data needed for usecase processing
- **Async Control**: ALL UseCase calls MUST be dispatched via async.Dispatch
- **Immediate Response**: Return 200 status immediately after successful parsing
- **Single UseCase Call**: Call exactly ONE usecase method per workflow
- **Response Formatting**: Format and return responses appropriately

#### Specific Implementation Patterns
- Controllers should only:
  1. Parse/validate request format (JSON, etc.)
  2. Extract parameters from request and organize into structured data
  3. Create background context via `async.NewBackgroundContext(ctx)`
  4. Dispatch usecase call via `async.Dispatch(backgroundCtx, func...)`
  5. Return nil immediately to send 200 response to Slack
- **CRITICAL**: ALL UseCase calls must be async dispatched - NO synchronous calls
- **CRITICAL**: Return 200 immediately after successful message interpretation
- **CRITICAL**: Each interaction workflow must call exactly ONE usecase method
- **CRITICAL**: Never call multiple usecases from a single controller method
- **CRITICAL**: Never mix incident and task operations in the same controller method
- **CRITICAL**: Controllers must NOT contain business logic, error formatting, or UI building

#### Examples
- ✅ **Correct**: `interaction.go` parses Slack payload, extracts data, dispatches via `async.Dispatch`, returns nil immediately
- ✅ **Correct**: `event.go` validates message, dispatches via `async.Dispatch`, returns nil to send 200 response
- ✅ **Correct**: ALL usecase calls wrapped in `async.Dispatch(backgroundCtx, func...)`
- ✅ **Correct**: Structured data types like `SlackInteractionData` for clean controller-usecase interface
- ❌ **Violation**: Direct synchronous usecase calls (causes Slack timeouts)
- ❌ **Violation**: Waiting for usecase completion before returning response
- ❌ **Violation**: Controller getting data, building UI blocks, handling errors separately
- ❌ **Violation**: Controller calling both incidentUC and taskUC in same method
- ❌ **Violation**: Controller implementing business validation logic
- ❌ **Violation**: Passing raw `[]byte` payload directly to usecase without parsing

#### UseCase Layer Boundaries
- **UseCase Responsibility**: Business logic execution, domain model manipulation, error handling, UI component building
- **Controller Responsibility**: Request decomposition, data preparation, usecase orchestration, async dispatch control

### Async Processing Principles

All controller methods that handle Slack events and interactions MUST follow async processing patterns to prevent timeouts:

#### Mandatory Async Dispatch Pattern
```go
// REQUIRED pattern for ALL controller methods
func (h *Handler) HandleRequest(ctx context.Context, payload []byte) error {
    // 1. Parse and validate request format
    // 2. Extract necessary data
    // 3. Create background context
    backgroundCtx := async.NewBackgroundContext(ctx)
    
    // 4. Dispatch usecase processing asynchronously
    async.Dispatch(backgroundCtx, func(asyncCtx context.Context) error {
        // Call exactly ONE usecase method
        return h.usecase.ProcessRequest(asyncCtx, data)
    })
    
    // 5. Return immediately to send 200 response
    return nil
}
```

#### Critical Requirements
- **NEVER** call usecases synchronously in controllers
- **ALWAYS** use `async.Dispatch` for usecase calls
- **ALWAYS** return immediately after dispatching
- **ALWAYS** use `async.NewBackgroundContext(ctx)` for background processing
- **ONE** usecase method call per workflow
- Controllers must send 200 response immediately after successful data interpretation

#### Files Following This Pattern
- `pkg/controller/slack/event.go` - All message and mention event handlers
- `pkg/controller/slack/interaction.go` - All interaction handlers (buttons, modals, shortcuts)

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

## Clean Architecture Reference Implementation

### Controller Layer Example
See `pkg/controller/slack/interaction.go` for proper controller implementation:
- Parses Slack interaction payload into structured data
- Extracts necessary information for usecase processing  
- Manages sync vs async processing based on interaction type
- Calls single usecase method per interaction type
- Returns immediate responses where required

### UseCase Layer Example
See `pkg/usecase/slack_interaction.go` for proper usecase implementation:
- Receives structured data from controller
- Executes business logic and domain operations
- Handles error conditions and validation
- Manages cross-cutting concerns (logging, metrics)
- Returns results in domain-appropriate format

### Interface Design
See `pkg/domain/interfaces/usecase.go` for clean interface boundaries:
- `SlackInteractionData` struct for controller-usecase data transfer
- Separate methods for different interaction types (HandleBlockActions, HandleViewSubmission, etc.)
- Clear separation of concerns between parsing (controller) and processing (usecase)

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

### String Constants Naming Convention

All string constants MUST use lowercase with underscores (snake_case) format.

- ✅ **Correct**: `"todo"`, `"follow_up"`, `"completed"`, `"in_progress"`
- ❌ **Violation**: `"follow-up"`, `"followUp"`, `"FOLLOW_UP"`, `"Follow-Up"`

**Examples:**
```go
const (
    TaskStatusTodo TaskStatus = "todo"
    TaskStatusFollowUp TaskStatus = "follow_up"  // NOT "follow-up"
    TaskStatusCompleted TaskStatus = "completed"
)
```

This rule applies to ALL string constants including:
- Enum values in Go code
- GraphQL enum values
- Status constants
- Any other string literals used as constants

Never use uppercase constants, CamelCase, or kebab-case for string constants.

### Model Changes and Dependencies

When changing domain models (especially enum values), always update all dependent files in the correct order:

1. **Go Model** (e.g., `pkg/domain/model/task.go`)
2. **GraphQL Schema** (`graphql/schema.graphql`)
3. **Regenerate GraphQL Code** (`task graphql`)
4. **Frontend Type Definitions** (`frontend/src/types/*.ts`)
5. **Frontend Components** (any `.tsx` files using the types)
6. **Run Tests** (`zenv go test ./...`)

**Example**: Changing any enum value requires updating:
- Go constants and types
- GraphQL schema definitions
- Generated GraphQL code
- Frontend type definitions
- All components and logic using those types

Never change models in isolation - always consider the full dependency chain.

### Testing

- Test files should have `package {name}_test`. Do not use same package name
- Test file name convention is: `xyz.go` → `xyz_test.go`. Other test file names (e.g., `xyz_e2e_test.go`) are not allowed.
- Repository Tests Best Practices:
  - Always use random IDs (e.g., using `time.Now().UnixNano()`) to avoid test conflicts
  - Never use hardcoded IDs like "msg-001", "user-001" as they cause test failures when running in parallel
  - Always verify ALL fields of returned values, not just checking for nil/existence
  - Compare expected values properly - don't just check if something exists, verify it matches what was saved
  - For timestamp comparisons, use tolerance (e.g., `< time.Second`) to account for storage precision

# Lycaon

<p align="center">
  <img src="./docs/images/logo_v0.png" height="128" />
</p>

[![Unit test](https://github.com/secmon-lab/lycaon/actions/workflows/test.yml/badge.svg)](https://github.com/secmon-lab/lycaon/actions/workflows/test.yml)
[![Lint](https://github.com/secmon-lab/lycaon/actions/workflows/lint.yml/badge.svg)](https://github.com/secmon-lab/lycaon/actions/workflows/lint.yml)
[![Gosec](https://github.com/secmon-lab/lycaon/actions/workflows/gosec.yml/badge.svg)](https://github.com/secmon-lab/lycaon/actions/workflows/gosec.yml)
[![Trivy](https://github.com/secmon-lab/lycaon/actions/workflows/trivy.yml/badge.svg)](https://github.com/secmon-lab/lycaon/actions/workflows/trivy.yml)

Slack-based Incident Management Service

## Overview

Lycaon is an incident management service that integrates with Slack to help teams manage and respond to incidents efficiently. It provides automatic message processing, LLM-powered insights, and a web dashboard for incident tracking.

## Features

- **Slack Integration**: Receive and process messages from Slack channels
- **LLM Support**: Analyze incidents using Google Gemini AI
- **Web Dashboard**: View and manage incidents through a web interface
- **Session-based Authentication**: Secure OAuth2 authentication with Slack
- **Firestore Persistence**: Store incident messages in Google Firestore

## Installation

### Prerequisites

- Go 1.22 or later
- Node.js 20 or later
- Google Cloud Project (for Firestore and Gemini)
- Slack App with OAuth2 configured

### Build from source

```bash
# Clone the repository
git clone https://github.com/secmon-lab/lycaon.git
cd lycaon

# Install dependencies and build
task build

# Or build manually
cd frontend && npm install && npm run build && cd ..
go build -o lycaon
```

## Configuration

Lycaon is configured through environment variables:

```bash
# Server
LYCAON_ADDR=:8080
LYCAON_DEV=false

# Slack Configuration (Required)
LYCAON_SLACK_CLIENT_ID=your-slack-client-id
LYCAON_SLACK_CLIENT_SECRET=your-slack-client-secret
LYCAON_SLACK_SIGNING_SECRET=your-signing-secret
LYCAON_SLACK_OAUTH_TOKEN=xoxb-your-oauth-token

# Firestore Configuration (Optional)
LYCAON_FIRESTORE_PROJECT_ID=your-gcp-project
LYCAON_FIRESTORE_DATABASE_ID=(default)

# Gemini Configuration (Optional)
LYCAON_GEMINI_PROJECT=your-gcp-project
LYCAON_GEMINI_MODEL=gemini-1.5-flash

# Logging
LYCAON_LOG_LEVEL=info
```

## Slack App Setup

1. Create a new Slack App at https://api.slack.com/apps
2. Configure OAuth & Permissions:
   - Add redirect URL: `http://your-domain/api/auth/callback`
   - Required scopes:
     - `channels:history`
     - `channels:read`
     - `chat:write`
     - `users:read`
3. Configure Event Subscriptions:
   - Request URL: `http://your-domain/hooks/slack/events`
   - Subscribe to events:
     - `message.channels`
     - `app_mention`
4. Configure Interactivity:
   - Request URL: `http://your-domain/hooks/slack/interactions`

## Usage

### Running the server

```bash
# Start the server
./lycaon serve

# Or with environment file
source .env && ./lycaon serve
```

### Development

```bash
# Run development environment (frontend and backend)
./dev.sh

# Run tests
task test

# Generate mocks
task mock
```

The development script will:
- Start the backend server on port 8080
- Start the frontend development server on port 3000 with hot reload
- Create a sample `.env` file if it doesn't exist

## Architecture

Lycaon follows a clean architecture pattern:

- **Domain Layer**: Core business entities and interfaces
- **Repository Layer**: Data persistence (Firestore/Memory)
- **UseCase Layer**: Business logic
- **Controller Layer**: HTTP handlers and Slack integration
- **Frontend**: React-based web interface

## Development

### Project Structure

```
lycaon/
├── main.go              # Entry point
├── pkg/
│   ├── domain/         # Domain models and interfaces
│   ├── repository/     # Data persistence
│   ├── usecase/        # Business logic
│   ├── controller/     # HTTP and Slack handlers
│   └── cli/            # CLI commands
├── frontend/           # React frontend
│   ├── src/           # Source code
│   └── dist/          # Build output (embedded)
└── Taskfile.yml       # Task automation
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/repository/...
```

## License

Apache License 2.0

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and questions, please use the GitHub issue tracker.

# Lycaon [![Unit test](https://github.com/secmon-lab/lycaon/actions/workflows/test.yml/badge.svg)](https://github.com/secmon-lab/lycaon/actions/workflows/test.yml) [![Lint](https://github.com/secmon-lab/lycaon/actions/workflows/lint.yml/badge.svg)](https://github.com/secmon-lab/lycaon/actions/workflows/lint.yml) [![Gosec](https://github.com/secmon-lab/lycaon/actions/workflows/gosec.yml/badge.svg)](https://github.com/secmon-lab/lycaon/actions/workflows/gosec.yml) [![Trivy](https://github.com/secmon-lab/lycaon/actions/workflows/trivy.yml/badge.svg)](https://github.com/secmon-lab/lycaon/actions/workflows/trivy.yml)

<p align="center">
  <img src="./docs/images/logo_v0.png" height="128" />
</p>

Slack-based Incident Management Service

## Overview

Lycaon is an incident management service that integrates with Slack to help teams manage and respond to incidents efficiently. It provides automatic message processing, LLM-powered insights, and a web dashboard for incident tracking.

## Features

- **Slack Integration**: Receive and process messages from Slack channels
- **LLM Support**: Analyze incidents using Google Gemini AI
- **Web Dashboard**: View and manage incidents through a web interface
- **Session-based Authentication**: Secure OAuth2 authentication with Slack
- **Firestore Persistence**: Store incident messages in Google Firestore
- **Automatic Bookmarks**: Automatically add Web UI links to incident channels as bookmarks

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
# Server Configuration
LYCAON_ADDR=localhost:8080
LYCAON_FRONTEND_URL=http://localhost:8080  # Optional: enables automatic bookmark creation to incident Web UI

# Slack Configuration (Required)
LYCAON_SLACK_CLIENT_ID=your-slack-client-id
LYCAON_SLACK_CLIENT_SECRET=your-slack-client-secret
LYCAON_SLACK_SIGNING_SECRET=your-slack-signing-secret
LYCAON_SLACK_OAUTH_TOKEN=xoxb-your-oauth-token
LYCAON_SLACK_CHANNEL_PREFIX=inc

# Firestore Configuration (Optional)
LYCAON_FIRESTORE_PROJECT_ID=your-gcp-project
LYCAON_FIRESTORE_DATABASE_ID=(default)

# Gemini Configuration (Optional for LLM analysis)
LYCAON_GEMINI_PROJECT_ID=your-gcp-project
LYCAON_GEMINI_LOCATION=us-central1
LYCAON_GEMINI_MODEL=gemini-2.5-flash

# Logging Configuration
LYCAON_LOG_LEVEL=info
LYCAON_LOG_FORMAT=auto

# Incident Configuration (Optional)
LYCAON_CONFIG_PATH=./config/config.yaml
```

### Incident Configuration

Configure incident categories and severities in a single YAML file:

```yaml
# config/config.yaml
categories:
  - id: security_incident
    name: Security Incident
    description: Security-related incidents requiring immediate attention
    invite_users:
      - U01234567  # User ID
      - "@alice"   # Username
    invite_groups:
      - S01234567  # Group ID
      - "@security-team"  # Group handle

  - id: service_outage
    name: Service Outage
    description: Service availability issues
    invite_users:
      - "@bob"
      - "@charlie"
    invite_groups:
      - "@sre-team"

  - id: performance_issue
    name: Performance Issue
    description: Performance degradation or optimization needed

  - id: unknown
    name: Unknown
    description: Category not yet determined

severities:
  - id: critical
    name: Critical
    description: System down, major business impact
    level: 90

  - id: high
    name: High
    description: Significant degradation, urgent response needed
    level: 70

  - id: medium
    name: Medium
    description: Moderate impact, schedule fix
    level: 50

  - id: low
    name: Low
    description: Minor issue, low priority
    level: 30

  - id: info
    name: Info
    description: Informational, no action required
    level: 10

  - id: unknown
    name: Unknown
    description: Severity not yet determined
    level: -1
```

**Category Fields:**
- `id`: Unique identifier (use snake_case)
- `name`: Display name shown in UI
- `description`: Help text for selecting the category
- `invite_users`: List of user IDs or @usernames to automatically invite (optional)
- `invite_groups`: List of group IDs or @groupnames to automatically invite (optional)
- **Note**: The `unknown` category is required

**Severity Fields:**
- `id`: Unique identifier (use snake_case)
- `name`: Display name shown in UI
- `description`: Help text for selecting the severity
- `level`: Importance level (higher = more severe)
  - `90-99`: Critical - System down, immediate response required
  - `70-89`: High - Significant impact, urgent attention needed
  - `50-69`: Medium - Moderate impact, timely response needed
  - `30-49`: Low - Minor impact, can be scheduled
  - `10-29`: Info - Informational, minimal or no impact
  - `0`: Ignorable - No action required
  - `-1`: Unknown - Severity not yet determined (special case)

## Slack App Setup

1. Create a new Slack App at https://api.slack.com/apps
2. Configure OAuth & Permissions:
   - Add redirect URL: `http://your-domain/api/auth/callback`
   - Required Bot Token Scopes:
     - `app_mentions:read` - Receive app mention events
     - `bookmarks:write` - Create bookmarks in channels (for automatic Web UI links)
     - `channels:history` - Read message history from channels
     - `channels:manage` - Create and manage public channels
     - `channels:read` - Read channel information
     - `chat:write` - Send and update messages
     - `users:read` - Read user information
3. Configure Event Subscriptions:
   - Request URL: `http://your-domain/hooks/slack/event`
   - Subscribe to Bot Events:
     - `message.channels` - Listen to messages in public channels
     - `app_mention` - Listen to app mentions
4. Configure Interactivity & Shortcuts:
   - Request URL: `http://your-domain/hooks/slack/interaction`

## Usage

### Running the server

```bash
# Start the server
./lycaon serve

# Or with environment file
source .env && ./lycaon serve
```

## Architecture

Lycaon follows a clean architecture pattern:

- **Domain Layer**: Core business entities and interfaces
- **Repository Layer**: Data persistence (Firestore/Memory)
- **UseCase Layer**: Business logic
- **Controller Layer**: HTTP handlers and Slack integration
- **Frontend**: React-based web interface

## License

Apache License 2.0

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and questions, please use the GitHub issue tracker.

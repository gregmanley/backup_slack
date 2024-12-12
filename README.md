# Backup Slack

## Description

A Go utility to back up Slack channels to a local SQLite database. It allows you to archive channel messages and associated files from specified Slack channels, storing them locally for backup and archival purposes.

## Getting Started

### Prerequisites

- Go 1.23.2 or higher
- A Slack workspace where you have appropriate permissions
- A Slack API token with the following scopes:
  - `channels:history`
  - `channels:read`
  - `files:read`
  - `groups:history`
  - `groups:read`

### Setting up a Slack API Token

1. Go to [Slack API Apps page](https://api.slack.com/apps) and sign in
   - Note: Your account must have the necessary permissions to create apps in your workspace
2. Click "Create New App" and choose "From scratch"
3. Name your app and select your workspace
4. Under "OAuth & Permissions", go to "Scopes" and add these Bot Token Scopes:
   - `channels:history`
   - `channels:read`
   - `files:read`
   - `groups:history` (if you want the app to backup private channels that it has been added to)
   - `groups:read` (if you want the app to backup private channels that it has been added to)
5. Click "Install to Workspace" at the top of the page under "OAuth Tokens for Your Workspace"
6. After installation, copy the "Bot User OAuth Token" that starts with `xoxb-`
7. Add this token to your `.env` file as `SLACK_BOT_TOKEN`

Note: Keep your token secret

### Installation

#### Manual Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/gregmanley/backup_slack.git
   cd backup_slack
   ```

2. Install dependencies
   ```bash
   go mod tidy
   ```

3. Copy the example environment file
   ```bash
   cp .env.example .env
   ```

4. Configure the application by editing `.env`:
   ```
   SLACK_API_TOKEN=xoxb-your-token-here
   SLACK_CHANNELS=C12345678,C87654321
   LOG_LEVEL=INFO
   DB_PATH=./data/slack_backup.db
   STORAGE_PATH=./data/storage
   LOG_PATH=./data/logs/backup.log
   ```

#### Installaion Script
1. Build the binary
2. Run the installation script:
   ```bash
   sudo ./scripts/install.sh
   ```
3. Create workspace directories and .env files for each workspace:
   ```bash
   # Replace workspace1 with your workspace name
   sudo mkdir -p /opt/backup_slack/workspaces/workspace1
   sudo cp .env.example /opt/backup_slack/workspaces/workspace1/.env
   sudo nano /opt/backup_slack/workspaces/workspace1/.env  # Edit configuration
   ```
4. Start the service:
   ```bash
   sudo /opt/backup_slack/manage-services.sh start
   ```


### Building and Running

#### Using Make

Build the application:
```bash
make build
```

Run the application:
```bash
make run
```

Run tests:
```bash
make test
```

#### Manual Build/Run

Build Manually:
```bash
go build -o bin/backup_slack cmd/backup_slack/main.go
```

Run Manually:
```bash
./bin/backup_slack
```

### Project Structure

```
.
├── bin/           # Compiled binary output
├── cmd/           # Main application entry points
├── config/        # Configuration files
├── data/          # Generated data directory
│   ├── logs/      # Application logs
│   └── storage/   # Downloaded file storage
├── internal/      # Private application code
│   ├── config/    # Configuration handling
│   ├── database/  # Database operations
│   ├── files/     # File handling
│   ├── logger/    # Logging utilities
│   ├── service/   # Business logic
│   └── slack/     # Slack API integration
├── pkg/           # Public libraries
└── test/          # Additional test files
```

### Configuration

The application uses environment variables for configuration, which can be set in the .env file:
- SLACK_API_TOKEN: Your Slack Bot User OAuth Token
- SLACK_CHANNELS: Comma-separated list of Slack channel IDs to backup
- LOG_LEVEL: Logging level (DEBUG, INFO, WARN, ERROR)
- DB_PATH: Path to SQLite database file
- STORAGE_PATH: Directory path for storing downloaded files
- LOG_PATH: Path to log file


## Contributing

I'm not currently accepting contributions to this project, but please feel free to fork the repository and do whatever you want with it.


## License

This project is licensed under the MIT License - see the LICENSE file for details.

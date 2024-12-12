#!/bin/bash

# Create directory structure
sudo mkdir -p /opt/backup_slack/{bin,workspaces}

# Copy binary
sudo cp ./bin/backup_slack /opt/backup_slack/bin/
sudo chmod +x /opt/backup_slack/bin/backup_slack

# Copy service management script
sudo cp ./scripts/manage-services.sh /opt/backup_slack/
sudo chmod +x /opt/backup_slack/manage-services.sh

# Create service user
sudo useradd -r -s /bin/false backup-slack
sudo chown -R backup-slack:backup-slack /opt/backup_slack

# Install systemd service template
sudo cp ./scripts/systemd/backup-slack@.service /etc/systemd/system/
sudo systemctl daemon-reload

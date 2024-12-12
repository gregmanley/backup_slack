#!/bin/bash

case "$1" in
    start)
        for workspace in /opt/backup_slack/workspaces/*; do
            name=$(basename "$workspace")
            sudo systemctl start backup-slack@"$name"
        done
        ;;
    stop)
        for workspace in /opt/backup_slack/workspaces/*; do
            name=$(basename "$workspace")
            sudo systemctl stop backup-slack@"$name"
        done
        ;;
    status)
        for workspace in /opt/backup_slack/workspaces/*; do
            name=$(basename "$workspace")
            echo "=== $name ==="
            sudo systemctl status backup-slack@"$name" --no-pager
            echo
        done
        ;;
    *)
        echo "Usage: $0 {start|stop|status}"
        exit 1
        ;;
esac
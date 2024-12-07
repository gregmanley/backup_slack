package service

import (
	"backup_slack/internal/database"
	"backup_slack/internal/logger"
	"database/sql"
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

// CollectMessages fetches and stores messages for the specified channel
func (s *SlackService) CollectMessages(channelID string) error {
	lastTimestamp, err := s.db.GetLastMessageTimestamp(channelID)
	if err != nil {
		return fmt.Errorf("failed to get last message timestamp: %w", err)
	}

	// Format as Unix timestamp with microseconds
	oldest := fmt.Sprintf("%.6f", float64(lastTimestamp.Unix()))
	nextCursor := ""

	for {
		messages, cursor, err := s.client.GetChannelMessages(channelID, oldest, nextCursor)
		if err != nil {
			return fmt.Errorf("failed to fetch messages: %w", err)
		}

		if err := s.processMessages(channelID, messages); err != nil {
			return fmt.Errorf("failed to process messages: %w", err)
		}

		if cursor == "" {
			break
		}
		nextCursor = cursor
	}

	logger.Info.Printf("Finished collecting messages for channel %s", channelID)
	return nil
}

func (s *SlackService) processMessages(channelID string, messages []slack.Message) error {
	for _, msg := range messages {
		dbMsg := database.Message{
			ID:        msg.Timestamp, // Slack uses timestamps as message IDs
			ChannelID: channelID,
			UserID:    msg.User,
			Content:   msg.Text,
			Timestamp: convertSlackTimestamp(msg.Timestamp),
			ThreadTS: sql.NullString{
				String: msg.ThreadTimestamp,
				Valid:  msg.ThreadTimestamp != "",
			},
			MessageType: "message", // Default type
		}

		if msg.Edited != nil {
			dbMsg.LastEdited = sql.NullTime{
				Time:  convertSlackTimestamp(msg.Edited.Timestamp),
				Valid: true,
			}
		}

		if err := s.db.InsertMessage(dbMsg); err != nil {
			return fmt.Errorf("failed to store message: %w", err)
		}

		// If message is part of a thread, fetch replies
		if msg.ThreadTimestamp != "" && msg.ThreadTimestamp == msg.Timestamp {
			if err := s.collectThreadReplies(channelID, msg.ThreadTimestamp); err != nil {
				logger.Error.Printf("Failed to collect thread replies: %v", err)
				continue
			}
		}
	}

	return nil
}

func (s *SlackService) collectThreadReplies(channelID, threadTS string) error {
	replies, err := s.client.GetMessageReplies(channelID, threadTS)
	if err != nil {
		return fmt.Errorf("failed to fetch thread replies: %w", err)
	}

	// Skip first message as it's the parent
	if len(replies) > 1 {
		return s.processMessages(channelID, replies[1:])
	}

	return nil
}

func convertSlackTimestamp(ts string) time.Time {
	sec := int64(float64(0))
	fmt.Sscanf(ts, "%d", &sec)
	return time.Unix(sec, 0)
}

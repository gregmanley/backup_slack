package service

import (
	"fmt"
	"time"

	"backup_slack/internal/database"
	"backup_slack/internal/logger"
	"database/sql"

	"github.com/slack-go/slack"
)

// CollectMessages fetches and stores messages for the specified channel
func (s *SlackService) CollectMessages(channelID string) (int, error) {
	var (
		latest        string // Will hold the oldest timestamp from previous batch
		totalMessages = 0
		seenMessages  = make(map[string]bool)
	)

	for {
		// Use latest as timestamp cursor to get next older batch of messages
		messages, _, err := s.client.GetChannelMessages(channelID, latest, "")
		if err != nil {
			return totalMessages, fmt.Errorf("failed to fetch messages: %w", err)
		}

		logger.Debug.Printf("Retrieved %d messages for channel %s", len(messages), channelID)

		if len(messages) == 0 {
			logger.Debug.Printf("No more messages to fetch for channel %s", channelID)
			break
		}

		// Find oldest message timestamp in this batch to use as cursor for next batch
		oldest := messages[0].Timestamp
		for _, msg := range messages {
			if msg.Timestamp < oldest {
				oldest = msg.Timestamp
			}
		}
		latest = oldest // Set latest to oldest message timestamp for next iteration

		// Check for messages we've already processed
		var newMessages []slack.Message
		for _, msg := range messages {
			if exists, err := s.db.MessageExists(msg.Timestamp); err != nil {
				return totalMessages, fmt.Errorf("failed to check message existence: %w", err)
			} else if exists {
				logger.Debug.Printf("Found existing message (ts: %s), continuing to older messages", msg.Timestamp)
				continue
			}

			if !seenMessages[msg.Timestamp] {
				newMessages = append(newMessages, msg)
				seenMessages[msg.Timestamp] = true
			}
		}

		if len(newMessages) > 0 {
			if err := s.processMessages(channelID, newMessages); err != nil {
				return totalMessages, fmt.Errorf("failed to process messages: %w", err)
			}
			totalMessages += len(newMessages)
			logger.Debug.Printf("Processed %d new messages, total so far: %d", len(newMessages), totalMessages)
		}

		// Add a small delay to prevent hitting rate limits too aggressively
		time.Sleep(time.Millisecond * 100)
	}

	return totalMessages, nil
}

func (s *SlackService) processMessages(channelID string, messages []slack.Message) error {
	// First, collect and store all unique users
	users := make(map[string]struct{})
	for _, msg := range messages {
		if msg.User != "" {
			users[msg.User] = struct{}{}
		}
	}

	logger.Debug.Printf("Channel %s: Found %d unique users in messages", channelID, len(users))

	// Store users before processing messages
	if err := s.storeUsers(users); err != nil {
		return fmt.Errorf("failed to store users: %w", err)
	}

	// Now process messages
	for i, msg := range messages {
		logger.Debug.Printf("Channel %s: Processing message %d/%d (ts: %s, user: %s)",
			channelID, i+1, len(messages), msg.Timestamp, msg.User)

		// Handle bot messages or messages without user IDs
		if msg.User == "" {
			if msg.BotID != "" {
				msg.User = msg.BotID
				logger.Debug.Printf("Using bot ID %s as user ID for message", msg.BotID)
			} else {
				msg.User = "UNKNOWN"
				logger.Debug.Printf("No user ID found for message, using UNKNOWN")
			}
			// Store the bot/unknown user
			if err := s.db.InsertUser(database.User{
				ID:        msg.User,
				Username:  msg.User,
				FirstSeen: time.Now(),
			}); err != nil {
				logger.Error.Printf("Failed to store system user %s: %v", msg.User, err)
				return fmt.Errorf("failed to store system user: %w", err)
			}
		}

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
			return fmt.Errorf("failed to store message (ts: %s, user: %s): %w",
				msg.Timestamp, msg.User, err)
		}

		// If message is part of a thread, fetch replies
		if msg.ThreadTimestamp != "" && msg.ThreadTimestamp == msg.Timestamp {
			if err := s.collectThreadReplies(channelID, msg.ThreadTimestamp); err != nil {
				logger.Error.Printf("Failed to collect thread replies: %v", err)
				continue
			}
		}

		// Process files attached to messages
		if len(msg.Files) > 0 {
			for _, file := range msg.Files {
				dbFile := database.File{
					ID:              file.ID,
					MessageID:       msg.Timestamp,
					OriginalURL:     file.URLPrivateDownload,
					LocalPath:       s.fileService.storage.GenerateFilePath(channelID, file.ID, file.Filetype, convertSlackTimestamp(msg.Timestamp)),
					FileName:        file.Name,
					FileType:        file.Filetype,
					SizeBytes:       int64(file.Size),
					UploadTimestamp: convertSlackTimestamp(msg.Timestamp),
					Checksum:        "", // Will be set after download
				}

				if err := s.fileService.ProcessFile(dbFile); err != nil {
					logger.Error.Printf("Failed to process file %s: %v", file.ID, err)
					continue
				}
			}
		}
	}

	return nil
}

func (s *SlackService) storeUsers(users map[string]struct{}) error {
	for userID := range users {
		err := s.db.InsertUser(database.User{
			ID:        userID,
			Username:  userID, // We'll just use ID as username initially
			FirstSeen: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to store user %s: %w", userID, err)
		}
		logger.Debug.Printf("Stored user: %s", userID)
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
	logger.Debug.Printf("Converting Slack timestamp: %s", ts)
	sec := int64(0)
	fmt.Sscanf(ts, "%d", &sec)
	converted := time.Unix(sec, 0)
	logger.Debug.Printf("Converted to time.Time: %v (Unix: %d)", converted, converted.Unix())
	return converted
}

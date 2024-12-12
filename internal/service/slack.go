package service

import (
	"fmt"
	"time"

	"backup_slack/internal/database"
	"backup_slack/internal/logger"
	"backup_slack/internal/slack"

	slackapi "github.com/slack-go/slack"
)

type SlackService struct {
	client      *slack.Client
	db          *database.DB
	fileService *FileService
}

func NewSlackService(token string, db *database.DB, storagePath string) (*SlackService, error) {
	fileService, err := NewFileService(storagePath, 1024*1024*1024, db, token) // 1GB max file size
	if err != nil {
		return nil, fmt.Errorf("failed to create file service: %w", err)
	}

	service := &SlackService{
		client:      slack.NewClient(token),
		db:          db,
		fileService: fileService,
	}

	return service, nil
}

// Initialize validates authentication and ensures we can access specified channels
func (s *SlackService) Initialize(targetChannelIDs []string) error {
	// Validate authentication
	auth, err := s.client.ValidateAuth()
	if err != nil {
		return fmt.Errorf("failed to validate auth: %w", err)
	}
	logger.Info.Printf("Authenticated as %s (team: %s)", auth.User, auth.Team)

	// Get available channels
	channels, err := s.client.GetChannels()
	if err != nil {
		return fmt.Errorf("failed to get channels: %w", err)
	}

	// Create ID->Channel map for validation
	channelMap := make(map[string]slackapi.Channel)
	for _, ch := range channels {
		channelMap[ch.ID] = ch
	}

	// Store channels in database
	for _, id := range targetChannelIDs {
		ch, exists := channelMap[id]
		if !exists {
			return fmt.Errorf("channel %s not found or not accessible", id)
		}

		dbChannel := database.Channel{
			ID:   ch.ID,
			Name: ch.Name,
			ChannelType: func() string {
				if ch.IsPrivate {
					return "private_channel"
				}
				return "public_channel"
			}(),
			IsArchived: ch.IsArchived,
			CreatedAt:  time.Unix(int64(ch.Created), 0),
			Topic:      ch.Topic.Value,
			Purpose:    ch.Purpose.Value,
		}

		if err := s.db.InsertChannel(dbChannel); err != nil {
			return fmt.Errorf("failed to store channel %s: %w", id, err)
		}
	}

	return nil
}

func (s *SlackService) storeChannels(channels []slackapi.Channel) error {
	logger.Debug.Printf("Storing %d channels", len(channels))

	for _, ch := range channels {
		createdAt := time.Unix(int64(ch.Created), 0)

		// Map channel type
		channelType := "public_channel"
		if ch.IsPrivate {
			channelType = "private_channel"
		}

		logger.Debug.Printf("Storing channel: ID=%s, Name=%s, Type=%s, Private=%v",
			ch.ID, ch.Name, channelType, ch.IsPrivate)

		err := s.db.InsertChannel(database.Channel{
			ID:          ch.ID,
			Name:        ch.Name,
			ChannelType: channelType,
			IsArchived:  ch.IsArchived,
			CreatedAt:   createdAt,
			Topic:       ch.Topic.Value,
			Purpose:     ch.Purpose.Value,
		})
		if err != nil {
			logger.Error.Printf("Failed to insert channel: %+v", ch)
			return fmt.Errorf("failed to insert channel %s: %w", ch.Name, err)
		}
		logger.Debug.Printf("Successfully stored channel %s", ch.Name)
	}
	return nil
}

// BackupChannelMessages initiates message collection for a channel
func (s *SlackService) BackupChannelMessages(channelID string) error {
	logger.Info.Printf("Starting message backup for channel %s", channelID)

	messageCount, err := s.CollectMessages(channelID)
	if err != nil {
		return fmt.Errorf("failed to backup messages for channel %s: %w", channelID, err)
	}

	logger.Info.Printf("Backed up %d messages for channel %s", messageCount, channelID)
	return nil
}

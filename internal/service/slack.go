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
	client *slack.Client
	db     *database.DB
}

func NewSlackService(token string, db *database.DB) *SlackService {
	return &SlackService{
		client: slack.NewClient(token),
		db:     db,
	}
}

// Initialize validates authentication and ensures we can access specified channels
func (s *SlackService) Initialize(targetChannelIDs []string) error {
	// Validate authentication
	auth, err := s.client.ValidateAuth()
	if err != nil {
		return fmt.Errorf("auth validation failed: %w", err)
	}
	logger.Info.Printf("Authenticated as %s (team: %s)", auth.User, auth.Team)

	// Get available channels
	channels, err := s.client.GetChannels()
	if err != nil {
		return fmt.Errorf("failed to list channels: %w", err)
	}

	// Create ID->Channel map for validation
	channelMap := make(map[string]slackapi.Channel)
	for _, ch := range channels {
		channelMap[ch.ID] = ch
	}

	// Verify access to target channels and build filtered list
	var targetChannelData []slackapi.Channel
	for _, targetID := range targetChannelIDs {
		if ch, exists := channelMap[targetID]; exists {
			targetChannelData = append(targetChannelData, ch)
			logger.Info.Printf("Will backup channel: %s (ID: %s)", ch.Name, ch.ID)
		} else {
			return fmt.Errorf("no access to channel ID: %s", targetID)
		}
	}

	// Store only target channels in database
	if err := s.storeChannels(targetChannelData); err != nil {
		return fmt.Errorf("failed to store channels: %w", err)
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

	if err := s.CollectMessages(channelID); err != nil {
		return fmt.Errorf("failed to backup messages for channel %s: %w", channelID, err)
	}

	return nil
}

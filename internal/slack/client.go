package slack

import (
	"context"
	"fmt"
	"time"

	"backup_slack/internal/logger"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

type Client struct {
	api         *slack.Client
	rateLimiter *rate.Limiter
	ctx         context.Context
}

func NewClient(token string) *Client {
	// Create rate limiter: 100 requests per minute as per requirements
	limiter := rate.NewLimiter(rate.Every(time.Minute/100), 100)

	return &Client{
		api:         slack.New(token),
		rateLimiter: limiter,
		ctx:         context.Background(),
	}
}

// GetChannels returns all channels the bot has access to
func (c *Client) GetChannels() ([]slack.Channel, error) {
	if err := c.rateLimiter.Wait(c.ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	channels, _, err := c.api.GetConversations(&slack.GetConversationsParameters{
		Types: []string{"public_channel", "private_channel"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get channels: %w", err)
	}

	return channels, nil
}

// ValidateAuth checks if the token is valid and returns basic auth info
func (c *Client) ValidateAuth() (*slack.AuthTestResponse, error) {
	if err := c.rateLimiter.Wait(c.ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	resp, err := c.api.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("auth validation failed: %w", err)
	}

	return resp, nil
}

// GetChannelMessages fetches messages from a channel since a specific timestamp
func (c *Client) GetChannelMessages(channelID string, latest string, cursor string) ([]slack.Message, string, error) {
	if err := c.rateLimiter.Wait(c.ctx); err != nil {
		return nil, "", fmt.Errorf("rate limiter error: %w", err)
	}

	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     200, // Increased from 100 to 200 for better performance
		Cursor:    cursor,
		Latest:    latest, // If empty, will get most recent messages
		Inclusive: false,
	}

	logger.Debug.Printf("Fetching messages: channel=%s cursor=%s latest=%s",
		channelID, cursor, latest)

	resp, err := c.api.GetConversationHistory(params)
	if err != nil {
		logger.Error.Printf("Slack API error for channel %s: %v", channelID, err)
		return nil, "", fmt.Errorf("failed to get channel history: %w", err)
	}

	logger.Debug.Printf("Successfully retrieved %d messages from Slack API", len(resp.Messages))
	nextCursor := resp.ResponseMetadata.Cursor
	if resp.HasMore {
		logger.Debug.Printf("More messages available, next cursor: %s", nextCursor)
	}

	return resp.Messages, nextCursor, nil
}

// GetMessageReplies fetches all replies in a thread
func (c *Client) GetMessageReplies(channelID, threadTS string) ([]slack.Message, error) {
	if err := c.rateLimiter.Wait(c.ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	params := &slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: threadTS,
	}

	messages, _, _, err := c.api.GetConversationReplies(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread replies: %w", err)
	}

	return messages, nil
}

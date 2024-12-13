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
	limiter := rate.NewLimiter(rate.Every(time.Minute/50), 50)

	return &Client{
		api:         slack.New(token),
		rateLimiter: limiter,
		ctx:         context.Background(),
	}
}

func (c *Client) retryWithBackoff(f func() error) error {
	maxRetries := 5
	baseDelay := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := c.rateLimiter.Wait(c.ctx); err != nil {
			return fmt.Errorf("rate limiter error: %w", err)
		}

		err := f()
		if err == nil {
			return nil
		}

		// Handle rate limits
		if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
			delay := time.Duration(rateLimitErr.RetryAfter) * time.Second
			logger.Debug.Printf("Rate limited, waiting %v before retry", delay)
			time.Sleep(delay)
			continue
		}

		// Exponential backoff for other errors
		delay := baseDelay * time.Duration(1<<uint(attempt))
		time.Sleep(delay)
	}
	return fmt.Errorf("failed after %d retries", maxRetries)
}

// GetChannels returns all channels the bot has access to
func (c *Client) GetChannels() ([]slack.Channel, error) {
	var channels []slack.Channel
	err := c.retryWithBackoff(func() error {
		var err error
		channels, _, err = c.api.GetConversations(&slack.GetConversationsParameters{
			Types: []string{"public_channel", "private_channel"},
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get channels: %w", err)
	}
	return channels, nil
}

// ValidateAuth checks if the token is valid and returns basic auth info
func (c *Client) ValidateAuth() (*slack.AuthTestResponse, error) {
	var resp *slack.AuthTestResponse
	err := c.retryWithBackoff(func() error {
		var err error
		resp, err = c.api.AuthTest()
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("auth validation failed: %w", err)
	}
	return resp, nil
}

// GetChannelMessages fetches messages from a channel since a specific timestamp
func (c *Client) GetChannelMessages(channelID string, latest string, cursor string) ([]slack.Message, string, error) {
	var messages []slack.Message
	var nextCursor string
	err := c.retryWithBackoff(func() error {
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
			return fmt.Errorf("failed to get channel history: %w", err)
		}

		logger.Debug.Printf("Successfully retrieved %d messages from Slack API", len(resp.Messages))
		messages = resp.Messages
		nextCursor = resp.ResponseMetadata.Cursor
		if resp.HasMore {
			logger.Debug.Printf("More messages available, next cursor: %s", nextCursor)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return messages, nextCursor, nil
}

// GetMessageReplies fetches all replies in a thread
func (c *Client) GetMessageReplies(channelID, threadTS string) ([]slack.Message, error) {
	var messages []slack.Message
	err := c.retryWithBackoff(func() error {
		params := &slack.GetConversationRepliesParameters{
			ChannelID: channelID,
			Timestamp: threadTS,
		}
		var err error
		messages, _, _, err = c.api.GetConversationReplies(params)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get thread replies: %w", err)
	}
	return messages, nil
}

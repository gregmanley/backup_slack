package slack

import (
	"context"
	"fmt"
	"time"

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

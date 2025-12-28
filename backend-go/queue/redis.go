package queue

import (
	"context"
	"encoding/json"
	"os"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Client wraps Redis operations for job queue
type Client struct {
	rdb *redis.Client
}

// JobPayload represents a job to be processed
type JobPayload struct {
	JobID       string `json:"job_id"`
	InvestorID  string `json:"investor_id,omitempty"`
	DeckContent string `json:"deck_content,omitempty"`
	DeckPath    string `json:"deck_path,omitempty"`
}

const jobQueue = "sago:jobs"

// NewClient creates a new Redis queue client
func NewClient() (*Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6380"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opts)

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{rdb: rdb}, nil
}

// EnqueueJob adds a job to the queue
func (c *Client) EnqueueJob(ctx context.Context, jobID uuid.UUID, investorID *uuid.UUID, deckContent string, deckPath string) error {
	payload := JobPayload{
		JobID:       jobID.String(),
		DeckContent: deckContent,
		DeckPath:    deckPath,
	}

	if investorID != nil {
		payload.InvestorID = investorID.String()
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return c.rdb.RPush(ctx, jobQueue, data).Err()
}

// GetQueueLength returns the number of jobs in queue
func (c *Client) GetQueueLength(ctx context.Context) (int64, error) {
	return c.rdb.LLen(ctx, jobQueue).Result()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Ping checks if Redis is available
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

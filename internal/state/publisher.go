package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// StateChangeChannel is the Redis pub/sub channel for state changes
	StateChangeChannel = "workflow:state_changes"
)

// RedisPublisher publishes state change events to Redis pub/sub
type RedisPublisher struct {
	client *redis.Client
}

// NewRedisPublisher creates a new Redis event publisher
func NewRedisPublisher(client *redis.Client) *RedisPublisher {
	return &RedisPublisher{
		client: client,
	}
}

// Publish publishes a state transition event to Redis
func (p *RedisPublisher) Publish(event TransitionEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to Redis channel
	if err := p.client.Publish(ctx, StateChangeChannel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish to Redis: %w", err)
	}

	return nil
}

// Subscribe subscribes to state change events
func (p *RedisPublisher) Subscribe(ctx context.Context, handler func(TransitionEvent) error) error {
	pubsub := p.client.Subscribe(ctx, StateChangeChannel)
	defer pubsub.Close()

	// Wait for subscription confirmation
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Listen for messages
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-ch:
			var event TransitionEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				// Log error but continue processing
				continue
			}

			if err := handler(event); err != nil {
				// Log error but continue processing
				continue
			}
		}
	}
}

// MultiPublisher publishes to multiple publishers
type MultiPublisher struct {
	publishers []EventPublisher
}

// NewMultiPublisher creates a publisher that publishes to multiple publishers
func NewMultiPublisher(publishers ...EventPublisher) *MultiPublisher {
	return &MultiPublisher{
		publishers: publishers,
	}
}

// Publish publishes to all publishers
func (p *MultiPublisher) Publish(event TransitionEvent) error {
	for _, publisher := range p.publishers {
		if err := publisher.Publish(event); err != nil {
			// Continue publishing to other publishers even if one fails
			// In production, you might want to log this error
			continue
		}
	}
	return nil
}

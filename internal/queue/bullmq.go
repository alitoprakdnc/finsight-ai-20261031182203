package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v9"
	"go.uber.org/zap"

	"finSight/internal/config"
	"finSight/internal/util/logger"
)

// BullMQ implements a simple Redis‑based queue compatible with the
// fallback strategy when Google Cloud Tasks is unavailable.
// It uses a Redis list (LPUSH / RPOP) to store JSON‑encoded task payloads.
// The consumer blocks on BLPOP with a configurable timeout and processes
// tasks concurrently respecting a concurrency limit.
type BullMQ struct {
	redisClient *redis.Client
	queueName   string
	log         *zap.Logger
}

// NewBullMQ creates a new BullMQ instance using the provided configuration.
// It expects the Redis connection details to be present in the Config struct.
func NewBullMQ(cfg *config.Config) (*BullMQ, error) {
	if cfg.RedisHost == "" {
		return nil, errors.New("redis host not configured")
	}
	addr := fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)
	options := &redis.Options{
		Addr:     addr,
		Username: cfg.RedisUser,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}
	client := redis.NewClient(options)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	b := &BullMQ{
		redisClient: client,
		queueName:   cfg.BullMQQueueName,
		log:         logger.New(),
	}
	if b.queueName == "" {
		b.queueName = "finSight:tasks"
	}
	return b, nil
}

// Enqueue pushes a task onto the Redis list.
// The task payload is marshaled to JSON before being stored.
func (b *BullMQ) Enqueue(ctx context.Context, payload *TaskPayload) error {
	if payload == nil {
		return errors.New("nil task payload")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}
	// Use RPUSH to maintain FIFO order.
	if err := b.redisClient.RPush(ctx, b.queueName, data).Err(); err != nil {
		return fmt.Errorf("redis rpush failed: %w", err)
	}
	b.log.Debug("task enqueued", zap.String("queue", b.queueName), zap.Any("payload", payload))
	return nil
}

// Consumer encapsulates the logic for pulling tasks from the queue
// and invoking a user‑provided handler.
type Consumer struct {
	bullmq      *BullMQ
	handler     func(context.Context, *TaskPayload) error
	concurrency int
	// internal
	wg        sync.WaitGroup
	sem       chan struct{}
	shutdown  chan struct{}
	closeOnce sync.Once
}

// NewConsumer creates a new Consumer.
// `handler` is called for each dequeued task.
// `concurrency` limits the number of tasks processed in parallel.
func NewConsumer(b *BullMQ, handler func(context.Context, *TaskPayload) error, concurrency int) (*Consumer, error) {
	if b == nil {
		return nil, errors.New("bullmq instance is nil")
	}
	if handler == nil {
		return nil, errors.New("handler function is nil")
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	c := &Consumer{
		bullmq:      b,
		handler:     handler,
		concurrency: concurrency,
		sem:         make(chan struct{}, concurrency),
		shutdown:    make(chan struct{}),
	}
	return c, nil
}

// Start begins pulling tasks from Redis and processing them.
// It blocks until the supplied context is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	c.bullmq.log.Info("starting BullMQ consumer", zap.String("queue", c.bullmq.queueName), zap.Int("concurrency", c.concurrency))

	// Run a goroutine that watches for context cancellation.
	go func() {
		<-ctx.Done()
		c.shutdownConsumer()
	}()

	for {
		// Check if shutdown was requested.
		select {
		case <-c.shutdown:
			c.wg.Wait()
			c.bullmq.log.Info("consumer shutdown complete")
			return nil
		default:
		}

		// Block on BLPOP with a timeout to allow graceful shutdown checks.
		result, err := c.bullmq.redisClient.BLPop(context.Background(), 5*time.Second, c.bullmq.queueName).Result()
		if err != nil {
			if err == redis.Nil {
				// Timeout – loop again to check shutdown.
				continue
			}
			// Log unexpected errors but continue processing.
			c.bullmq.log.Error("BLPOP error", zap.Error(err))
			continue
		}
		if len(result) != 2 {
			c.bullmq.log.Warn("unexpected BLPOP result length", zap.Any("result", result))
			continue
		}
		raw := result[1]

		var payload TaskPayload
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			c.bullmq.log.Error("failed to unmarshal task payload", zap.String("raw", raw), zap.Error(err))
			// Discard malformed task and continue.
			continue
		}

		// Acquire semaphore slot.
		c.sem <- struct{}{}
		c.wg.Add(1)
		go c.processTask(context.Background(), &payload)
	}
}

// processTask runs the user handler and ensures accounting for the waitgroup and semaphore.
func (c *Consumer) processTask(ctx context.Context, payload *TaskPayload) {
	defer func() {
		<-c.sem
		c.wg.Done()
	}()

	// Create a derived context with a timeout to avoid runaway tasks.
	taskCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := c.handler(taskCtx, payload); err != nil {
		c.bullmq.log.Error("task handler error", zap.Any("payload", payload), zap.Error(err))
		// In a production system we might re‑queue or move to a dead‑letter queue.
	}
}

// shutdownConsumer signals the consumer loop to stop and waits for in‑flight tasks.
func (c *Consumer) shutdownConsumer() {
	c.closeOnce.Do(func() {
		close(c.shutdown)
	})
}

// Close gracefully stops the consumer and releases resources.
// It can be called directly if you are not using a context for cancellation.
func (c *Consumer) Close() error {
	c.shutdownConsumer()
	c.wg.Wait()
	if err := c.bullmq.redisClient.Close(); err != nil {
		return fmt.Errorf("failed to close redis client: %w", err)
	}
	c.bullmq.log.Info("BullMQ consumer closed")
	return nil
}
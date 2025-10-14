package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisClient(redisURL string) (*RedisClient, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	// Connection pool configuration for high concurrency
	opts.PoolSize = 100
	opts.MinIdleConns = 10
	opts.MaxRetries = 3
	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second
	opts.PoolTimeout = 4 * time.Second

	client := redis.NewClient(opts)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Storage provides high-level storage operations
type Storage struct {
	redis *RedisClient
}

func NewStorage(redis *RedisClient) *Storage {
	return &Storage{redis: redis}
}

// API Key operations
type APIKey struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Usage struct {
	ID               string    `json:"id"`
	StartDate        string    `json:"start_date"`
	EndDate          string    `json:"end_date"`
	TotalAllowance   float64   `json:"total_allowance"`
	OrgTotalUsed     float64   `json:"org_total_used"`
	Remaining        float64   `json:"remaining"`
	UsedRatio        float64   `json:"used_ratio"`
	LastUpdated      time.Time `json:"last_updated"`
	Error            string    `json:"error,omitempty"`
}

// SaveAPIKey stores an API key
func (s *Storage) SaveAPIKey(key *APIKey) error {
	ctx := context.Background()
	pipe := s.redis.client.Pipeline()

	// Save key data
	keyData, err := json.Marshal(key)
	if err != nil {
		return err
	}

	pipe.HSet(ctx, fmt.Sprintf("key:%s", key.ID), "data", keyData)
	pipe.SAdd(ctx, "keys:list", key.ID)

	_, err = pipe.Exec(ctx)
	return err
}

// GetAPIKey retrieves an API key
func (s *Storage) GetAPIKey(id string) (*APIKey, error) {
	ctx := context.Background()
	data, err := s.redis.client.HGet(ctx, fmt.Sprintf("key:%s", id), "data").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var key APIKey
	if err := json.Unmarshal([]byte(data), &key); err != nil {
		return nil, err
	}

	return &key, nil
}

// GetAllAPIKeys retrieves all API keys
func (s *Storage) GetAllAPIKeys() ([]*APIKey, error) {
	ctx := context.Background()
	
	// Get all key IDs
	ids, err := s.redis.client.SMembers(ctx, "keys:list").Result()
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return []*APIKey{}, nil
	}

	// Use pipeline to fetch all keys
	pipe := s.redis.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(ids))

	for i, id := range ids {
		cmds[i] = pipe.HGet(ctx, fmt.Sprintf("key:%s", id), "data")
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	keys := make([]*APIKey, 0, len(ids))
	for _, cmd := range cmds {
		data, err := cmd.Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, err
		}

		var key APIKey
		if err := json.Unmarshal([]byte(data), &key); err != nil {
			continue
		}
		keys = append(keys, &key)
	}

	return keys, nil
}

// DeleteAPIKey removes an API key
func (s *Storage) DeleteAPIKey(id string) error {
	ctx := context.Background()
	pipe := s.redis.client.Pipeline()

	pipe.Del(ctx, fmt.Sprintf("key:%s", id))
	pipe.Del(ctx, fmt.Sprintf("key:%s:usage", id))
	pipe.SRem(ctx, "keys:list", id)

	_, err := pipe.Exec(ctx)
	return err
}

// BatchDeleteAPIKeys removes multiple API keys
func (s *Storage) BatchDeleteAPIKeys(ids []string) (int, int) {
	success := 0
	failed := 0

	// Use pipeline for batch deletion
	ctx := context.Background()
	pipe := s.redis.client.Pipeline()

	for _, id := range ids {
		pipe.Del(ctx, fmt.Sprintf("key:%s", id))
		pipe.Del(ctx, fmt.Sprintf("key:%s:usage", id))
		pipe.SRem(ctx, "keys:list", id)
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		failed = len(ids)
		return success, failed
	}

	// Count successes
	for i := 0; i < len(ids); i++ {
		if i*3 < len(cmds) && cmds[i*3].Err() == nil {
			success++
		} else {
			failed++
		}
	}

	return success, failed
}

// SaveUsage stores usage data with cache
func (s *Storage) SaveUsage(usage *Usage, ttl time.Duration) error {
	ctx := context.Background()
	data, err := json.Marshal(usage)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("key:%s:usage", usage.ID)
	return s.redis.client.Set(ctx, key, data, ttl).Err()
}

// GetUsage retrieves cached usage data
func (s *Storage) GetUsage(id string) (*Usage, error) {
	ctx := context.Background()
	key := fmt.Sprintf("key:%s:usage", id)
	
	data, err := s.redis.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var usage Usage
	if err := json.Unmarshal([]byte(data), &usage); err != nil {
		return nil, err
	}

	return &usage, nil
}

// BatchSaveUsage saves multiple usage records using pipeline
func (s *Storage) BatchSaveUsage(usages []*Usage, ttl time.Duration) error {
	ctx := context.Background()
	pipe := s.redis.client.Pipeline()

	for _, usage := range usages {
		data, err := json.Marshal(usage)
		if err != nil {
			continue
		}
		key := fmt.Sprintf("key:%s:usage", usage.ID)
		pipe.Set(ctx, key, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Session operations
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Storage) SaveSession(session *Session, ttl time.Duration) error {
	ctx := context.Background()
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("session:%s", session.ID)
	return s.redis.client.Set(ctx, key, data, ttl).Err()
}

func (s *Storage) GetSession(id string) (*Session, error) {
	ctx := context.Background()
	key := fmt.Sprintf("session:%s", id)
	
	data, err := s.redis.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *Storage) DeleteSession(id string) error {
	ctx := context.Background()
	key := fmt.Sprintf("session:%s", id)
	return s.redis.client.Del(ctx, key).Err()
}

// Metrics operations
func (s *Storage) IncrementMetric(metric string) error {
	ctx := context.Background()
	key := fmt.Sprintf("metrics:%s", metric)
	return s.redis.client.Incr(ctx, key).Err()
}

func (s *Storage) GetMetric(metric string) (int64, error) {
	ctx := context.Background()
	key := fmt.Sprintf("metrics:%s", metric)
	
	val, err := s.redis.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	
	return val, nil
}

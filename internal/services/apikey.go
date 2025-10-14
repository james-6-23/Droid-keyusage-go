package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/droid-keyusage-go/internal/models"
	"github.com/droid-keyusage-go/internal/storage"
	"github.com/google/uuid"
)

// APIKeyService handles API key operations
type APIKeyService struct {
	store       *storage.Storage
	workerPool  *WorkerPool
	localCache  *bigcache.BigCache
	cacheTTL    time.Duration
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(store *storage.Storage, workerPool *WorkerPool) *APIKeyService {
	// Configure local cache
	config := bigcache.DefaultConfig(5 * time.Minute)
	config.Shards = 16
	config.MaxEntriesInWindow = 10000
	config.MaxEntrySize = 500
	config.Verbose = false
	
	cache, _ := bigcache.New(context.Background(), config)

	return &APIKeyService{
		store:      store,
		workerPool: workerPool,
		localCache: cache,
		cacheTTL:   5 * time.Minute,
	}
}

// ImportKeys imports multiple API keys
func (s *APIKeyService) ImportKeys(keys []string) (*models.ImportResult, error) {
	result := &models.ImportResult{
		Success:    0,
		Failed:     0,
		Duplicates: 0,
	}

	// Get existing keys to check for duplicates
	existingKeys, err := s.store.GetAllAPIKeys()
	if err != nil {
		return result, err
	}

	// Create a map for fast duplicate checking
	existingMap := make(map[string]bool)
	for _, k := range existingKeys {
		existingMap[k.Key] = true
	}

	// Process each key
	for _, keyStr := range keys {
		keyStr = strings.TrimSpace(keyStr)
		if keyStr == "" {
			continue
		}

		// Check for duplicate
		if existingMap[keyStr] {
			result.Duplicates++
			continue
		}

		// Generate unique ID
		id := fmt.Sprintf("key-%s-%d", uuid.New().String()[:8], time.Now().Unix())

		// Create API key object
		apiKey := &storage.APIKey{
			ID:        id,
			Key:       keyStr,
			Name:      fmt.Sprintf("Key %s", id),
			CreatedAt: time.Now(),
		}

		// Save to storage
		if err := s.store.SaveAPIKey(apiKey); err != nil {
			result.Failed++
		} else {
			result.Success++
			existingMap[keyStr] = true // Add to map to prevent duplicates in same batch
		}
	}

	return result, nil
}

// GetAllKeys retrieves all API keys with masked values
func (s *APIKeyService) GetAllKeys() ([]*models.APIKeyMasked, error) {
	keys, err := s.store.GetAllAPIKeys()
	if err != nil {
		return nil, err
	}

	maskedKeys := make([]*models.APIKeyMasked, len(keys))
	for i, key := range keys {
		masked := s.maskKey(key.Key)
		maskedKeys[i] = &models.APIKeyMasked{
			ID:        key.ID,
			Name:      key.Name,
			Masked:    masked,
			CreatedAt: key.CreatedAt,
		}
	}

	return maskedKeys, nil
}

// GetFullKey retrieves the full API key by ID
func (s *APIKeyService) GetFullKey(id string) (*storage.APIKey, error) {
	return s.store.GetAPIKey(id)
}

// DeleteKey deletes an API key
func (s *APIKeyService) DeleteKey(id string) error {
	// Clear from local cache
	_ = s.localCache.Delete(id)
	
	return s.store.DeleteAPIKey(id)
}

// BatchDeleteKeys deletes multiple API keys
func (s *APIKeyService) BatchDeleteKeys(ids []string) (*models.BatchDeleteResult, error) {
	success, failed := s.store.BatchDeleteAPIKeys(ids)
	
	// Clear from local cache
	for _, id := range ids {
		_ = s.localCache.Delete(id)
	}

	return &models.BatchDeleteResult{
		Success: success,
		Failed:  failed,
	}, nil
}

// GetAggregatedData fetches and aggregates usage data for all keys
func (s *APIKeyService) GetAggregatedData() (*models.AggregatedData, error) {
	// Get all API keys
	keys, err := s.store.GetAllAPIKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	if len(keys) == 0 {
		return &models.AggregatedData{
			UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
			TotalCount: 0,
			Totals:     models.Totals{},
			Data:       []*models.Usage{},
		}, nil
	}

	// Check cache first
	cachedResults := make([]*models.Usage, 0)
	uncachedKeys := make([]*storage.APIKey, 0)

	for _, key := range keys {
		// Try to get from cache
		usage, err := s.store.GetUsage(key.ID)
		if err == nil && usage != nil {
			// Check if cache is still valid (within TTL)
			if time.Since(usage.LastUpdated) < s.cacheTTL {
				cachedResults = append(cachedResults, usage)
				continue
			}
		}
		uncachedKeys = append(uncachedKeys, key)
	}

	// Fetch uncached keys using worker pool
	var freshResults []*models.Usage
	if len(uncachedKeys) > 0 {
		freshResults, err = s.workerPool.BatchProcess(uncachedKeys)
		if err != nil {
			return nil, fmt.Errorf("failed to process keys: %w", err)
		}

		// Save fresh results to cache
		validResults := make([]*storage.Usage, 0)
		for _, usage := range freshResults {
			if usage.Error == "" {
				storageUsage := &storage.Usage{
					ID:             usage.ID,
					StartDate:      usage.StartDate,
					EndDate:        usage.EndDate,
					TotalAllowance: usage.TotalAllowance,
					OrgTotalUsed:   usage.OrgTotalUsed,
					Remaining:      usage.Remaining,
					UsedRatio:      usage.UsedRatio,
					LastUpdated:    usage.LastUpdated,
				}
				validResults = append(validResults, storageUsage)
			}
		}
		
		if len(validResults) > 0 {
			_ = s.store.BatchSaveUsage(validResults, s.cacheTTL)
		}
	}

	// Combine results
	allResults := append(cachedResults, freshResults...)

	// Calculate totals
	totals := models.Totals{
		TotalOrgTotalTokensUsed: 0,
		TotalAllowance:          0,
	}

	for _, usage := range allResults {
		if usage.Error == "" {
			totals.TotalOrgTotalTokensUsed += usage.OrgTotalUsed
			totals.TotalAllowance += usage.TotalAllowance
		}
	}

	// Print keys with remaining balance > 0
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📋 API Keys with remaining balance > 0:")
	fmt.Println(strings.Repeat("-", 80))
	
	hasPositiveBalance := false
	for i, usage := range allResults {
		if usage.Error == "" && usage.Remaining > 0 {
			// Find the original key
			for _, key := range keys {
				if key.ID == usage.ID {
					fmt.Println(key.Key)
					hasPositiveBalance = true
					break
				}
			}
		}
	}
	
	if !hasPositiveBalance {
		fmt.Println("⚠️  No API Keys with remaining balance > 0")
	}
	fmt.Println(strings.Repeat("=", 80) + "\n")

	return &models.AggregatedData{
		UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
		TotalCount: len(keys),
		Totals:     totals,
		Data:       allResults,
	}, nil
}

// maskKey masks an API key for display
func (s *APIKeyService) maskKey(key string) string {
	if len(key) <= 8 {
		return key
	}
	return fmt.Sprintf("%s...%s", key[:4], key[len(key)-4:])
}

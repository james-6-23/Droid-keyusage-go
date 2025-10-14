package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/droid-keyusage-go/internal/models"
	"github.com/droid-keyusage-go/internal/storage"
)

// Task represents a work task
type Task struct {
	ID     string
	APIKey string
}

// Result represents task result
type Result struct {
	ID    string
	Usage *models.Usage
	Error error
}

// WorkerPool manages concurrent API calls
type WorkerPool struct {
	maxWorkers   int
	queueSize    int
	taskQueue    chan Task
	resultQueue  chan Result
	wg           sync.WaitGroup
	shutdown     chan struct{}
	httpClient   *http.Client
	activeWorkers int32
	processedTasks int64
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers, queueSize int) *WorkerPool {
	// Create HTTP client with connection pooling
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        maxWorkers * 2,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true,
			DisableKeepAlives:   false,
		},
	}

	return &WorkerPool{
		maxWorkers:  maxWorkers,
		queueSize:   queueSize,
		taskQueue:   make(chan Task, queueSize),
		resultQueue: make(chan Result, queueSize),
		shutdown:    make(chan struct{}),
		httpClient:  httpClient,
	}
}

// Start initializes and starts worker goroutines
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.maxWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop gracefully shuts down the worker pool
func (wp *WorkerPool) Stop() {
	close(wp.shutdown)
	wp.wg.Wait()
	close(wp.taskQueue)
	close(wp.resultQueue)
}

// worker processes tasks from the queue
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	atomic.AddInt32(&wp.activeWorkers, 1)
	defer atomic.AddInt32(&wp.activeWorkers, -1)

	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}
			
			result := wp.processTask(task)
			
			select {
			case wp.resultQueue <- result:
				atomic.AddInt64(&wp.processedTasks, 1)
			case <-wp.shutdown:
				return
			}
			
		case <-wp.shutdown:
			return
		}
	}
}

// processTask fetches usage data for an API key
func (wp *WorkerPool) processTask(task Task) Result {
	usage, err := wp.fetchUsageFromAPI(task.ID, task.APIKey)
	return Result{
		ID:    task.ID,
		Usage: usage,
		Error: err,
	}
}

// fetchUsageFromAPI calls Factory.ai API
func (wp *WorkerPool) fetchUsageFromAPI(id, apiKey string) (*models.Usage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", 
		"https://app.factory.ai/api/organization/members/chat-usage", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := wp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &models.Usage{
			ID:    id,
			Error: fmt.Sprintf("HTTP %d", resp.StatusCode),
		}, nil
	}

	// Parse response
	var apiResp models.FactoryAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Format dates
	formatDate := func(timestamp int64) string {
		if timestamp == 0 {
			return "N/A"
		}
		return time.Unix(timestamp/1000, 0).Format("2006-01-02")
	}

	// Mask API key
	maskedKey := fmt.Sprintf("%s...%s", apiKey[:min(4, len(apiKey))], 
		apiKey[max(0, len(apiKey)-4):])

	usage := &models.Usage{
		ID:             id,
		Key:            maskedKey,
		StartDate:      formatDate(apiResp.Usage.StartDate),
		EndDate:        formatDate(apiResp.Usage.EndDate),
		TotalAllowance: apiResp.Usage.Standard.TotalAllowance,
		OrgTotalUsed:   apiResp.Usage.Standard.OrgTotalTokensUsed,
		Remaining:      apiResp.Usage.Standard.TotalAllowance - apiResp.Usage.Standard.OrgTotalTokensUsed,
		UsedRatio:      apiResp.Usage.Standard.UsedRatio,
		LastUpdated:    time.Now(),
	}

	return usage, nil
}

// SubmitTask adds a task to the queue
func (wp *WorkerPool) SubmitTask(task Task) error {
	select {
	case wp.taskQueue <- task:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("task queue is full")
	}
}

// GetResult retrieves a result from the result queue
func (wp *WorkerPool) GetResult() (Result, bool) {
	select {
	case result, ok := <-wp.resultQueue:
		return result, ok
	case <-time.After(100 * time.Millisecond):
		return Result{}, false
	}
}

// BatchProcess processes multiple API keys concurrently
func (wp *WorkerPool) BatchProcess(keys []*storage.APIKey) ([]*models.Usage, error) {
	results := make([]*models.Usage, 0, len(keys))
	resultMap := make(map[string]*models.Usage)
	var mu sync.Mutex

	// Submit all tasks
	for _, key := range keys {
		task := Task{
			ID:     key.ID,
			APIKey: key.Key,
		}
		if err := wp.SubmitTask(task); err != nil {
			// Log error but continue with other keys
			continue
		}
	}

	// Collect results
	timeout := time.After(30 * time.Second)
	received := 0

	for received < len(keys) {
		select {
		case result, ok := <-wp.resultQueue:
			if !ok {
				break
			}
			
			mu.Lock()
			if result.Error != nil {
				// Create error usage entry
				resultMap[result.ID] = &models.Usage{
					ID:    result.ID,
					Error: result.Error.Error(),
				}
			} else {
				resultMap[result.ID] = result.Usage
			}
			mu.Unlock()
			
			received++
			
		case <-timeout:
			// Timeout reached, return what we have
			break
		}
	}

	// Convert map to slice maintaining order
	for _, key := range keys {
		if usage, exists := resultMap[key.ID]; exists {
			results = append(results, usage)
		} else {
			// Add placeholder for missing results
			results = append(results, &models.Usage{
				ID:    key.ID,
				Error: "Processing timeout",
			})
		}
	}

	return results, nil
}

// GetStats returns worker pool statistics
func (wp *WorkerPool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"active_workers":   atomic.LoadInt32(&wp.activeWorkers),
		"queue_size":       len(wp.taskQueue),
		"result_queue_size": len(wp.resultQueue),
		"processed_tasks":  atomic.LoadInt64(&wp.processedTasks),
		"max_workers":      wp.maxWorkers,
		"queue_capacity":   wp.queueSize,
	}
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

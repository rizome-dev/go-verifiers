package utils

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BatchProcessor processes items concurrently in batches
type BatchProcessor[T any, R any] struct {
	maxConcurrent int
	timeout       time.Duration
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor[T any, R any](maxConcurrent int, timeout time.Duration) *BatchProcessor[T, R] {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	
	return &BatchProcessor[T, R]{
		maxConcurrent: maxConcurrent,
		timeout:       timeout,
	}
}

// ProcessResult contains the result of processing a single item
type ProcessResult[R any] struct {
	Index  int
	Result R
	Error  error
}

// Process executes the processor function on all items concurrently
func (b *BatchProcessor[T, R]) Process(ctx context.Context, items []T, processor func(context.Context, T) (R, error)) []ProcessResult[R] {
	results := make([]ProcessResult[R], len(items))
	
	// Create semaphore for concurrency control
	sem := make(chan struct{}, b.maxConcurrent)
	
	// WaitGroup to track completion
	var wg sync.WaitGroup
	
	// Process each item
	for i, item := range items {
		wg.Add(1)
		
		go func(index int, item T) {
			defer wg.Done()
			
			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[index] = ProcessResult[R]{
					Index: index,
					Error: ctx.Err(),
				}
				return
			}
			
			// Create timeout context for this item
			itemCtx, cancel := context.WithTimeout(ctx, b.timeout)
			defer cancel()
			
			// Process the item
			result, err := processor(itemCtx, item)
			results[index] = ProcessResult[R]{
				Index:  index,
				Result: result,
				Error:  err,
			}
		}(i, item)
	}
	
	// Wait for all items to complete
	wg.Wait()
	
	return results
}

// ProcessWithProgress processes items and reports progress
func (b *BatchProcessor[T, R]) ProcessWithProgress(
	ctx context.Context,
	items []T,
	processor func(context.Context, T) (R, error),
	progress func(completed, total int),
) []ProcessResult[R] {
	
	results := make([]ProcessResult[R], len(items))
	completed := 0
	var mu sync.Mutex
	
	// Create semaphore for concurrency control
	sem := make(chan struct{}, b.maxConcurrent)
	
	// WaitGroup to track completion
	var wg sync.WaitGroup
	
	// Process each item
	for i, item := range items {
		wg.Add(1)
		
		go func(index int, item T) {
			defer wg.Done()
			
			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[index] = ProcessResult[R]{
					Index: index,
					Error: ctx.Err(),
				}
				mu.Lock()
				completed++
				progress(completed, len(items))
				mu.Unlock()
				return
			}
			
			// Create timeout context for this item
			itemCtx, cancel := context.WithTimeout(ctx, b.timeout)
			defer cancel()
			
			// Process the item
			result, err := processor(itemCtx, item)
			results[index] = ProcessResult[R]{
				Index:  index,
				Result: result,
				Error:  err,
			}
			
			// Update progress
			mu.Lock()
			completed++
			progress(completed, len(items))
			mu.Unlock()
		}(i, item)
	}
	
	// Wait for all items to complete
	wg.Wait()
	
	return results
}

// Retry implements exponential backoff retry logic
func Retry[T any](ctx context.Context, maxRetries int, initialDelay time.Duration, fn func(context.Context) (T, error)) (T, error) {
	var result T
	var err error
	
	delay := initialDelay
	
	for i := 0; i <= maxRetries; i++ {
		result, err = fn(ctx)
		if err == nil {
			return result, nil
		}
		
		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		
		// Last attempt, return the error
		if i == maxRetries {
			break
		}
		
		// Wait before retrying
		select {
		case <-time.After(delay):
			// Exponential backoff
			delay *= 2
		case <-ctx.Done():
			return result, ctx.Err()
		}
	}
	
	return result, fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}

// ParallelMap applies a function to all items in parallel
func ParallelMap[T any, R any](ctx context.Context, items []T, maxConcurrent int, fn func(context.Context, T) (R, error)) ([]R, error) {
	processor := NewBatchProcessor[T, R](maxConcurrent, 0)
	results := processor.Process(ctx, items, fn)
	
	// Extract results and check for errors
	output := make([]R, len(results))
	for _, res := range results {
		if res.Error != nil {
			return nil, fmt.Errorf("error processing item %d: %w", res.Index, res.Error)
		}
		output[res.Index] = res.Result
	}
	
	return output, nil
}

// ChunkSlice splits a slice into chunks of specified size
func ChunkSlice[T any](items []T, chunkSize int) [][]T {
	if chunkSize <= 0 {
		chunkSize = 1
	}
	
	var chunks [][]T
	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	
	return chunks
}
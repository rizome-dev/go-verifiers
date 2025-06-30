package types

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
)

// SimpleDataset implements the Dataset interface
type SimpleDataset struct {
	data []map[string]interface{}
	mu   sync.RWMutex
}

// NewSimpleDataset creates a new simple dataset
func NewSimpleDataset(data []map[string]interface{}) *SimpleDataset {
	return &SimpleDataset{
		data: data,
	}
}

// Len returns the number of items in the dataset
func (d *SimpleDataset) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.data)
}

// Get returns the item at the specified index
func (d *SimpleDataset) Get(idx int) map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	if idx < 0 || idx >= len(d.data) {
		return nil
	}
	
	// Return a copy to prevent modification
	item := make(map[string]interface{})
	for k, v := range d.data[idx] {
		item[k] = v
	}
	return item
}

// Shuffle returns a new shuffled dataset
func (d *SimpleDataset) Shuffle(seed int64) Dataset {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Create a copy of the data
	newData := make([]map[string]interface{}, len(d.data))
	copy(newData, d.data)
	
	// Shuffle the copy
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(newData), func(i, j int) {
		newData[i], newData[j] = newData[j], newData[i]
	})
	
	return NewSimpleDataset(newData)
}

// Select returns a new dataset with only the specified indices
func (d *SimpleDataset) Select(indices []int) Dataset {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	newData := make([]map[string]interface{}, 0, len(indices))
	for _, idx := range indices {
		if idx >= 0 && idx < len(d.data) {
			// Deep copy the item
			item := make(map[string]interface{})
			for k, v := range d.data[idx] {
				item[k] = v
			}
			newData = append(newData, item)
		}
	}
	
	return NewSimpleDataset(newData)
}

// Map applies a function to each item and returns a new dataset
func (d *SimpleDataset) Map(fn func(map[string]interface{}) map[string]interface{}) Dataset {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	newData := make([]map[string]interface{}, len(d.data))
	for i, item := range d.data {
		// Create a copy of the item
		itemCopy := make(map[string]interface{})
		for k, v := range item {
			itemCopy[k] = v
		}
		// Apply the function
		newData[i] = fn(itemCopy)
	}
	
	return NewSimpleDataset(newData)
}

// DatasetBuilder helps construct datasets
type DatasetBuilder struct {
	data []map[string]interface{}
}

// NewDatasetBuilder creates a new dataset builder
func NewDatasetBuilder() *DatasetBuilder {
	return &DatasetBuilder{
		data: make([]map[string]interface{}, 0),
	}
}

// Add adds an item to the dataset
func (b *DatasetBuilder) Add(item map[string]interface{}) *DatasetBuilder {
	b.data = append(b.data, item)
	return b
}

// AddFromJSON adds items from JSON data
func (b *DatasetBuilder) AddFromJSON(jsonData string) error {
	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &items); err != nil {
		// Try single item
		var item map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &item); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		items = []map[string]interface{}{item}
	}
	
	b.data = append(b.data, items...)
	return nil
}

// Build creates the dataset
func (b *DatasetBuilder) Build() Dataset {
	return NewSimpleDataset(b.data)
}

// DatasetUtils provides utility functions for datasets
type DatasetUtils struct{}

// LoadFromPromptAnswer creates a dataset from prompt-answer pairs
func (DatasetUtils) LoadFromPromptAnswer(pairs []struct{ Prompt, Answer string }) Dataset {
	builder := NewDatasetBuilder()
	for _, pair := range pairs {
		builder.Add(map[string]interface{}{
			"prompt": pair.Prompt,
			"answer": pair.Answer,
		})
	}
	return builder.Build()
}

// LoadFromQuestionAnswer creates a dataset from question-answer pairs
func (DatasetUtils) LoadFromQuestionAnswer(pairs []struct{ Question, Answer string }) Dataset {
	builder := NewDatasetBuilder()
	for _, pair := range pairs {
		builder.Add(map[string]interface{}{
			"question": pair.Question,
			"answer":   pair.Answer,
		})
	}
	return builder.Build()
}

// Filter filters a dataset based on a predicate
func (DatasetUtils) Filter(dataset Dataset, predicate func(map[string]interface{}) bool) Dataset {
	indices := make([]int, 0)
	for i := 0; i < dataset.Len(); i++ {
		if predicate(dataset.Get(i)) {
			indices = append(indices, i)
		}
	}
	return dataset.Select(indices)
}

// Concatenate combines multiple datasets
func (DatasetUtils) Concatenate(datasets ...Dataset) Dataset {
	builder := NewDatasetBuilder()
	for _, dataset := range datasets {
		for i := 0; i < dataset.Len(); i++ {
			builder.Add(dataset.Get(i))
		}
	}
	return builder.Build()
}
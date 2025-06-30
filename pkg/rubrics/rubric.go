package rubrics

import (
	"context"
	"strings"

	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// Rubric is the interface for evaluating model outputs
type Rubric interface {
	// GetRewardFuncs returns the reward functions for this rubric
	GetRewardFuncs() []types.RewardFunc
	
	// GetRewardWeights returns the weights for each reward function
	GetRewardWeights() []float64
	
	// ComputeReward computes the total reward given parsed response and ground truth
	ComputeReward(ctx context.Context, parsed string, groundTruth string) (float64, error)
}

// BaseRubric provides a default exact match implementation
type BaseRubric struct {
	rewardFuncs   []types.RewardFunc
	rewardWeights []float64
}

// NewBaseRubric creates a new base rubric with exact match
func NewBaseRubric() *BaseRubric {
	rubric := &BaseRubric{
		rewardWeights: []float64{1.0},
	}
	
	// Default exact match reward function
	exactMatchReward := func(ctx context.Context, parsed, groundTruth string) (float64, error) {
		if strings.TrimSpace(parsed) == strings.TrimSpace(groundTruth) {
			return 1.0, nil
		}
		return 0.0, nil
	}
	
	rubric.rewardFuncs = []types.RewardFunc{exactMatchReward}
	return rubric
}

// GetRewardFuncs returns the reward functions
func (r *BaseRubric) GetRewardFuncs() []types.RewardFunc {
	return r.rewardFuncs
}

// GetRewardWeights returns the reward weights
func (r *BaseRubric) GetRewardWeights() []float64 {
	return r.rewardWeights
}

// ComputeReward computes the weighted sum of all reward functions
func (r *BaseRubric) ComputeReward(ctx context.Context, parsed string, groundTruth string) (float64, error) {
	if len(r.rewardFuncs) == 0 {
		return 0.0, nil
	}
	
	totalReward := 0.0
	totalWeight := 0.0
	
	for i, fn := range r.rewardFuncs {
		weight := 1.0
		if i < len(r.rewardWeights) {
			weight = r.rewardWeights[i]
		}
		
		reward, err := fn(ctx, parsed, groundTruth)
		if err != nil {
			return 0.0, err
		}
		
		totalReward += reward * weight
		totalWeight += weight
	}
	
	if totalWeight > 0 {
		return totalReward / totalWeight, nil
	}
	
	return 0.0, nil
}

// MultiMetricRubric supports multiple evaluation metrics
type MultiMetricRubric struct {
	BaseRubric
	metrics map[string]types.RewardFunc
}

// NewMultiMetricRubric creates a rubric with multiple metrics
func NewMultiMetricRubric() *MultiMetricRubric {
	return &MultiMetricRubric{
		BaseRubric: *NewBaseRubric(),
		metrics:    make(map[string]types.RewardFunc),
	}
}

// AddMetric adds a named metric to the rubric
func (r *MultiMetricRubric) AddMetric(name string, fn types.RewardFunc, weight float64) {
	r.metrics[name] = fn
	r.rewardFuncs = append(r.rewardFuncs, fn)
	r.rewardWeights = append(r.rewardWeights, weight)
}

// GetMetric returns a specific metric by name
func (r *MultiMetricRubric) GetMetric(name string) (types.RewardFunc, bool) {
	fn, ok := r.metrics[name]
	return fn, ok
}
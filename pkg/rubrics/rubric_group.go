package rubrics

import (
	"context"
	"fmt"

	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// RubricGroup aggregates multiple rubrics into one
type RubricGroup struct {
	rubrics      []Rubric
	rubricNames  []string
	mergeWeights bool // Whether to merge weights for same-named functions
}

// NewRubricGroup creates a new rubric group
func NewRubricGroup(rubrics map[string]Rubric, mergeWeights bool) *RubricGroup {
	group := &RubricGroup{
		rubrics:      make([]Rubric, 0, len(rubrics)),
		rubricNames:  make([]string, 0, len(rubrics)),
		mergeWeights: mergeWeights,
	}

	// Maintain consistent ordering
	for name, rubric := range rubrics {
		group.rubricNames = append(group.rubricNames, name)
		group.rubrics = append(group.rubrics, rubric)
	}

	return group
}

// GetRewardFuncs returns combined reward functions from all rubrics
func (r *RubricGroup) GetRewardFuncs() []types.RewardFunc {
	funcs := make([]types.RewardFunc, 0)

	if r.mergeWeights {
		// Merge functions with the same name
		funcMap := make(map[string][]types.RewardFunc)
		
		for i, rubric := range r.rubrics {
			rubricFuncs := rubric.GetRewardFuncs()
			for j, fn := range rubricFuncs {
				// Create a unique key for each function
				// In practice, we'd need a way to identify function names
				key := fmt.Sprintf("func_%d_%d", i, j)
				funcMap[key] = append(funcMap[key], fn)
			}
		}

		// Create merged functions
		for _, fns := range funcMap {
			if len(fns) == 1 {
				funcs = append(funcs, fns[0])
			} else {
				// Create a merged function that runs all and averages
				mergedFunc := r.createMergedFunc(fns)
				funcs = append(funcs, mergedFunc)
			}
		}
	} else {
		// Simply concatenate all functions
		for _, rubric := range r.rubrics {
			funcs = append(funcs, rubric.GetRewardFuncs()...)
		}
	}

	return funcs
}

// GetRewardWeights returns combined weights from all rubrics
func (r *RubricGroup) GetRewardWeights() []float64 {
	weights := make([]float64, 0)

	if r.mergeWeights {
		// When merging, weights should match the merged functions
		// For simplicity, we'll average weights for merged functions
		funcCount := len(r.GetRewardFuncs())
		totalWeight := 0.0
		
		for _, rubric := range r.rubrics {
			rubricWeights := rubric.GetRewardWeights()
			for _, w := range rubricWeights {
				totalWeight += w
			}
		}

		// Distribute weight evenly among merged functions
		if funcCount > 0 {
			avgWeight := totalWeight / float64(funcCount)
			for i := 0; i < funcCount; i++ {
				weights = append(weights, avgWeight)
			}
		}
	} else {
		// Simply concatenate all weights
		for _, rubric := range r.rubrics {
			weights = append(weights, rubric.GetRewardWeights()...)
		}
	}

	return weights
}

// ComputeReward runs all rubrics and combines their scores
func (r *RubricGroup) ComputeReward(ctx context.Context, parsed string, groundTruth string) (float64, error) {
	totalScore := 0.0
	totalWeight := 0.0

	// Run each rubric
	for _, rubric := range r.rubrics {
		score, err := rubric.ComputeReward(ctx, parsed, groundTruth)
		if err != nil {
			// Log error but continue with other rubrics
			// In practice, we might want to handle this differently
			continue
		}

		// Get the weight for this rubric (sum of its function weights)
		rubricWeights := rubric.GetRewardWeights()
		rubricWeight := 0.0
		for _, w := range rubricWeights {
			rubricWeight += w
		}

		// If no weights defined, assume weight of 1.0
		if rubricWeight == 0 && len(rubricWeights) == 0 {
			rubricWeight = 1.0
		}

		totalScore += score * rubricWeight
		totalWeight += rubricWeight
	}

	if totalWeight > 0 {
		return totalScore / totalWeight, nil
	}

	return 0.0, nil
}

// createMergedFunc creates a function that runs multiple functions and averages their results
func (r *RubricGroup) createMergedFunc(funcs []types.RewardFunc) types.RewardFunc {
	return func(ctx context.Context, parsed, groundTruth string) (float64, error) {
		totalScore := 0.0
		successCount := 0

		for _, fn := range funcs {
			score, err := fn(ctx, parsed, groundTruth)
			if err == nil {
				totalScore += score
				successCount++
			}
		}

		if successCount > 0 {
			return totalScore / float64(successCount), nil
		}

		return 0.0, nil
	}
}

// AddRubric adds a new rubric to the group
func (r *RubricGroup) AddRubric(name string, rubric Rubric) {
	r.rubricNames = append(r.rubricNames, name)
	r.rubrics = append(r.rubrics, rubric)
}

// GetRubric returns a specific rubric by name
func (r *RubricGroup) GetRubric(name string) (Rubric, bool) {
	for i, rubricName := range r.rubricNames {
		if rubricName == name {
			return r.rubrics[i], true
		}
	}
	return nil, false
}

// Names returns the names of all rubrics in the group
func (r *RubricGroup) Names() []string {
	names := make([]string, len(r.rubricNames))
	copy(names, r.rubricNames)
	return names
}

// EnvGroupRubric is a specialized rubric for environment groups
type EnvGroupRubric struct {
	*RubricGroup
	envRubrics map[string]Rubric
}

// NewEnvGroupRubric creates a rubric for environment groups
func NewEnvGroupRubric(envRubrics map[string]Rubric) *EnvGroupRubric {
	return &EnvGroupRubric{
		RubricGroup: NewRubricGroup(envRubrics, false),
		envRubrics:  envRubrics,
	}
}

// ComputeRewardForTask computes reward for a specific task
func (r *EnvGroupRubric) ComputeRewardForTask(ctx context.Context, task string, parsed string, groundTruth string) (float64, error) {
	rubric, exists := r.envRubrics[task]
	if !exists {
		return 0.0, fmt.Errorf("no rubric found for task: %s", task)
	}

	return rubric.ComputeReward(ctx, parsed, groundTruth)
}
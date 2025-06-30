package envs

import (
	"context"
	"fmt"
	"strings"

	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// EnvGroup manages multiple environments as a single unified environment
type EnvGroup struct {
	*BaseEnvironment
	envs      map[string]Environment
	envNames  []string // Ordered list of environment names
}

// NewEnvGroup creates a new environment group
func NewEnvGroup(config types.Config, envs map[string]Environment) *EnvGroup {
	group := &EnvGroup{
		BaseEnvironment: NewBaseEnvironment(config),
		envs:            envs,
		envNames:        make([]string, 0, len(envs)),
	}

	// Maintain consistent ordering
	for name := range envs {
		group.envNames = append(group.envNames, name)
	}

	return group
}

// Rollout routes to the appropriate sub-environment based on task
func (g *EnvGroup) Rollout(ctx context.Context, client types.Client, model string, prompt interface{}, answer string, samplingArgs types.SamplingArgs) (*types.Rollout, error) {
	// Extract task from answer format "task:answer"
	task, actualAnswer := g.parseTaskAnswer(answer)
	
	// Find the appropriate environment
	env, exists := g.envs[task]
	if !exists {
		return nil, fmt.Errorf("unknown task: %s", task)
	}

	// Delegate to the specific environment
	return env.Rollout(ctx, client, model, prompt, actualAnswer, samplingArgs)
}

// GetDataset returns concatenated datasets with task labels
func (g *EnvGroup) GetDataset(n int, seed int64) types.Dataset {
	datasets := make([]types.Dataset, 0)
	
	for _, envName := range g.envNames {
		env := g.envs[envName]
		dataset := env.GetDataset(-1, seed) // Get all items
		
		if dataset != nil {
			// Add task label to each item
			labeledDataset := dataset.Map(func(item map[string]interface{}) map[string]interface{} {
				newItem := make(map[string]interface{})
				for k, v := range item {
					newItem[k] = v
				}
				// Store original answer and create task-prefixed answer
				if answer, ok := item["answer"].(string); ok {
					newItem["answer"] = fmt.Sprintf("%s:%s", envName, answer)
				}
				newItem["task"] = envName
				return newItem
			})
			datasets = append(datasets, labeledDataset)
		}
	}

	// Concatenate all datasets
	if len(datasets) == 0 {
		return nil
	}

	combined := types.DatasetUtils{}.Concatenate(datasets...)
	
	// Apply sampling if requested
	if n > 0 && n < combined.Len() {
		return combined.Shuffle(seed).Select(makeRange(n))
	}
	
	return combined
}

// GetEvalDataset returns concatenated eval datasets with task labels
func (g *EnvGroup) GetEvalDataset(n int, seed int64) types.Dataset {
	datasets := make([]types.Dataset, 0)
	
	for _, envName := range g.envNames {
		env := g.envs[envName]
		dataset := env.GetEvalDataset(-1, seed)
		
		if dataset != nil {
			// Add task label to each item
			labeledDataset := dataset.Map(func(item map[string]interface{}) map[string]interface{} {
				newItem := make(map[string]interface{})
				for k, v := range item {
					newItem[k] = v
				}
				// Store original answer and create task-prefixed answer
				if answer, ok := item["answer"].(string); ok {
					newItem["answer"] = fmt.Sprintf("%s:%s", envName, answer)
				}
				newItem["task"] = envName
				return newItem
			})
			datasets = append(datasets, labeledDataset)
		}
	}

	// Concatenate all datasets
	if len(datasets) == 0 {
		return nil
	}

	combined := types.DatasetUtils{}.Concatenate(datasets...)
	
	// Apply sampling if requested
	if n > 0 && n < combined.Len() {
		return combined.Shuffle(seed).Select(makeRange(n))
	}
	
	return combined
}

// GetRewardFuncs returns reward functions from all environments
func (g *EnvGroup) GetRewardFuncs() []types.RewardFunc {
	funcs := make([]types.RewardFunc, 0)
	
	// Collect reward functions from each environment
	for _, envName := range g.envNames {
		env := g.envs[envName]
		envFuncs := env.GetRewardFuncs()
		
		// Wrap each function to handle task routing
		for _, fn := range envFuncs {
			wrappedFn := g.wrapRewardFunc(envName, fn)
			funcs = append(funcs, wrappedFn)
		}
	}
	
	return funcs
}

// GetRewardWeights returns weights for all reward functions
func (g *EnvGroup) GetRewardWeights() []float64 {
	weights := make([]float64, 0)
	
	// Collect weights from each environment
	for _, envName := range g.envNames {
		env := g.envs[envName]
		envWeights := env.GetRewardWeights()
		weights = append(weights, envWeights...)
	}
	
	return weights
}

// parseTaskAnswer extracts task and answer from "task:answer" format
func (g *EnvGroup) parseTaskAnswer(answer string) (string, string) {
	parts := strings.SplitN(answer, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	// Default to first environment if no task specified
	if len(g.envNames) > 0 {
		return g.envNames[0], answer
	}
	return "", answer
}

// wrapRewardFunc wraps a reward function to handle task routing
func (g *EnvGroup) wrapRewardFunc(envName string, fn types.RewardFunc) types.RewardFunc {
	return func(ctx context.Context, parsed, groundTruth string) (float64, error) {
		// Extract task from ground truth
		task, actualGroundTruth := g.parseTaskAnswer(groundTruth)
		
		// If this isn't the right task, return 0
		if task != envName {
			return 0.0, nil
		}
		
		// Call the original function with the actual ground truth
		return fn(ctx, parsed, actualGroundTruth)
	}
}
# go-verifiers

A pure Go implementation of verifiers for reinforcement learning with LLMs, migrated from [willccbb/verifiers](https://github.com/willccbb/verifiers).


[![GoDoc](https://pkg.go.dev/badge/github.com/rizome-dev/go-verifiers)](https://pkg.go.dev/github.com/rizome-dev/go-verifiers)
[![Go Report Card](https://goreportcard.com/badge/github.com/rizome-dev/go-verifiers)](https://goreportcard.com/report/github.com/rizome-dev/go-verifiers)

```shell
go get github.com/rizome-dev/go-verifiers
```

built by: [rizome labs](https://rizome.dev)

contact us: [hi (at) rizome.dev](mailto:hi@rizome.dev)

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rizome-dev/go-verifiers/pkg/envs"
    "github.com/rizome-dev/go-verifiers/pkg/inference"
    "github.com/rizome-dev/go-verifiers/pkg/parsers"
    "github.com/rizome-dev/go-verifiers/pkg/rubrics"
    "github.com/rizome-dev/go-verifiers/pkg/types"
)

func main() {
    // Configure environment
    config := types.Config{
        Model:        "gpt-4",
        SystemPrompt: "You are a helpful assistant.",
        MessageType:  "chat",
        SamplingArgs: types.SamplingArgs{
            Temperature: 0.7,
            MaxTokens:   150,
        },
    }

    // Create single-turn environment
    env := envs.NewSingleTurnEnv(config)
    env.SetParser(parsers.NewBaseParser())
    env.SetRubric(rubrics.NewBaseRubric())

    // Create client
    client := inference.NewHTTPClient("http://localhost:8000/v1", "api-key")

    // Run rollout
    ctx := context.Background()
    prompt := env.FormatPrompt("What is 2 + 2?")
    rollout, err := env.Rollout(ctx, client, config.Model, prompt, "4", config.SamplingArgs)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Response: %s\n", rollout.Response)
    fmt.Printf("Score: %.2f\n", rollout.Score)
}
```

## Package Structure

```
pkg/
├── types/       # Core types and interfaces
├── envs/        # Environment implementations
├── parsers/     # Response parser implementations
├── rubrics/     # Evaluation rubric implementations
├── inference/   # Inference client implementations
├── tools/       # Tool implementations (calculator, search, etc.)
├── trainers/    # Training utilities
└── utils/       # Utility functions
```

## Key Components

### Environments

- **SingleTurnEnv**: For one-shot question-answer tasks
- **MultiTurnEnv**: For multi-turn conversations and interactions
- **DialogMultiTurnEnv**: Example implementation for dialog-based tasks

### Parsers

- **BaseParser**: Returns trimmed response as-is
- **LastLineParser**: Extracts the last non-empty line
- **RegexParser**: Extracts content matching a pattern (coming soon)

### Rubrics

- **BaseRubric**: Simple exact match evaluation
- **MultiMetricRubric**: Supports multiple weighted metrics

### Inference Client

- **HTTPClient**: OpenAI-compatible HTTP client with connection pooling

## Migration Status

### ✅ Fully Implemented (Pure Go)

**Core Infrastructure:**
- Environment interface with BaseEnvironment
- Parser interface with all implementations (Base, XML, Think, Smola)
- Rubric interface with all implementations (Base, MultiMetric, Math, Tool, CodeMath, Judge, RubricGroup, SmolaToolRubric)
- Type-safe message and configuration structures
- HTTP inference client with OpenAI API compatibility
- Concurrent batch processing utilities
- Dataset interface with builder pattern

**Environment Types:**
- SingleTurnEnv - One-shot question/answer tasks
- MultiTurnEnv - Multi-turn conversations
- ToolEnv - JSON-based tool calling
- SmolaToolEnv - SmolaAgents-style tool usage
- CodeMathEnv - Mathematical expression evaluation (Go-based)
- DoubleCheckEnv - Answer verification mechanism
- EnvGroup - Multiple environments as unified interface

**Parsers:**
- BaseParser - Simple trimming
- XMLParser - Field extraction with alternatives
- ThinkParser - Extract content after </think>
- SmolaParser - XML with tool JSON support

**Rubrics:**
- BaseRubric - Exact match evaluation
- MultiMetricRubric - Weighted metrics
- MathRubric - Mathematical answer evaluation
- ToolRubric - Tool usage evaluation
- CodeMathRubric - Code/expression execution scoring
- JudgeRubric - LLM-based evaluation
- RubricGroup - Aggregate multiple rubrics
- SmolaToolRubric - SmolaAgents tool scoring

**Tools:**
- Calculator - Mathematical expression evaluator
- WebSearch - Web search with caching
- Tool execution framework with JSON parsing

**Utilities:**
- Math utilities (boxed answer extraction, normalization)
- Concurrent processing with progress tracking
- Dataset manipulation and filtering

### ⏳ Not Implemented

These components require external dependencies or services that don't align with pure Go:
- Full training pipeline (GRPO trainer) - Requires deep learning framework
- vLLM server implementation - Python-specific
- HuggingFace dataset integration - Python ecosystem
- TextArena and ReasoningGym environments - External dependencies
- Python code executor - Replaced with Go expression evaluator

## Key Differences from Python Implementation

1. **Pure Go**: No Python dependencies, all mathematical evaluation done in Go
2. **Better Concurrency**: Native goroutines instead of asyncio
3. **Type Safety**: Compile-time type checking prevents runtime errors
4. **Performance**: HTTP connection pooling and efficient concurrent processing
5. **Simplified Architecture**: Cleaner interfaces without Python's complexity

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see LICENSE file for details

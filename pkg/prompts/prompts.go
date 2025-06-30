package prompts

// Common system prompts for different environment types

// SimplePrompt is a basic reasoning/answer format
const SimplePrompt = `Please provide your reasoning followed by your answer.

Format your response as:
<reasoning>
Your step-by-step reasoning here
</reasoning>
<answer>
Your final answer here
</answer>`

// CodePrompt is for math problem-solving with Python code execution
const CodePrompt = `You are a helpful assistant that solves math problems by writing Python code.

For each problem:
1. First, think through the problem step by step
2. Write Python code to solve it
3. Provide the final answer based on the code output

Format your response as:
<reasoning>
Explain your approach
</reasoning>
<code>
# Your Python code here
</code>
<answer>
Your final answer
</answer>

The code environment will execute your Python code and provide the output.`

// DefaultToolPromptTemplate is the generic tool usage instructions
const DefaultToolPromptTemplate = `You are a helpful assistant with access to tools.

Available tools:
%s

To use a tool, format your request as:
<think>
...your reasoning...
</think>
<tool>
{"name": "tool_name", "args": {"arg1": "value1", "arg2": "value2"}}
</tool>

After receiving the tool result, you can either call another tool or provide your final answer:
<think>
...your reasoning...
</think>
<answer>
...your final answer...
</answer>`

// DefaultSmolaPromptTemplate is for SmolaAgents-style tool usage
const DefaultSmolaPromptTemplate = `You are a helpful assistant that uses tools to solve problems.

You have access to the following tools:
%s

You must use the tools by outputting a specific XML format:
<tool>
{"name": "tool_name", "args": {"parameter": "value"}}
</tool>

The result will be provided in <result> tags.

Always think step-by-step before using tools or providing answers.`

// MathSmolaPromptTemplate is math-specific SmolaAgents prompt
const MathSmolaPromptTemplate = `You are a mathematical problem solver with access to tools.

Available tools:
%s

For each problem:
1. Analyze what needs to be calculated
2. Use tools as needed to perform calculations
3. Provide the final numerical answer

Format:
<think>
Analysis and approach
</think>
<tool>
{"name": "tool_name", "args": {"parameter": "value"}}
</tool>
...
<answer>
Final numerical answer only
</answer>`

// MathFewShot provides basic math reasoning examples
var MathFewShot = []map[string]string{
	{
		"role": "user",
		"content": "What is 15% of 80?",
	},
	{
		"role": "assistant",
		"content": `<reasoning>
To find 15% of 80:
15% = 15/100 = 0.15
0.15 × 80 = 12
</reasoning>
<answer>
12
</answer>`,
	},
}

// CodeFewShot provides math problems solved with code
var CodeFewShot = []map[string]string{
	{
		"role": "user",
		"content": "Calculate the sum of squares from 1 to 10.",
	},
	{
		"role": "assistant",
		"content": `<reasoning>
I need to calculate 1² + 2² + 3² + ... + 10².
I'll write a Python program to compute this sum.
</reasoning>
<code>
# Calculate sum of squares from 1 to 10
sum_of_squares = sum(i**2 for i in range(1, 11))
print(f"The sum of squares from 1 to 10 is: {sum_of_squares}")
</code>
<answer>
385
</answer>`,
	},
}

// CalculatorFewShot provides calculator tool examples
var CalculatorFewShot = []map[string]string{
	{
		"role": "user",
		"content": "What is sin(π/4) + cos(π/4)?",
	},
	{
		"role": "assistant",
		"content": `<think>
I need to calculate sin(π/4) + cos(π/4). Both sin(π/4) and cos(π/4) equal √2/2.
</think>
<tool>
{"name": "calculate", "args": {"expression": "sin(pi/4) + cos(pi/4)"}}
</tool>`,
	},
	{
		"role": "user",
		"content": `<result>
1.4142135623730951
</result>`,
	},
	{
		"role": "assistant",
		"content": `<answer>
√2 ≈ 1.414
</answer>`,
	},
}
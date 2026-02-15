package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"rcodegen/pkg/envelope"
)

type Context struct {
	mu           sync.RWMutex
	ctx          context.Context // Signal-aware context for cancellation propagation
	Inputs       map[string]string
	StepResults  map[string]*envelope.Envelope
	Variables    map[string]string
	ToolSessions map[string]string // Tool name -> session ID for reuse
}

func NewContext(parentCtx context.Context, inputs map[string]string) *Context {
	return &Context{
		ctx:          parentCtx,
		Inputs:       inputs,
		StepResults:  make(map[string]*envelope.Envelope),
		Variables:    make(map[string]string),
		ToolSessions: make(map[string]string),
	}
}

// Ctx returns the context.Context for cancellation propagation
func (c *Context) Ctx() context.Context {
	return c.ctx
}

// GetToolSession returns the session ID for a tool, if any
func (c *Context) GetToolSession(toolName string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ToolSessions[toolName]
}

// SetToolSession stores the session ID for a tool
func (c *Context) SetToolSession(toolName, sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ToolSessions[toolName] = sessionID
}

var varPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

func (c *Context) Resolve(s string) string {
	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		ref := match[2 : len(match)-1] // Strip ${ and }
		parts := strings.Split(ref, ".")

		switch parts[0] {
		case "inputs":
			if len(parts) >= 2 {
				c.mu.RLock()
				v, ok := c.Inputs[parts[1]]
				c.mu.RUnlock()
				if ok {
					return v
				}
			}
		case "steps":
			if len(parts) >= 3 {
				stepName := parts[1]
				c.mu.RLock()
				env, ok := c.StepResults[stepName]
				c.mu.RUnlock()
				if ok {
					switch parts[2] {
					case "output_ref":
						return env.OutputRef
					case "status":
						return string(env.Status)
					case "stdout", "stderr":
						// Read from output file — done outside lock
						if env.OutputRef != "" {
							if data, err := os.ReadFile(env.OutputRef); err == nil {
								var output map[string]interface{}
								if err := json.Unmarshal(data, &output); err == nil {
									if v, ok := output[parts[2]]; ok {
										content := fmt.Sprintf("%v", v)
										return extractStreamingResult(content)
									}
								}
							}
						}
					case "result":
						if len(parts) == 3 {
							if b, err := json.Marshal(env.Result); err == nil {
								return string(b)
							}
						} else if len(parts) >= 4 {
							if v, ok := env.Result[parts[3]]; ok {
								return fmt.Sprintf("%v", v)
							}
						}
					}
				}
			}
		}
		return match // Leave unresolved
	})
}

func (c *Context) SetResult(name string, env *envelope.Envelope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.StepResults[name] = env
}

// GetResult safely retrieves a step result with proper locking.
func (c *Context) GetResult(name string) (*envelope.Envelope, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	env, ok := c.StepResults[name]
	return env, ok
}

// extractStreamingResult parses streaming JSON output (from Claude/Codex)
// and extracts the final result text from the "type":"result" object.
func extractStreamingResult(content string) string {
	// Try to find and parse the final result object
	// Streaming output has newline-delimited JSON objects
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		// Look for the result object
		if objType, ok := obj["type"].(string); ok && objType == "result" {
			if result, ok := obj["result"].(string); ok {
				return result
			}
		}
	}
	// If no result object found, return as-is (might be plain text output)
	return content
}

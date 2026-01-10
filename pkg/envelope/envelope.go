package envelope

import "time"

type Status string

const (
	StatusSuccess Status = "success"
	StatusFailure Status = "failure"
	StatusPartial Status = "partial"
	StatusSkipped Status = "skipped"
)

type Envelope struct {
	Status    Status                 `json:"status"`
	Result    map[string]interface{} `json:"result,omitempty"`
	OutputRef string                 `json:"output_ref,omitempty"`
	Error     *ErrorInfo             `json:"error,omitempty"`
	Metrics   *Metrics               `json:"metrics,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Metrics struct {
	Tool       string        `json:"tool"`
	DurationMs int64         `json:"duration_ms"`
	StartTime  time.Time     `json:"start_time"`
	EndTime    time.Time     `json:"end_time"`
}

// Builder pattern
type Builder struct {
	env *Envelope
}

func New() *Builder {
	return &Builder{env: &Envelope{Result: make(map[string]interface{})}}
}

func (b *Builder) WithTool(name string) *Builder {
	if b.env.Metrics == nil {
		b.env.Metrics = &Metrics{}
	}
	b.env.Metrics.Tool = name
	return b
}

func (b *Builder) Success() *Builder {
	b.env.Status = StatusSuccess
	return b
}

func (b *Builder) Failure(code, message string) *Builder {
	b.env.Status = StatusFailure
	b.env.Error = &ErrorInfo{Code: code, Message: message}
	return b
}

func (b *Builder) WithResult(key string, value interface{}) *Builder {
	b.env.Result[key] = value
	return b
}

func (b *Builder) WithOutputRef(path string) *Builder {
	b.env.OutputRef = path
	return b
}

func (b *Builder) WithDuration(ms int64) *Builder {
	if b.env.Metrics == nil {
		b.env.Metrics = &Metrics{}
	}
	b.env.Metrics.DurationMs = ms
	return b
}

func (b *Builder) Build() *Envelope {
	return b.env
}

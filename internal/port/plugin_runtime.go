package port

import (
	"context"
	"encoding/json"
)

type PluginRuntime interface {
	Start(ctx context.Context) error
	Call(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error)
	Stop(ctx context.Context) error
	Healthy() bool
}

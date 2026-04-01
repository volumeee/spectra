package port

import (
	"context"
	"encoding/json"

	"github.com/spectra-browser/spectra/internal/domain"
)

type PluginManager interface {
	Discover(ctx context.Context) error
	Get(name string) (*domain.PluginInfo, error)
	List() []domain.PluginInfo
	Execute(ctx context.Context, plugin, method string, params json.RawMessage) (*domain.JobResult, error)
	Shutdown(ctx context.Context) error
}

package spectra

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spectra-browser/spectra/pkg/protocol"
)

type HandlerFunc func(ctx context.Context, params json.RawMessage) (interface{}, error)

type Plugin struct {
	name     string
	handlers map[string]HandlerFunc
}

func NewPlugin(name string) *Plugin {
	return &Plugin{name: name, handlers: make(map[string]HandlerFunc)}
}

func (p *Plugin) Handle(method string, handler HandlerFunc) {
	p.handlers[method] = handler
}

func (p *Plugin) Run() {
	log.SetOutput(os.Stderr) // stdout reserved for JSON-RPC
	log.Printf("[%s] plugin started", p.name)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var req protocol.Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			writeError(req.ID, protocol.ErrCodeParse, "parse error")
			continue
		}

		handler, ok := p.handlers[req.Method]
		if !ok {
			writeError(req.ID, protocol.ErrCodeNotFound, fmt.Sprintf("method %s not found", req.Method))
			continue
		}

		result, err := handler(ctx, req.Params)
		if err != nil {
			writeError(req.ID, protocol.ErrCodeInternal, err.Error())
			continue
		}

		resp, err := protocol.NewResponse(req.ID, result)
		if err != nil {
			writeError(req.ID, protocol.ErrCodeInternal, "marshal result: "+err.Error())
			continue
		}
		writeJSON(resp)
	}
}

func writeError(id int64, code int, msg string) {
	writeJSON(protocol.NewErrorResponse(id, code, msg))
}

func writeJSON(v interface{}) {
	b, _ := json.Marshal(v)
	b = append(b, '\n')
	os.Stdout.Write(b)
}

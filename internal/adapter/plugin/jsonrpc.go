package plugin

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"

	"github.com/spectra-browser/spectra/pkg/protocol"
)

type JSONRPCCodec struct {
	mu      sync.Mutex
	writer  io.Writer
	scanner *bufio.Scanner
}

func NewJSONRPCCodec(r io.Reader, w io.Writer) *JSONRPCCodec {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 4*1024*1024), 4*1024*1024) // 4MB max line
	return &JSONRPCCodec{writer: w, scanner: s}
}

func (c *JSONRPCCodec) WriteRequest(req *protocol.Request) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = c.writer.Write(b)
	return err
}

func (c *JSONRPCCodec) ReadResponse() (*protocol.Response, error) {
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	var resp protocol.Response
	if err := json.Unmarshal(c.scanner.Bytes(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

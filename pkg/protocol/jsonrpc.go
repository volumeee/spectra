package protocol

import "encoding/json"

const JSONRPCVersion = "2.0"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	ErrCodeParse      = -32700
	ErrCodeInvalidReq = -32600
	ErrCodeNotFound   = -32601
	ErrCodeInternal   = -32603
)

func NewRequest(id int64, method string, params interface{}) (*Request, error) {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	return &Request{JSONRPC: JSONRPCVersion, ID: id, Method: method, Params: raw}, nil
}

func NewResponse(id int64, result interface{}) (*Response, error) {
	b, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &Response{JSONRPC: JSONRPCVersion, ID: id, Result: b}, nil
}

func NewErrorResponse(id int64, code int, message string) *Response {
	return &Response{JSONRPC: JSONRPCVersion, ID: id, Error: &RPCError{Code: code, Message: message}}
}

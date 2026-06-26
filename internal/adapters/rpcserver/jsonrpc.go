package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func NewRPCError(code int, message string, data any) *RPCError {
	return &RPCError{Code: code, Message: message, Data: data}
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("json-rpc error %d: %s", e.Code, e.Message)
}

type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
}

type HandlerFunc func(ctx context.Context, params json.RawMessage) (any, error)

type Dispatcher struct {
	handlers map[string]HandlerFunc
	LogError func(error)
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: make(map[string]HandlerFunc)}
}

func (d *Dispatcher) Register(method string, h HandlerFunc) {
	d.handlers[method] = h
}

func (d *Dispatcher) Dispatch(ctx context.Context, raw []byte) []byte {
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return marshalResponse(Response{
			JSONRPC: "2.0",
			ID:      nullID(),
			Error:   NewRPCError(ParseError, "parse error", err.Error()),
		})
	}

	if req.ID == nil {
		if h, ok := d.handlers[req.Method]; ok {
			if _, err := h(ctx, req.Params); err != nil {
				d.logError(err)
			}
		}
		return nil
	}

	if req.JSONRPC != "2.0" {
		return marshalResponse(errorResponse(nullID(), NewRPCError(InvalidRequest, "invalid request", nil)))
	}

	if req.Method == "" {
		return marshalResponse(errorResponse(nullID(), NewRPCError(InvalidRequest, "invalid request", nil)))
	}

	h, ok := d.handlers[req.Method]
	if !ok {
		return marshalResponse(errorResponse(req.ID, NewRPCError(MethodNotFound, "method not found", nil)))
	}

	result, err := h(ctx, req.Params)
	if err != nil {
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) {
			d.logError(err)
			rpcErr = NewRPCError(InternalError, "internal error", nil)
		}
		return marshalResponse(errorResponse(req.ID, rpcErr))
	}

	return marshalResponse(Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	})
}

type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

func NewNotification(method string, params any) ([]byte, error) {
	return json.Marshal(Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
}

func errorResponse(id *json.RawMessage, rpcErr *RPCError) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   rpcErr,
	}
}

func nullID() *json.RawMessage {
	raw := json.RawMessage("null")
	return &raw
}

func (d *Dispatcher) logError(err error) {
	if d.LogError != nil {
		d.LogError(err)
	}
}

func marshalResponse(resp Response) []byte {
	data, err := json.Marshal(resp)
	if err != nil {
		fallback := Response{
			JSONRPC: "2.0",
			ID:      resp.ID,
			Error:   NewRPCError(InternalError, "internal error", err.Error()),
		}
		data, _ = json.Marshal(fallback)
	}
	return data
}

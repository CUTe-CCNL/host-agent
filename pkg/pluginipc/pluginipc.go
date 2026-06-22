package pluginipc

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	methodHealth     = "plugin.health"
	methodHandleHTTP = "plugin.handle_http"
)

type HTTPRequest struct {
	Method   string
	Path     string
	RawQuery string
	Headers  map[string][]string
	Body     []byte
}

type HTTPResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

type Handler interface {
	Health(context.Context) error
	HandleHTTP(context.Context, HTTPRequest) (HTTPResponse, error)
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcHTTPRequest struct {
	Method     string              `json:"method"`
	Path       string              `json:"path"`
	RawQuery   string              `json:"raw_query"`
	Headers    map[string][]string `json:"headers"`
	BodyBase64 string              `json:"body_base64"`
}

type rpcHTTPResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	BodyBase64 string              `json:"body_base64"`
}

func Serve(ctx context.Context, handler Handler) error {
	return ServeStream(ctx, os.Stdin, os.Stdout, handler)
}

func ServeStream(ctx context.Context, input io.Reader, output io.Writer, handler Handler) error {
	if handler == nil {
		return errors.New("pluginipc: handler is nil")
	}

	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	bufferedOutput := bufio.NewWriter(output)
	encoder := json.NewEncoder(bufferedOutput)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		response := handleLine(ctx, scanner.Bytes(), handler)
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("pluginipc: encode response: %w", err)
		}
		if err := bufferedOutput.Flush(); err != nil {
			return fmt.Errorf("pluginipc: flush response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("pluginipc: read request: %w", err)
	}
	return nil
}

func handleLine(ctx context.Context, line []byte, handler Handler) rpcResponse {
	var request rpcRequest
	if err := json.Unmarshal(line, &request); err != nil {
		return rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32700, Message: err.Error()},
		}
	}

	response := rpcResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	switch request.Method {
	case methodHealth:
		if err := handler.Health(ctx); err != nil {
			response.Error = &rpcError{Code: -32000, Message: err.Error()}
			return response
		}
		response.Result = map[string]bool{"ok": true}
		return response

	case methodHandleHTTP:
		httpRequest, err := decodeHTTPRequest(request.Params)
		if err != nil {
			response.Error = &rpcError{Code: -32602, Message: err.Error()}
			return response
		}

		httpResponse, err := handler.HandleHTTP(ctx, httpRequest)
		if err != nil {
			response.Error = &rpcError{Code: -32000, Message: err.Error()}
			return response
		}
		response.Result = encodeHTTPResponse(httpResponse)
		return response

	default:
		response.Error = &rpcError{Code: -32601, Message: "method not found"}
		return response
	}
}

func decodeHTTPRequest(params json.RawMessage) (HTTPRequest, error) {
	var wire rpcHTTPRequest
	if err := json.Unmarshal(params, &wire); err != nil {
		return HTTPRequest{}, fmt.Errorf("decode http params: %w", err)
	}

	body, err := base64.StdEncoding.DecodeString(wire.BodyBase64)
	if err != nil {
		return HTTPRequest{}, fmt.Errorf("decode body_base64: %w", err)
	}

	return HTTPRequest{
		Method:   wire.Method,
		Path:     wire.Path,
		RawQuery: wire.RawQuery,
		Headers:  wire.Headers,
		Body:     body,
	}, nil
}

func encodeHTTPResponse(response HTTPResponse) rpcHTTPResponse {
	return rpcHTTPResponse{
		StatusCode: response.StatusCode,
		Headers:    response.Headers,
		BodyBase64: base64.StdEncoding.EncodeToString(response.Body),
	}
}

package plugin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

var errRPCClientClosed = errors.New("json-rpc client closed")

type stdioRPCClient struct {
	stdout io.ReadCloser
	stdin  io.WriteCloser

	writeMu sync.Mutex

	mu      sync.Mutex
	nextID  int64
	pending map[int64]chan rpcResponse
	closed  bool
	err     error

	closeOnce sync.Once
	onClose   func(error)
}

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return fmt.Sprintf("json-rpc error %d", e.Code)
	}
	return fmt.Sprintf("json-rpc error %d: %s", e.Code, e.Message)
}

func newStdioRPCClient(stdout io.ReadCloser, stdin io.WriteCloser) *stdioRPCClient {
	return &stdioRPCClient{
		stdout:  stdout,
		stdin:   stdin,
		nextID:  1,
		pending: make(map[int64]chan rpcResponse),
	}
}

func (c *stdioRPCClient) start(onClose func(error)) {
	c.onClose = onClose
	go c.readLoop()
}

func (c *stdioRPCClient) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	id, responseCh, err := c.registerCall()
	if err != nil {
		return err
	}

	request := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	if err := c.writeRequest(request); err != nil {
		c.removePending(id)
		return err
	}

	select {
	case response := <-responseCh:
		if response.Error != nil {
			return response.Error
		}
		if result == nil {
			return nil
		}
		if len(response.Result) == 0 {
			return fmt.Errorf("json-rpc response missing result")
		}
		if err := json.Unmarshal(response.Result, result); err != nil {
			return fmt.Errorf("decode json-rpc result: %w", err)
		}
		return nil
	case <-ctx.Done():
		c.removePending(id)
		return ctx.Err()
	}
}

func (c *stdioRPCClient) Close() error {
	c.closeWithError(errRPCClientClosed)
	return nil
}

func (c *stdioRPCClient) registerCall() (int64, chan rpcResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		if c.err != nil {
			return 0, nil, c.err
		}
		return 0, nil, errRPCClientClosed
	}

	id := c.nextID
	c.nextID++
	responseCh := make(chan rpcResponse, 1)
	c.pending[id] = responseCh
	return id, responseCh, nil
}

func (c *stdioRPCClient) writeRequest(request rpcRequest) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := json.NewEncoder(c.stdin).Encode(request); err != nil {
		c.closeWithError(fmt.Errorf("write json-rpc request: %w", err))
		return err
	}
	return nil
}

func (c *stdioRPCClient) removePending(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pending, id)
}

func (c *stdioRPCClient) readLoop() {
	reader := bufio.NewReader(c.stdout)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if handleErr := c.handleLine(line); handleErr != nil {
				c.closeWithError(handleErr)
				return
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.closeWithError(io.EOF)
			} else {
				c.closeWithError(fmt.Errorf("read json-rpc response: %w", err))
			}
			return
		}
	}
}

func (c *stdioRPCClient) handleLine(line []byte) error {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}

	var response rpcResponse
	if err := json.Unmarshal(line, &response); err != nil {
		return fmt.Errorf("decode json-rpc response: %w", err)
	}
	if response.JSONRPC != "2.0" {
		return fmt.Errorf("invalid json-rpc version %q", response.JSONRPC)
	}
	if response.ID == 0 {
		return fmt.Errorf("json-rpc response missing id")
	}

	c.mu.Lock()
	responseCh, ok := c.pending[response.ID]
	if ok {
		delete(c.pending, response.ID)
	}
	c.mu.Unlock()

	if ok {
		responseCh <- response
	}
	return nil
}

func (c *stdioRPCClient) closeWithError(err error) {
	if err == nil {
		err = errRPCClientClosed
	}

	c.closeOnce.Do(func() {
		_ = c.stdin.Close()
		_ = c.stdout.Close()

		c.mu.Lock()
		c.closed = true
		c.err = err
		pending := c.pending
		c.pending = make(map[int64]chan rpcResponse)
		c.mu.Unlock()

		for _, responseCh := range pending {
			responseCh <- rpcResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32000, Message: err.Error()},
			}
		}

		if c.onClose != nil {
			c.onClose(err)
		}
	})
}

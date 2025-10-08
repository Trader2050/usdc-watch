package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"usdc-watch/internal/config"
)

// Client dispatches JSON-RPC requests across a pool of endpoints.
type Client struct {
	endpoints []config.Endpoint
	http      *http.Client

	mu   sync.Mutex
	next int

	callID uint64
}

// NewClient creates a new RPC client using the provided endpoints.
func NewClient(endpoints []config.Endpoint, httpClient *http.Client) (*Client, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("rpc client requires at least one endpoint")
	}
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	return &Client{endpoints: endpoints, http: client}, nil
}

// Call performs the JSON-RPC call, rotating through endpoints until one succeeds.
func (c *Client) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, config.Endpoint, error) {
	if len(c.endpoints) == 0 {
		return nil, config.Endpoint{}, fmt.Errorf("no endpoints configured")
	}
	start := c.nextIndex()
	var errs []string
	for i := 0; i < len(c.endpoints); i++ {
		idx := (start + i) % len(c.endpoints)
		endpoint := c.endpoints[idx]
		result, err := c.callSingle(ctx, endpoint, method, params)
		if err == nil {
			return result, endpoint, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", endpoint.Name, err))
	}
	return nil, config.Endpoint{}, fmt.Errorf("all endpoints failed: %s", strings.Join(errs, "; "))
}

func (c *Client) nextIndex() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	idx := c.next
	c.next = (c.next + 1) % len(c.endpoints)
	return idx
}

func (c *Client) callSingle(ctx context.Context, endpoint config.Endpoint, method string, params interface{}) (json.RawMessage, error) {
	id := atomic.AddUint64(&c.callID, 1)
	payload := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, buf)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rpcResp jsonRPCResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      uint64      `json:"id"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error"`
}

type jsonRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

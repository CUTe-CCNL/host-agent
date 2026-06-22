package plugin

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"host-agent/config"
)

type Status string

const (
	TransportStdioJSONRPC = "stdio-jsonrpc"

	StatusDisabled  Status = "disabled"
	StatusStopped   Status = "stopped"
	StatusStarting  Status = "starting"
	StatusRunning   Status = "running"
	StatusUnhealthy Status = "unhealthy"
	StatusFailed    Status = "failed"
)

type Info struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description,omitempty"`
	Status      Status    `json:"status"`
	Error       string    `json:"error,omitempty"`
	Transport   string    `json:"transport"`
	Routes      []Route   `json:"routes,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Registry struct {
	cfg     config.PluginConfig
	mu      sync.RWMutex
	plugins map[string]*instance
}

type instance struct {
	manifest     Manifest
	status       Status
	lastError    string
	updatedAt    time.Time
	cmd          *exec.Cmd
	waitDone     chan struct{}
	rpc          *stdioRPCClient
	healthCancel context.CancelFunc
}

type pluginHealthResult struct {
	OK bool `json:"ok"`
}

type pluginHTTPRequest struct {
	Method     string              `json:"method"`
	Path       string              `json:"path"`
	RawQuery   string              `json:"raw_query"`
	Headers    map[string][]string `json:"headers"`
	BodyBase64 string              `json:"body_base64"`
}

type pluginHTTPResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	BodyBase64 string              `json:"body_base64"`
}

func NewRegistry(cfg config.PluginConfig) *Registry {
	if cfg.StartupTimeout <= 0 {
		cfg.StartupTimeout = 10 * time.Second
	}
	if cfg.HealthInterval <= 0 {
		cfg.HealthInterval = 15 * time.Second
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 30 * time.Second
	}

	return &Registry{
		cfg:     cfg,
		plugins: make(map[string]*instance),
	}
}

func (r *Registry) Register(manifests []Manifest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range manifests {
		manifest := manifests[i]
		if err := validateManifest(&manifest); err != nil {
			return err
		}
		if _, exists := r.plugins[manifest.ID]; exists {
			return fmt.Errorf("duplicate plugin id %q", manifest.ID)
		}

		status := StatusStopped
		if !manifest.Enabled {
			status = StatusDisabled
		}
		r.plugins[manifest.ID] = &instance{
			manifest:  manifest,
			status:    status,
			updatedAt: time.Now(),
		}
	}

	return nil
}

func (r *Registry) Load(directory string) error {
	manifests, err := LoadManifests(directory)
	if err != nil {
		return err
	}
	return r.Register(manifests)
}

func (r *Registry) Start(ctx context.Context) error {
	ids := r.ids()
	for _, id := range ids {
		info, ok := r.Get(id)
		if !ok || !info.Enabled {
			continue
		}
		if err := r.startPlugin(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) Stop(ctx context.Context) error {
	ids := r.ids()
	var firstErr error
	for _, id := range ids {
		if err := r.stopPlugin(ctx, id, StatusStopped, ""); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (r *Registry) Restart(ctx context.Context, id string) error {
	info, ok := r.Get(id)
	if !ok {
		return fmt.Errorf("plugin %q not found", id)
	}
	if !info.Enabled {
		return fmt.Errorf("plugin %q is disabled", id)
	}

	if err := r.stopPlugin(ctx, id, StatusStopped, ""); err != nil {
		return err
	}
	return r.startPlugin(ctx, id)
}

func (r *Registry) List() []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]Info, 0, len(r.plugins))
	for _, inst := range r.plugins {
		infos = append(infos, infoFromInstance(inst))
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})
	return infos
}

func (r *Registry) Get(id string) (Info, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inst, ok := r.plugins[id]
	if !ok {
		return Info{}, false
	}
	return infoFromInstance(inst), true
}

func (r *Registry) ProxyHTTP(w http.ResponseWriter, req *http.Request, id, requestPath string) {
	manifest, status, rpcClient, ok := r.snapshot(id)
	if !ok {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}
	if status != StatusRunning || rpcClient == nil {
		http.Error(w, "plugin is not running", http.StatusServiceUnavailable)
		return
	}

	requestPath = normalizeRequestPath(requestPath)
	if !manifestAllows(manifest, requestPath, req.Method) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadGateway)
		return
	}

	timeoutCtx, cancel := context.WithTimeout(req.Context(), r.cfg.RequestTimeout)
	defer cancel()

	headers := req.Header.Clone()
	removeHopByHopHeaders(headers)
	params := pluginHTTPRequest{
		Method:     req.Method,
		Path:       requestPath,
		RawQuery:   req.URL.RawQuery,
		Headers:    map[string][]string(headers),
		BodyBase64: base64.StdEncoding.EncodeToString(body),
	}

	var response pluginHTTPResponse
	if err := rpcClient.Call(timeoutCtx, "plugin.handle_http", params, &response); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			http.Error(w, "plugin timeout", http.StatusGatewayTimeout)
			return
		}
		http.Error(w, "plugin error", http.StatusBadGateway)
		return
	}

	if response.StatusCode < 100 || response.StatusCode > 599 {
		http.Error(w, "invalid plugin response", http.StatusBadGateway)
		return
	}

	responseBody, err := base64.StdEncoding.DecodeString(response.BodyBase64)
	if err != nil {
		http.Error(w, "invalid plugin response", http.StatusBadGateway)
		return
	}

	copyHeader(w.Header(), http.Header(response.Headers))
	removeHopByHopHeaders(w.Header())
	w.WriteHeader(response.StatusCode)
	if _, err := w.Write(responseBody); err != nil {
		return
	}
}

func (r *Registry) ids() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.plugins))
	for id := range r.plugins {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r *Registry) startPlugin(ctx context.Context, id string) error {
	r.mu.Lock()
	inst, ok := r.plugins[id]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("plugin %q not found", id)
	}
	if !inst.manifest.Enabled {
		r.mu.Unlock()
		return fmt.Errorf("plugin %q is disabled", id)
	}
	if inst.healthCancel != nil {
		inst.healthCancel()
		inst.healthCancel = nil
	}

	manifest := inst.manifest
	inst.status = StatusStarting
	inst.lastError = ""
	inst.updatedAt = time.Now()
	r.mu.Unlock()

	cmd := exec.Command(manifest.Command, manifest.Args...)
	if manifest.WorkingDir != "" {
		cmd.Dir = manifest.WorkingDir
	}
	cmd.Env = mergeEnv(os.Environ(), manifest.Env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		r.setStatus(id, StatusFailed, err.Error())
		return fmt.Errorf("open plugin %q stdin: %w", id, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.setStatus(id, StatusFailed, err.Error())
		return fmt.Errorf("open plugin %q stdout: %w", id, err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		r.setStatus(id, StatusFailed, err.Error())
		return fmt.Errorf("start plugin %q: %w", id, err)
	}

	rpcClient := newStdioRPCClient(stdout, stdin)
	rpcClient.start(func(err error) {
		r.handleIPCClosed(id, rpcClient, err)
	})

	done := make(chan struct{})
	r.mu.Lock()
	inst, ok = r.plugins[id]
	if !ok {
		r.mu.Unlock()
		_ = rpcClient.Close()
		_ = cmd.Process.Kill()
		return fmt.Errorf("plugin %q not found", id)
	}
	inst.cmd = cmd
	inst.waitDone = done
	inst.rpc = rpcClient
	inst.updatedAt = time.Now()
	r.mu.Unlock()

	go r.watchProcess(id, cmd, done)

	startupCtx, cancel := context.WithTimeout(ctx, r.cfg.StartupTimeout)
	defer cancel()
	if err := r.waitForHealth(startupCtx, rpcClient, done); err != nil {
		r.setStatus(id, StatusFailed, err.Error())
		_ = rpcClient.Close()
		_ = cmd.Process.Kill()
		return fmt.Errorf("plugin %q health check failed: %w", id, err)
	}

	select {
	case <-done:
		return fmt.Errorf("plugin %q exited during startup", id)
	default:
	}

	healthCtx, healthCancel := context.WithCancel(context.Background())
	started := false
	r.mu.Lock()
	inst, ok = r.plugins[id]
	if ok && inst.cmd == cmd && inst.status == StatusStarting {
		inst.status = StatusRunning
		inst.lastError = ""
		inst.updatedAt = time.Now()
		inst.healthCancel = healthCancel
		started = true
	}
	r.mu.Unlock()

	if !started {
		healthCancel()
		return fmt.Errorf("plugin %q exited during startup", id)
	}

	go r.healthLoop(healthCtx, id, rpcClient)
	return nil
}

func (r *Registry) stopPlugin(ctx context.Context, id string, status Status, message string) error {
	r.mu.Lock()
	inst, ok := r.plugins[id]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("plugin %q not found", id)
	}
	if inst.healthCancel != nil {
		inst.healthCancel()
		inst.healthCancel = nil
	}
	if !inst.manifest.Enabled && status == StatusStopped {
		status = StatusDisabled
	}

	cmd := inst.cmd
	done := inst.waitDone
	rpcClient := inst.rpc
	inst.cmd = nil
	inst.waitDone = nil
	inst.rpc = nil
	inst.status = status
	inst.lastError = message
	inst.updatedAt = time.Now()
	r.mu.Unlock()

	if rpcClient != nil {
		_ = rpcClient.Close()
	}

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if cmd.ProcessState == nil {
		_ = cmd.Process.Kill()
	}
	if done == nil {
		return nil
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Registry) watchProcess(id string, cmd *exec.Cmd, done chan struct{}) {
	err := cmd.Wait()
	close(done)

	r.mu.Lock()
	inst, ok := r.plugins[id]
	if !ok || inst.cmd != cmd {
		r.mu.Unlock()
		return
	}
	if inst.status == StatusStopped || inst.status == StatusDisabled {
		r.mu.Unlock()
		return
	}

	rpcClient := inst.rpc
	inst.rpc = nil
	inst.status = StatusFailed
	inst.updatedAt = time.Now()
	if inst.lastError == "" {
		if err != nil {
			inst.lastError = err.Error()
		} else {
			inst.lastError = "process exited"
		}
	}
	r.mu.Unlock()

	if rpcClient != nil {
		_ = rpcClient.Close()
	}
}

func (r *Registry) waitForHealth(ctx context.Context, rpcClient *stdioRPCClient, done <-chan struct{}) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := r.checkHealth(ctx, rpcClient); err == nil {
			return nil
		}

		select {
		case <-done:
			return fmt.Errorf("process exited during health check")
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *Registry) healthLoop(ctx context.Context, id string, rpcClient *stdioRPCClient) {
	ticker := time.NewTicker(r.cfg.HealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.checkHealth(ctx, rpcClient); err != nil {
				r.setHealthStatus(id, rpcClient, StatusUnhealthy, err.Error())
			} else {
				r.setHealthStatus(id, rpcClient, StatusRunning, "")
			}
		}
	}
}

func (r *Registry) checkHealth(ctx context.Context, rpcClient *stdioRPCClient) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, r.cfg.RequestTimeout)
	defer cancel()

	var result pluginHealthResult
	if err := rpcClient.Call(timeoutCtx, "plugin.health", map[string]interface{}{}, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("plugin health returned ok=false")
	}
	return nil
}

func (r *Registry) setStatus(id string, status Status, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, ok := r.plugins[id]
	if !ok {
		return
	}
	inst.status = status
	inst.lastError = message
	inst.updatedAt = time.Now()
}

func (r *Registry) setHealthStatus(id string, rpcClient *stdioRPCClient, status Status, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, ok := r.plugins[id]
	if !ok {
		return
	}
	if inst.rpc != rpcClient {
		return
	}
	if inst.status != StatusRunning && inst.status != StatusUnhealthy {
		return
	}
	inst.status = status
	inst.lastError = message
	inst.updatedAt = time.Now()
}

func (r *Registry) handleIPCClosed(id string, rpcClient *stdioRPCClient, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, ok := r.plugins[id]
	if !ok || inst.rpc != rpcClient {
		return
	}
	if inst.status == StatusStopped || inst.status == StatusDisabled {
		return
	}

	inst.rpc = nil
	inst.status = StatusFailed
	inst.updatedAt = time.Now()
	if inst.lastError != "" {
		return
	}
	if err != nil {
		inst.lastError = fmt.Sprintf("ipc closed: %v", err)
	} else {
		inst.lastError = "ipc closed"
	}
}

func (r *Registry) snapshot(id string) (Manifest, Status, *stdioRPCClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inst, ok := r.plugins[id]
	if !ok {
		return Manifest{}, "", nil, false
	}
	return inst.manifest, inst.status, inst.rpc, true
}

func infoFromInstance(inst *instance) Info {
	return Info{
		ID:          inst.manifest.ID,
		Name:        inst.manifest.Name,
		Version:     inst.manifest.Version,
		Enabled:     inst.manifest.Enabled,
		Description: inst.manifest.Description,
		Status:      inst.status,
		Error:       inst.lastError,
		Transport:   TransportStdioJSONRPC,
		Routes:      append([]Route(nil), inst.manifest.Routes...),
		UpdatedAt:   inst.updatedAt,
	}
}

func mergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}

	env := append([]string(nil), base...)
	keys := make([]string, 0, len(extra))
	for key := range extra {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		env = append(env, key+"="+extra[key])
	}
	return env
}

func normalizeRequestPath(requestPath string) string {
	if requestPath == "" {
		return "/"
	}
	if !strings.HasPrefix(requestPath, "/") {
		return "/" + requestPath
	}
	return requestPath
}

func manifestAllows(manifest Manifest, requestPath, method string) bool {
	method = strings.ToUpper(method)
	for _, route := range manifest.Routes {
		if !pathMatchesPrefix(requestPath, route.PathPrefix) {
			continue
		}
		for _, allowed := range route.Methods {
			if strings.ToUpper(allowed) == method {
				return true
			}
		}
	}
	return false
}

func pathMatchesPrefix(path, prefix string) bool {
	if prefix == "/" {
		return true
	}
	prefix = strings.TrimRight(prefix, "/")
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func removeHopByHopHeaders(header http.Header) {
	for _, key := range []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		header.Del(key)
	}
}

package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	UpstreamURL string    `json:"upstream_url"`
	Routes      []Route   `json:"routes,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Registry struct {
	cfg     config.PluginConfig
	client  *http.Client
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
	healthCancel context.CancelFunc
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
		client:  &http.Client{},
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
	manifest, status, ok := r.snapshot(id)
	if !ok {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}
	if status != StatusRunning {
		http.Error(w, "plugin is not running", http.StatusServiceUnavailable)
		return
	}

	requestPath = normalizeRequestPath(requestPath)
	if !manifestAllows(manifest, requestPath, req.Method) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	target, err := buildProxyURL(manifest.UpstreamURL, requestPath, req.URL.RawQuery)
	if err != nil {
		http.Error(w, "invalid upstream", http.StatusBadGateway)
		return
	}

	timeoutCtx, cancel := context.WithTimeout(req.Context(), r.cfg.RequestTimeout)
	defer cancel()

	upstreamReq, err := http.NewRequestWithContext(timeoutCtx, req.Method, target.String(), req.Body)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadGateway)
		return
	}
	upstreamReq.Header = req.Header.Clone()
	removeHopByHopHeaders(upstreamReq.Header)

	resp, err := r.client.Do(upstreamReq)
	if err != nil {
		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			http.Error(w, "upstream timeout", http.StatusGatewayTimeout)
			return
		}
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	copyHeader(w.Header(), resp.Header)
	removeHopByHopHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
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
	if err := cmd.Start(); err != nil {
		r.setStatus(id, StatusFailed, err.Error())
		return fmt.Errorf("start plugin %q: %w", id, err)
	}

	done := make(chan struct{})
	r.mu.Lock()
	inst, ok = r.plugins[id]
	if !ok {
		r.mu.Unlock()
		_ = cmd.Process.Kill()
		return fmt.Errorf("plugin %q not found", id)
	}
	inst.cmd = cmd
	inst.waitDone = done
	inst.updatedAt = time.Now()
	r.mu.Unlock()

	go r.watchProcess(id, cmd, done)

	startupCtx, cancel := context.WithTimeout(ctx, r.cfg.StartupTimeout)
	defer cancel()
	if err := r.waitForHealth(startupCtx, manifest); err != nil {
		r.setStatus(id, StatusFailed, err.Error())
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

	go r.healthLoop(healthCtx, id, manifest)
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
	inst.cmd = nil
	inst.waitDone = nil
	inst.status = status
	inst.lastError = message
	inst.updatedAt = time.Now()
	r.mu.Unlock()

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
	defer r.mu.Unlock()

	inst, ok := r.plugins[id]
	if !ok || inst.cmd != cmd {
		return
	}
	if inst.status == StatusStopped || inst.status == StatusDisabled {
		return
	}

	inst.status = StatusFailed
	inst.updatedAt = time.Now()
	if err != nil {
		inst.lastError = err.Error()
	} else {
		inst.lastError = "process exited"
	}
}

func (r *Registry) waitForHealth(ctx context.Context, manifest Manifest) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := r.checkHealth(ctx, manifest); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *Registry) healthLoop(ctx context.Context, id string, manifest Manifest) {
	ticker := time.NewTicker(r.cfg.HealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.checkHealth(ctx, manifest); err != nil {
				r.setHealthStatus(id, StatusUnhealthy, err.Error())
			} else {
				r.setHealthStatus(id, StatusRunning, "")
			}
		}
	}
}

func (r *Registry) checkHealth(ctx context.Context, manifest Manifest) error {
	healthURL, err := buildProxyURL(manifest.UpstreamURL, manifest.HealthPath, "")
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL.String(), nil)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health returned status %d", resp.StatusCode)
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

func (r *Registry) setHealthStatus(id string, status Status, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, ok := r.plugins[id]
	if !ok {
		return
	}
	if inst.status != StatusRunning && inst.status != StatusUnhealthy {
		return
	}
	inst.status = status
	inst.lastError = message
	inst.updatedAt = time.Now()
}

func (r *Registry) snapshot(id string) (Manifest, Status, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inst, ok := r.plugins[id]
	if !ok {
		return Manifest{}, "", false
	}
	return inst.manifest, inst.status, true
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
		UpstreamURL: inst.manifest.UpstreamURL,
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

func buildProxyURL(upstream, requestPath, rawQuery string) (*url.URL, error) {
	parsed, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}

	requestPath = normalizeRequestPath(requestPath)
	basePath := strings.TrimRight(parsed.Path, "/")
	if basePath == "" {
		parsed.Path = requestPath
	} else {
		parsed.Path = basePath + requestPath
	}
	parsed.RawQuery = rawQuery
	return parsed, nil
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

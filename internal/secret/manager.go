package secret

import (
	"context"
	"fmt"
	"sync"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// Manager provides a thin abstraction over Google Cloud Secret Manager.
// It caches secret values in memory for a configurable TTL to reduce
// latency and API costs while still allowing timely rotation.
type Manager struct {
	client    *secretmanager.Client
	projectID string
	ttl       time.Duration

	mu    sync.RWMutex
	cache map[string]cachedSecret
}

// cachedSecret holds a secret value together with its expiration time.
type cachedSecret struct {
	value   string
	expires time.Time
}

// NewManager creates a new Manager.
//   ctx        – context for the underlying GCP client.
//   projectID  – GCP project where the secrets reside.
//   ttl        – cache TTL; if <=0 a default of 5 minutes is used.
//   clientOpts – optional client options (e.g., credentials).
func NewManager(ctx context.Context, projectID string, ttl time.Duration, clientOpts ...option.ClientOption) (*Manager, error) {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	c, err := secretmanager.NewClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("secretmanager.NewClient: %w", err)
	}
	m := &Manager{
		client:    c,
		projectID: projectID,
		ttl:       ttl,
		cache:     make(map[string]cachedSecret),
	}
	return m, nil
}

// Get retrieves the secret value identified by name.
// The name should be the secret identifier as created in Secret Manager
// (e.g., "my-api-key"). The latest version is always fetched.
// Results are cached for the Manager's TTL.
func (m *Manager) Get(ctx context.Context, name string) (string, error) {
	// Fast path: return cached value if still valid.
	m.mu.RLock()
	if cs, ok := m.cache[name]; ok && time.Now().Before(cs.expires) {
		m.mu.RUnlock()
		return cs.value, nil
	}
	m.mu.RUnlock()

	// Build the fully‑qualified secret version name.
	secretPath := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", m.projectID, name)
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretPath,
	}

	resp, err := m.client.AccessSecretVersion(ctx, req)
	if err != nil {
		zap.L().Error("failed to access secret version",
			zap.String("secret", name),
			zap.Error(err),
		)
		return "", fmt.Errorf("access secret %s: %w", name, err)
	}
	value := string(resp.Payload.Data)

	// Store the fresh value in the cache.
	m.mu.Lock()
	m.cache[name] = cachedSecret{
		value:   value,
		expires: time.Now().Add(m.ttl),
	}
	m.mu.Unlock()

	return value, nil
}

// Invalidate removes a secret from the cache, forcing the next Get call
// to fetch a fresh copy from Secret Manager.
func (m *Manager) Invalidate(name string) {
	m.mu.Lock()
	delete(m.cache, name)
	m.mu.Unlock()
}

// SetTTL updates the cache TTL for future entries.
// Existing cached entries keep their original expiration.
func (m *Manager) SetTTL(d time.Duration) {
	if d <= 0 {
		return
	}
	m.mu.Lock()
	m.ttl = d
	m.mu.Unlock()
}

// Close releases the underlying Secret Manager client.
func (m *Manager) Close() error {
	if m.client == nil {
		return nil
	}
	return m.client.Close()
}
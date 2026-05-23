package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/me-nazim/criminal-archive/backend/internal/settings"
)

// Driver is a stable identifier for a storage backend. All current
// drivers are S3-compatible — they only differ in defaults (path-style,
// region, public URL handling).
type Driver string

const (
	DriverR2           Driver = "r2"
	DriverAWSS3        Driver = "aws_s3"
	DriverMinIO        Driver = "minio"
	DriverS3Compatible Driver = "s3_compatible"
)

// ErrUnconfigured is returned by the manager when the admin hasn't yet
// supplied a working storage configuration.
var ErrUnconfigured = errors.New("storage: not configured")

// Manager owns the live *Client and refreshes it when settings change.
type Manager struct {
	store  *settings.Store
	logger interface{ Warn(string, ...any) }

	mu         sync.RWMutex
	client     *Client
	cfgVersion uint64
	cfg        settings.StorageConfig
}

// NewManager constructs a Manager. logger is optional.
func NewManager(store *settings.Store, logger interface{ Warn(string, ...any) }) *Manager {
	return &Manager{store: store, logger: logger}
}

// Refresh re-reads the storage config and rebuilds the underlying client.
func (m *Manager) Refresh(ctx context.Context) error {
	cfg, err := m.store.GetStorage(ctx)
	if err != nil && !errors.Is(err, settings.ErrNotFound) {
		return err
	}
	c, buildErr := buildClient(ctx, cfg)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = cfg
	m.cfgVersion = m.store.Version()
	if buildErr != nil {
		m.client = nil
		return buildErr
	}
	m.client = c
	return nil
}

// Client returns the live storage client. If settings have changed since
// the last refresh, a transparent rebuild is attempted.
func (m *Manager) Client(ctx context.Context) (*Client, error) {
	m.mu.RLock()
	cur := m.cfgVersion
	live := m.store.Version()
	m.mu.RUnlock()
	if cur != live {
		_ = m.Refresh(ctx)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.client == nil {
		return nil, ErrUnconfigured
	}
	return m.client, nil
}

// Config returns the active config (without secrets re-encrypted).
func (m *Manager) Config() settings.StorageConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

// TestStorage satisfies settings.ProviderTester. We probe by building an
// ad-hoc client and calling HeadBucket — non-existent bucket and bad
// credentials surface as different errors but both fail this check.
func (m *Manager) TestStorage(_ *http.Request, cfg settings.StorageConfig) error {
	cfg.Enabled = true
	c, err := buildClient(context.Background(), cfg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.EnsureBucket(ctx); err != nil {
		return fmt.Errorf("bucket access failed: %w", err)
	}
	return nil
}

// buildClient applies driver-specific defaults then constructs a *Client.
func buildClient(ctx context.Context, cfg settings.StorageConfig) (*Client, error) {
	if !cfg.Enabled {
		return nil, ErrUnconfigured
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		return nil, ErrUnconfigured
	}
	driver := Driver(strings.ToLower(strings.TrimSpace(cfg.Driver)))
	if driver == "" {
		driver = DriverS3Compatible
	}
	switch driver {
	case DriverR2:
		if cfg.Region == "" {
			cfg.Region = "auto"
		}
		// R2 honours virtual-hosted style; force_path_style off by default.
	case DriverAWSS3:
		if cfg.Region == "" {
			cfg.Region = "us-east-1"
		}
		if cfg.Endpoint == "" {
			// Empty endpoint = official AWS endpoint resolution
		}
	case DriverMinIO:
		if cfg.Region == "" {
			cfg.Region = "us-east-1"
		}
		cfg.ForcePathStyle = true
	case DriverS3Compatible:
		if cfg.Region == "" {
			cfg.Region = "auto"
		}
	default:
		return nil, fmt.Errorf("storage: unknown driver %q", driver)
	}
	return NewClient(ctx, Config{
		Endpoint:       cfg.Endpoint,
		Region:         cfg.Region,
		AccessKey:      cfg.AccessKey,
		SecretKey:      cfg.SecretKey,
		Bucket:         cfg.Bucket,
		PublicBaseURL:  cfg.PublicBaseURL,
		ForcePathStyle: cfg.ForcePathStyle,
	})
}

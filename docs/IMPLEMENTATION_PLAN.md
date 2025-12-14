# Implementation Plan: Competitive Feature Improvements

*Created: December 2025*

This plan outlines the implementation of three key improvements identified in the competitive analysis. OAuth2 and BTP integration are excluded as the primary use case is on-premise SAP systems (see [SOLMAN_HUB_ARCHITECTURE.md](SOLMAN_HUB_ARCHITECTURE.md) for future cloud considerations).

---

## Executive Summary

| Feature | Priority | Effort | Impact | Target Version |
|---------|----------|--------|--------|----------------|
| Credential Masking | HIGH | Small | Security hygiene | v1.6.0 ✅ |
| Exponential Backoff Retry | HIGH | Small | Improved reliability | v1.6.0 ✅ |
| Token-Optimized Discovery | HIGH | Large | ~90% token reduction | v1.7.0 |
| Skill Generator | MEDIUM | Medium | AI-native workflows | v1.8.0 |

**Implementation Order:**

1. ✅ Credential Masking (foundation for safe logging)
2. ✅ Exponential Backoff Retry (quick win, improves reliability)
3. Token-Optimized Discovery (largest impact, most complex)
4. Skill Generator (AI-native documentation & workflows)

---

## Phase 1: Foundation (v1.6.0)

### 1.1 Credential Masking in Logs

**Goal:** Prevent accidental credential exposure in verbose output.

#### New Files
```
internal/
  debug/
    masking.go       # Core masking logic
    masking_test.go  # Unit tests
```

#### Implementation

**`internal/debug/masking.go`:**
```go
package debug

import (
    "net/url"
    "strings"
)

// SensitiveKeys that trigger automatic masking
var SensitiveKeys = []string{
    "password", "passwd", "pwd", "secret",
    "token", "api_key", "apikey", "api-key",
    "authorization", "auth", "credential",
    "x-csrf-token", "csrf",
}

// MaskValue masks a sensitive value, showing only last N chars
func MaskValue(value string, showLastChars int) string {
    if len(value) == 0 {
        return ""
    }
    if len(value) <= showLastChars {
        return strings.Repeat("*", len(value))
    }
    return strings.Repeat("*", len(value)-showLastChars) + value[len(value)-showLastChars:]
}

// MaskPassword completely masks a password
func MaskPassword(password string) string {
    if len(password) == 0 {
        return ""
    }
    return "***"
}

// MaskToken shows token type and last 8 characters
func MaskToken(token string) string {
    if len(token) == 0 {
        return ""
    }
    if len(token) <= 8 {
        return "****"
    }
    return "****" + token[len(token)-8:]
}

// MaskURL removes sensitive query parameters from URL
func MaskURL(rawURL string) string {
    parsed, err := url.Parse(rawURL)
    if err != nil {
        return rawURL
    }

    // Mask userinfo (user:password@host)
    if parsed.User != nil {
        if _, hasPass := parsed.User.Password(); hasPass {
            parsed.User = url.UserPassword(parsed.User.Username(), "***")
        }
    }

    // Mask sensitive query parameters
    query := parsed.Query()
    for key := range query {
        if isSensitiveKey(key) {
            query.Set(key, "***")
        }
    }
    parsed.RawQuery = query.Encode()

    return parsed.String()
}

// MaskHeader masks Authorization and other sensitive headers
func MaskHeader(name, value string) string {
    nameLower := strings.ToLower(name)
    if nameLower == "authorization" {
        parts := strings.SplitN(value, " ", 2)
        if len(parts) == 2 {
            return parts[0] + " " + MaskToken(parts[1])
        }
        return MaskToken(value)
    }
    if isSensitiveKey(nameLower) {
        return MaskToken(value)
    }
    return value
}

func isSensitiveKey(key string) bool {
    keyLower := strings.ToLower(key)
    for _, sensitive := range SensitiveKeys {
        if strings.Contains(keyLower, sensitive) {
            return true
        }
    }
    return false
}
```

#### Files to Modify

**`internal/client/client.go`:**
- Import `internal/debug`
- Replace direct URL logging with `debug.MaskURL()`
- Replace header logging with `debug.MaskHeader()`
- Update CSRF token logging to use consistent masking

**`cmd/odata-mcp/main.go`:**
- Line ~486: Change `fmt.Fprintf(os.Stderr, "[VERBOSE] Using basic authentication for user: %s\n", cfg.Username)` to omit password confirmation
- Add masking helper for any auth-related verbose output

#### CLI Flags
None required — masking is always enabled.

#### Tests
```go
func TestMaskPassword(t *testing.T) {
    assert.Equal(t, "***", MaskPassword("secret123"))
    assert.Equal(t, "", MaskPassword(""))
}

func TestMaskURL(t *testing.T) {
    masked := MaskURL("https://user:pass@host.com/path?token=abc123")
    assert.NotContains(t, masked, "pass")
    assert.NotContains(t, masked, "abc123")
}

func TestMaskToken(t *testing.T) {
    assert.Equal(t, "****abcd1234", MaskToken("verylongtokenabcd1234"))
    assert.Equal(t, "****", MaskToken("short"))
}

func TestMaskHeader(t *testing.T) {
    masked := MaskHeader("Authorization", "Basic dXNlcjpwYXNz")
    assert.Equal(t, "Basic ****cjpwYXNz", masked)
}
```

---

### 1.2 Exponential Backoff Retry

**Goal:** Improve reliability for flaky SAP services with configurable retry logic.

#### New Files
```
internal/
  client/
    retry.go       # Retry configuration and logic
    retry_test.go  # Unit tests
```

#### Implementation

**`internal/client/retry.go`:**
```go
package client

import (
    "math"
    "math/rand"
    "net/http"
    "strings"
    "time"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
    MaxRetries         int           // Maximum number of retry attempts
    InitialBackoff     time.Duration // Initial delay before first retry
    MaxBackoff         time.Duration // Maximum delay between retries
    BackoffMultiplier  float64       // Multiplier for exponential backoff
    JitterFraction     float64       // Random jitter (0.0-1.0)
    RetryableStatuses  []int         // HTTP status codes that trigger retry
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() *RetryConfig {
    return &RetryConfig{
        MaxRetries:        3,
        InitialBackoff:    100 * time.Millisecond,
        MaxBackoff:        10 * time.Second,
        BackoffMultiplier: 2.0,
        JitterFraction:    0.1,
        RetryableStatuses: []int{429, 500, 502, 503, 504},
    }
}

// CalculateBackoff returns the delay for a given attempt (0-indexed)
func (c *RetryConfig) CalculateBackoff(attempt int) time.Duration {
    if attempt <= 0 {
        return c.InitialBackoff
    }

    // Exponential backoff: initial * multiplier^attempt
    backoff := float64(c.InitialBackoff) * math.Pow(c.BackoffMultiplier, float64(attempt))

    // Cap at max backoff
    if backoff > float64(c.MaxBackoff) {
        backoff = float64(c.MaxBackoff)
    }

    // Apply jitter to prevent thundering herd
    if c.JitterFraction > 0 {
        jitter := backoff * c.JitterFraction * (rand.Float64()*2 - 1) // ±jitter
        backoff += jitter
    }

    return time.Duration(backoff)
}

// ShouldRetry determines if a request should be retried
func (c *RetryConfig) ShouldRetry(statusCode int, attempt int) bool {
    if attempt >= c.MaxRetries {
        return false
    }

    for _, code := range c.RetryableStatuses {
        if statusCode == code {
            return true
        }
    }

    return false
}

// IsCSRFFailure checks if the response indicates CSRF token failure
func IsCSRFFailure(resp *http.Response, body []byte) bool {
    if resp.StatusCode != 403 {
        return false
    }
    // Check for SAP-specific CSRF error messages
    bodyStr := string(body)
    return strings.Contains(bodyStr, "CSRF") ||
           strings.Contains(bodyStr, "csrf") ||
           strings.Contains(bodyStr, "token validation failed")
}
```

#### Modify `internal/client/client.go`

Add retry config to ODataClient struct:
```go
type ODataClient struct {
    // ... existing fields ...
    retryConfig *RetryConfig
}

func NewODataClient(cfg *config.Config) *ODataClient {
    client := &ODataClient{
        // ... existing initialization ...
        retryConfig: DefaultRetryConfig(),
    }

    // Apply config overrides
    if cfg.RetryMaxAttempts > 0 {
        client.retryConfig.MaxRetries = cfg.RetryMaxAttempts
    }
    if cfg.RetryInitialBackoffMs > 0 {
        client.retryConfig.InitialBackoff = time.Duration(cfg.RetryInitialBackoffMs) * time.Millisecond
    }
    if cfg.RetryMaxBackoffMs > 0 {
        client.retryConfig.MaxBackoff = time.Duration(cfg.RetryMaxBackoffMs) * time.Millisecond
    }
    if cfg.RetryBackoffMultiplier > 0 {
        client.retryConfig.BackoffMultiplier = cfg.RetryBackoffMultiplier
    }

    return client
}
```

Replace the current `doRequestWithRetry` with:

```go
func (c *ODataClient) doRequestWithRetry(ctx context.Context, method, url string, body io.Reader, contentType string) (*http.Response, []byte, error) {
    var lastErr error
    var lastResp *http.Response
    var lastBody []byte
    var bodyBytes []byte

    // Read body once if present (needed for retries)
    if body != nil {
        var err error
        bodyBytes, err = io.ReadAll(body)
        if err != nil {
            return nil, nil, fmt.Errorf("failed to read request body: %w", err)
        }
    }

    for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
        // Wait before retry (skip first attempt)
        if attempt > 0 {
            backoff := c.retryConfig.CalculateBackoff(attempt - 1)
            if c.verbose {
                fmt.Fprintf(os.Stderr, "[VERBOSE] Retry attempt %d/%d after %v\n",
                    attempt, c.retryConfig.MaxRetries, backoff)
            }
            select {
            case <-ctx.Done():
                return nil, nil, ctx.Err()
            case <-time.After(backoff):
            }
        }

        // Build request with fresh body reader
        var bodyReader io.Reader
        if bodyBytes != nil {
            bodyReader = bytes.NewReader(bodyBytes)
        }

        req, err := c.buildRequest(ctx, method, url, bodyReader, contentType)
        if err != nil {
            return nil, nil, err
        }

        resp, respBody, err := c.doRequest(req)
        if err != nil {
            lastErr = err
            if c.verbose {
                fmt.Fprintf(os.Stderr, "[VERBOSE] Request failed: %v\n", err)
            }
            continue // Network error, retry
        }

        lastResp = resp
        lastBody = respBody

        // Check for CSRF failure (special handling)
        if IsCSRFFailure(resp, respBody) && !c.csrfRetried {
            c.csrfRetried = true
            c.csrfToken = "" // Clear token to fetch new one
            if c.verbose {
                fmt.Fprintf(os.Stderr, "[VERBOSE] CSRF token expired, fetching new token\n")
            }
            continue
        }

        // Check if we should retry based on status code
        if c.retryConfig.ShouldRetry(resp.StatusCode, attempt) {
            if c.verbose {
                fmt.Fprintf(os.Stderr, "[VERBOSE] Received status %d, will retry\n", resp.StatusCode)
            }
            continue
        }

        // Success or non-retryable error
        c.csrfRetried = false
        return resp, respBody, nil
    }

    // All retries exhausted
    if lastResp != nil {
        return lastResp, lastBody, nil
    }
    return nil, nil, fmt.Errorf("all %d retries failed: %w", c.retryConfig.MaxRetries, lastErr)
}
```

#### Config Changes

**`internal/config/config.go`:**
```go
// Add to Config struct
RetryMaxAttempts       int     `mapstructure:"retry_max_attempts"`
RetryInitialBackoffMs  int     `mapstructure:"retry_initial_backoff_ms"`
RetryMaxBackoffMs      int     `mapstructure:"retry_max_backoff_ms"`
RetryBackoffMultiplier float64 `mapstructure:"retry_backoff_multiplier"`
```

#### CLI Flags

**`cmd/odata-mcp/main.go`:**
```go
// Retry configuration
rootCmd.Flags().IntVar(&cfg.RetryMaxAttempts, "retry-max-attempts", 3,
    "Maximum retry attempts for failed requests")
rootCmd.Flags().IntVar(&cfg.RetryInitialBackoffMs, "retry-initial-backoff-ms", 100,
    "Initial backoff delay in milliseconds")
rootCmd.Flags().IntVar(&cfg.RetryMaxBackoffMs, "retry-max-backoff-ms", 10000,
    "Maximum backoff delay in milliseconds")
rootCmd.Flags().Float64Var(&cfg.RetryBackoffMultiplier, "retry-backoff-multiplier", 2.0,
    "Backoff multiplier for exponential increase")

// Viper bindings
viper.BindEnv("retry_max_attempts", "ODATA_RETRY_MAX_ATTEMPTS")
viper.BindEnv("retry_initial_backoff_ms", "ODATA_RETRY_INITIAL_BACKOFF_MS")
viper.BindEnv("retry_max_backoff_ms", "ODATA_RETRY_MAX_BACKOFF_MS")
viper.BindEnv("retry_backoff_multiplier", "ODATA_RETRY_BACKOFF_MULTIPLIER")
```

#### Environment Variables
```
ODATA_RETRY_MAX_ATTEMPTS=3
ODATA_RETRY_INITIAL_BACKOFF_MS=100
ODATA_RETRY_MAX_BACKOFF_MS=10000
ODATA_RETRY_BACKOFF_MULTIPLIER=2.0
```

#### Tests
```go
func TestCalculateBackoff(t *testing.T) {
    cfg := DefaultRetryConfig()
    cfg.JitterFraction = 0 // Disable jitter for predictable tests

    assert.Equal(t, 100*time.Millisecond, cfg.CalculateBackoff(0))
    assert.Equal(t, 200*time.Millisecond, cfg.CalculateBackoff(1))
    assert.Equal(t, 400*time.Millisecond, cfg.CalculateBackoff(2))
}

func TestCalculateBackoffMaxCap(t *testing.T) {
    cfg := DefaultRetryConfig()
    cfg.JitterFraction = 0
    cfg.MaxBackoff = 500 * time.Millisecond

    // Should cap at 500ms
    assert.Equal(t, 500*time.Millisecond, cfg.CalculateBackoff(10))
}

func TestShouldRetry(t *testing.T) {
    cfg := DefaultRetryConfig()

    assert.True(t, cfg.ShouldRetry(503, 0))
    assert.True(t, cfg.ShouldRetry(429, 1))
    assert.False(t, cfg.ShouldRetry(503, 3)) // Max retries exceeded
    assert.False(t, cfg.ShouldRetry(404, 0)) // Not retryable
    assert.False(t, cfg.ShouldRetry(200, 0)) // Success
}

func TestIsCSRFFailure(t *testing.T) {
    resp := &http.Response{StatusCode: 403}

    assert.True(t, IsCSRFFailure(resp, []byte("CSRF token validation failed")))
    assert.True(t, IsCSRFFailure(resp, []byte("csrf error")))
    assert.False(t, IsCSRFFailure(resp, []byte("Access denied")))

    resp.StatusCode = 401
    assert.False(t, IsCSRFFailure(resp, []byte("CSRF"))) // Wrong status code
}
```

---

## Phase 2: Token Optimization (v1.7.0)

### 2.1 Token-Optimized Discovery (Lazy Loading)

**Goal:** Reduce token consumption by ~90% for large SAP services through lazy metadata loading.

This is the most impactful improvement and requires careful architecture changes.

#### Design Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     LAZY LOADING ARCHITECTURE                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Mode: FULL (default, current behavior)                         │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐       │
│  │   Startup   │ ──▶ │ Load Full   │ ──▶ │ Generate    │       │
│  │             │     │ $metadata   │     │ All Tools   │       │
│  └─────────────┘     └─────────────┘     └─────────────┘       │
│                                                                  │
│  Mode: LAZY (new)                                               │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐       │
│  │   Startup   │ ──▶ │ Load Entity │ ──▶ │ Generate    │       │
│  │             │     │ Names Only  │     │ Stub Tools  │       │
│  └─────────────┘     └─────────────┘     └─────────────┘       │
│         │                                       │               │
│         │            ┌─────────────┐           │               │
│         └──────────▶ │ get_schema_ │ ◀─────────┘               │
│                      │ {EntitySet} │                            │
│                      └──────┬──────┘                            │
│                             │                                   │
│                      ┌──────▼──────┐                            │
│                      │ Load Entity │                            │
│                      │ Metadata    │                            │
│                      └──────┬──────┘                            │
│                             │                                   │
│                      ┌──────▼──────┐                            │
│                      │ Generate    │                            │
│                      │ Full Tools  │                            │
│                      └─────────────┘                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### Why This Matters

Large SAP services can have 100+ entity sets. In full mode:
- All metadata is loaded upfront
- All tools are generated (potentially 500+ tools for large services)
- Tool descriptions consume significant tokens in every LLM context

In lazy mode:
- Only entity names are loaded initially
- Minimal stub tools are generated
- Full metadata loaded on-demand when a specific entity is accessed
- **~90% reduction in initial token usage**

#### New Files
```
internal/
  cache/
    metadata_cache.go      # Metadata caching layer
    metadata_cache_test.go # Unit tests
  bridge/
    lazy_loader.go         # Lazy loading orchestration
    lazy_loader_test.go    # Unit tests
```

#### Implementation

**`internal/cache/metadata_cache.go`:**
```go
package cache

import (
    "sync"
    "time"

    "odata-mcp/internal/models"
)

// MetadataCache stores parsed metadata with TTL
type MetadataCache struct {
    mu           sync.RWMutex
    fullMetadata *CachedMetadata
    entityTypes  map[string]*CachedEntityType
    ttl          time.Duration
}

type CachedMetadata struct {
    Metadata *models.Metadata
    LoadedAt time.Time
}

type CachedEntityType struct {
    EntityType *models.EntityType
    LoadedAt   time.Time
}

func NewMetadataCache(ttl time.Duration) *MetadataCache {
    return &MetadataCache{
        entityTypes: make(map[string]*CachedEntityType),
        ttl:         ttl,
    }
}

func (c *MetadataCache) GetFullMetadata() (*models.Metadata, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if c.fullMetadata == nil {
        return nil, false
    }

    if time.Since(c.fullMetadata.LoadedAt) > c.ttl {
        return nil, false // Expired
    }

    return c.fullMetadata.Metadata, true
}

func (c *MetadataCache) SetFullMetadata(m *models.Metadata) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.fullMetadata = &CachedMetadata{
        Metadata: m,
        LoadedAt: time.Now(),
    }

    // Also cache individual entity types
    for i := range m.EntityTypes {
        et := &m.EntityTypes[i]
        c.entityTypes[et.Name] = &CachedEntityType{
            EntityType: et,
            LoadedAt:   time.Now(),
        }
    }
}

func (c *MetadataCache) GetEntityType(name string) (*models.EntityType, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    cached, exists := c.entityTypes[name]
    if !exists {
        return nil, false
    }

    if time.Since(cached.LoadedAt) > c.ttl {
        return nil, false // Expired
    }

    return cached.EntityType, true
}

func (c *MetadataCache) GetLoadedEntityNames() []string {
    c.mu.RLock()
    defer c.mu.RUnlock()

    names := make([]string, 0, len(c.entityTypes))
    for name, cached := range c.entityTypes {
        if time.Since(cached.LoadedAt) <= c.ttl {
            names = append(names, name)
        }
    }
    return names
}

func (c *MetadataCache) Invalidate() {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.fullMetadata = nil
    c.entityTypes = make(map[string]*CachedEntityType)
}
```

**`internal/bridge/lazy_loader.go`:**
```go
package bridge

import (
    "context"
    "fmt"
    "os"
    "sync"
    "time"

    "odata-mcp/internal/cache"
    "odata-mcp/internal/client"
    "odata-mcp/internal/models"
)

// LazyLoader handles on-demand metadata loading
type LazyLoader struct {
    client       *client.ODataClient
    cache        *cache.MetadataCache
    entityNames  []string        // Known entity set names (from initial scan)
    mu           sync.RWMutex
    verbose      bool
}

// NewLazyLoader creates a lazy loader with entity names only
func NewLazyLoader(c *client.ODataClient, entityNames []string, cacheTTL time.Duration, verbose bool) *LazyLoader {
    return &LazyLoader{
        client:      c,
        cache:       cache.NewMetadataCache(cacheTTL),
        entityNames: entityNames,
        verbose:     verbose,
    }
}

// GetFullMetadata loads and returns full metadata (triggers full load)
func (l *LazyLoader) GetFullMetadata(ctx context.Context) (*models.Metadata, error) {
    // Check cache first
    if m, ok := l.cache.GetFullMetadata(); ok {
        if l.verbose {
            fmt.Fprintf(os.Stderr, "[VERBOSE] Using cached metadata\n")
        }
        return m, nil
    }

    l.mu.Lock()
    defer l.mu.Unlock()

    // Double-check after lock
    if m, ok := l.cache.GetFullMetadata(); ok {
        return m, nil
    }

    if l.verbose {
        fmt.Fprintf(os.Stderr, "[VERBOSE] Lazy loading full metadata\n")
    }

    metadata, err := l.client.GetMetadata(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to load metadata: %w", err)
    }

    l.cache.SetFullMetadata(metadata)
    return metadata, nil
}

// GetEntityType loads metadata if needed and returns specific entity type
func (l *LazyLoader) GetEntityType(ctx context.Context, entityName string) (*models.EntityType, error) {
    // Check cache first
    if et, ok := l.cache.GetEntityType(entityName); ok {
        return et, nil
    }

    // Need to load full metadata to get entity type
    metadata, err := l.GetFullMetadata(ctx)
    if err != nil {
        return nil, err
    }

    // Find the entity type
    for i := range metadata.EntityTypes {
        if metadata.EntityTypes[i].Name == entityName {
            return &metadata.EntityTypes[i], nil
        }
    }

    return nil, fmt.Errorf("entity type not found: %s", entityName)
}

// GetAllEntityNames returns all known entity set names (no metadata load needed)
func (l *LazyLoader) GetAllEntityNames() []string {
    return l.entityNames
}

// IsMetadataLoaded checks if full metadata is cached
func (l *LazyLoader) IsMetadataLoaded() bool {
    _, ok := l.cache.GetFullMetadata()
    return ok
}

// InvalidateCache clears the metadata cache
func (l *LazyLoader) InvalidateCache() {
    l.cache.Invalidate()
}
```

#### Modify `internal/bridge/bridge.go`

Add lazy mode support:

```go
type ODataMCPBridge struct {
    // ... existing fields ...
    lazyMode     bool
    lazyLoader   *LazyLoader
    entityNames  []string  // For lazy mode: known entity names
}

func NewODataMCPBridge(cfg *config.Config) (*ODataMCPBridge, error) {
    bridge := &ODataMCPBridge{
        cfg:      cfg,
        client:   client.NewODataClient(cfg),
        lazyMode: cfg.LazyLoadMetadata,
        tools:    make([]models.Tool, 0),
    }

    if err := bridge.initialize(); err != nil {
        return nil, err
    }

    return bridge, nil
}

func (b *ODataMCPBridge) initialize() error {
    ctx := context.Background()

    if b.lazyMode {
        return b.initializeLazy(ctx)
    }
    return b.initializeFull(ctx)
}

func (b *ODataMCPBridge) initializeLazy(ctx context.Context) error {
    if b.cfg.Verbose {
        fmt.Fprintf(os.Stderr, "[VERBOSE] Initializing in lazy mode\n")
    }

    // Quick scan to get entity names only
    entityNames, err := b.client.GetEntitySetNames(ctx)
    if err != nil {
        return fmt.Errorf("failed to get entity names: %w", err)
    }

    b.entityNames = entityNames
    b.lazyLoader = NewLazyLoader(
        b.client,
        entityNames,
        time.Duration(b.cfg.MetadataCacheTTL)*time.Second,
        b.cfg.Verbose,
    )

    if b.cfg.Verbose {
        fmt.Fprintf(os.Stderr, "[VERBOSE] Found %d entity sets\n", len(entityNames))
    }

    // Generate service info tool
    b.addServiceInfoTool()

    // Generate list entities tool
    b.addListEntitiesTool()

    // Generate schema inspection tool
    b.addSchemaInspectionTool()

    // Generate minimal query tools for each entity
    b.generateLazyTools(entityNames)

    return nil
}

func (b *ODataMCPBridge) addListEntitiesTool() {
    tool := models.Tool{
        Name:        b.formatToolName("list_entities"),
        Description: "List all available entity sets in this OData service",
        InputSchema: map[string]interface{}{
            "type":       "object",
            "properties": map[string]interface{}{},
        },
    }
    b.tools = append(b.tools, tool)
}

func (b *ODataMCPBridge) addSchemaInspectionTool() {
    tool := models.Tool{
        Name:        b.formatToolName("get_entity_schema"),
        Description: "Get full schema, properties, and available operations for an entity set. Call this before querying an entity to understand its structure.",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "entity_name": map[string]interface{}{
                    "type":        "string",
                    "description": "Name of the entity set to inspect",
                    "enum":        b.entityNames,
                },
            },
            "required": []string{"entity_name"},
        },
    }
    b.tools = append(b.tools, tool)
}

func (b *ODataMCPBridge) generateLazyTools(entityNames []string) {
    for _, name := range entityNames {
        // In lazy mode, generate only a single "query" tool per entity
        // with generic parameters. Full schema loads on first use.
        tool := models.Tool{
            Name:        b.formatToolName(fmt.Sprintf("query_%s", name)),
            Description: fmt.Sprintf("Query %s entity set. Use get_entity_schema first to see available fields.", name),
            InputSchema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "filter": map[string]interface{}{
                        "type":        "string",
                        "description": "OData $filter expression",
                    },
                    "select": map[string]interface{}{
                        "type":        "string",
                        "description": "Comma-separated list of fields to return",
                    },
                    "top": map[string]interface{}{
                        "type":        "integer",
                        "description": "Maximum number of results",
                    },
                    "skip": map[string]interface{}{
                        "type":        "integer",
                        "description": "Number of results to skip",
                    },
                    "orderby": map[string]interface{}{
                        "type":        "string",
                        "description": "Field to sort by (add ' desc' for descending)",
                    },
                },
            },
        }
        b.tools = append(b.tools, tool)
    }
}
```

#### Add Client Method for Entity Names

**`internal/client/client.go`:**
```go
// GetEntitySetNames performs a lightweight $metadata scan for entity names only
func (c *ODataClient) GetEntitySetNames(ctx context.Context) ([]string, error) {
    // Fetch $metadata
    metadataURL := strings.TrimSuffix(c.baseURL, "/") + "/$metadata"

    if c.verbose {
        fmt.Fprintf(os.Stderr, "[VERBOSE] Fetching entity names from %s\n", metadataURL)
    }

    resp, body, err := c.doRequestWithRetry(ctx, "GET", metadataURL, nil, "")
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("metadata request failed with status %d", resp.StatusCode)
    }

    // Quick regex scan for EntitySet names (faster than full XML parse)
    // Handles both OData v2 and v4 formats:
    // v2: <EntitySet Name="Products" ...>
    // v4: <EntitySet Name="Products" ...>
    re := regexp.MustCompile(`<EntitySet[^>]+Name="([^"]+)"`)
    matches := re.FindAllSubmatch(body, -1)

    names := make([]string, 0, len(matches))
    seen := make(map[string]bool)
    for _, match := range matches {
        name := string(match[1])
        if !seen[name] {
            seen[name] = true
            names = append(names, name)
        }
    }

    if c.verbose {
        fmt.Fprintf(os.Stderr, "[VERBOSE] Found %d entity sets\n", len(names))
    }

    return names, nil
}
```

#### Config Changes

**`internal/config/config.go`:**
```go
// Add to Config struct
LazyLoadMetadata  bool `mapstructure:"lazy_load_metadata"`
MetadataCacheTTL  int  `mapstructure:"metadata_cache_ttl"` // seconds, default 3600
```

#### CLI Flags

**`cmd/odata-mcp/main.go`:**
```go
// Lazy loading configuration
rootCmd.Flags().BoolVar(&cfg.LazyLoadMetadata, "lazy", false,
    "Enable lazy loading of metadata (reduces initial token usage)")
rootCmd.Flags().IntVar(&cfg.MetadataCacheTTL, "metadata-cache-ttl", 3600,
    "TTL for cached metadata in seconds")

// Viper bindings
viper.BindEnv("lazy_load_metadata", "ODATA_LAZY_LOAD_METADATA")
viper.BindEnv("metadata_cache_ttl", "ODATA_METADATA_CACHE_TTL")
```

#### Environment Variables
```
ODATA_LAZY_LOAD_METADATA=true
ODATA_METADATA_CACHE_TTL=3600
```

#### Usage Example

```bash
# Full mode (default) - loads all metadata upfront
./odata-mcp --service https://solman.company.com/sap/opu/odata/sap/HUGE_SERVICE/

# Lazy mode - only loads entity names, metadata on-demand
./odata-mcp --lazy --service https://solman.company.com/sap/opu/odata/sap/HUGE_SERVICE/
```

In lazy mode, the AI workflow becomes:
1. `list_entities` → See all 100+ entity sets
2. `get_entity_schema Products` → Load and see Products schema
3. `query_Products` → Execute query with known fields

---

## Phase 3: Skill Generator (v1.8.0)

### 3.1 Overview

**Goal:** Auto-generate Claude Code Skills from OData service metadata, creating AI-native documentation that guides LLMs through complex multi-tool workflows.

Skills are markdown files that serve as intelligent usage guides for MCP tools. Unlike raw tool schemas, Skills provide:
- **Context**: What the service/entity is for
- **Workflows**: Multi-step procedures combining multiple tools
- **Best Practices**: Recommended patterns, common filters, pagination hints
- **Domain Knowledge**: Business terminology and relationships

### 3.2 Design Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     SKILL GENERATOR FLOW                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐       │
│  │   OData     │ ──▶ │   Parse     │ ──▶ │  Generate   │       │
│  │  $metadata  │     │  Schema     │     │  Skill MD   │       │
│  └─────────────┘     └─────────────┘     └─────────────┘       │
│                                                                  │
│  Optional Enhancement:                                           │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐       │
│  │   Hints     │ ──▶ │   Merge     │ ──▶ │  Enhanced   │       │
│  │   File      │     │  Context    │     │  Skill MD   │       │
│  └─────────────┘     └─────────────┘     └─────────────┘       │
│                                                                  │
│  Output Structure:                                               │
│  skills/                                                         │
│  ├── {service-id}/                                              │
│  │   ├── README.md           # Service overview                 │
│  │   ├── entities/                                              │
│  │   │   ├── {EntitySet}.md  # Per-entity skills                │
│  │   │   └── ...                                                │
│  │   └── workflows/                                              │
│  │       ├── common-queries.md                                   │
│  │       └── {domain-workflow}.md                                │
│  └── ...                                                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 3.3 Example Use Case: SAP Solution Manager BPA

SAP Business Process Analytics (BPA) in Solution Manager provides OData services for:

- Process step monitoring
- Exception analysis
- Volume analytics
- Process mining integration

**Generated Skill Example (`skills/solman-bpa/workflows/exception-analysis.md`):**

```markdown
# Exception Analysis Workflow

## Purpose
Analyze process exceptions to identify bottlenecks and failures in SAP business processes.

## Prerequisites
- Access to Solution Manager BPA OData service
- Process step data populated from monitored systems

## Workflow Steps

### Step 1: Get Available Process Definitions
Use `list_entities` to discover available BPA entities, then:
\`\`\`
query_ProcessDefinitions --top 100 --select "ProcessID,ProcessName,Status"
\`\`\`

### Step 2: Identify Exception Patterns
Query exception logs for a specific process:
\`\`\`
query_ProcessStepExceptions --filter "ProcessID eq 'SALES_ORDER_PROC'" --orderby "ExceptionCount desc" --top 20
\`\`\`

### Step 3: Drill Down to Step Details
For the top exception, get step-level details:
\`\`\`
get_ProcessStep --ProcessStepID '{step_id}' --expand "Exceptions,Metrics"
\`\`\`

### Step 4: Analyze Time Patterns
Check if exceptions correlate with time:
\`\`\`
query_ProcessMetrics --filter "ProcessID eq 'SALES_ORDER_PROC' and MetricType eq 'EXCEPTION_RATE'" --select "Timestamp,Value" --orderby "Timestamp desc" --top 100
\`\`\`

## Common Filters
- `Status eq 'ERROR'` - Failed process steps only
- `Timestamp ge datetime'2025-01-01T00:00:00'` - Recent data
- `ExceptionCount gt 10` - Frequent exceptions

## Tips
- Always start with `list_entities` if unsure of available data
- Use `get_entity_schema` to understand field types before filtering
- BPA data can be large; use `--top` to limit initial queries
- Exception timestamps are in UTC
```

### 3.4 Implementation

#### New Files

```
internal/
  skills/
    generator.go       # Skill generation logic
    generator_test.go  # Unit tests
    templates/         # Go templates for skill markdown
      service.md.tmpl
      entity.md.tmpl
      workflow.md.tmpl
```

#### CLI Flags

```go
// Skill generation
rootCmd.Flags().BoolVar(&cfg.GenerateSkills, "generate-skills", false,
    "Generate Claude Code Skills from OData metadata")
rootCmd.Flags().StringVar(&cfg.SkillsOutputDir, "skills-output", "./skills",
    "Output directory for generated skills")
rootCmd.Flags().StringVar(&cfg.SkillsHintsFile, "skills-hints", "",
    "Optional hints file with domain-specific context for skill generation")

// Viper bindings
viper.BindEnv("generate_skills", "ODATA_GENERATE_SKILLS")
viper.BindEnv("skills_output", "ODATA_SKILLS_OUTPUT")
viper.BindEnv("skills_hints", "ODATA_SKILLS_HINTS")
```

#### Config Additions

**`internal/config/config.go`:**
```go
// Add to Config struct
GenerateSkills   bool   `mapstructure:"generate_skills"`
SkillsOutputDir  string `mapstructure:"skills_output"`
SkillsHintsFile  string `mapstructure:"skills_hints"`
```

#### Generator Logic

**`internal/skills/generator.go`:**
```go
package skills

import (
    "fmt"
    "os"
    "path/filepath"
    "text/template"

    "odata-mcp/internal/models"
)

type SkillGenerator struct {
    metadata    *models.Metadata
    serviceID   string
    outputDir   string
    hints       map[string]interface{}
    templates   *template.Template
}

type ServiceSkillData struct {
    ServiceID     string
    ServiceURL    string
    ODataVersion  string
    EntitySets    []EntitySetData
    FunctionImports []FunctionImportData
    Hints         map[string]interface{}
}

type EntitySetData struct {
    Name        string
    EntityType  string
    Description string
    Properties  []PropertyData
    Keys        []string
    Operations  []string // C, R, U, D
}

type PropertyData struct {
    Name        string
    Type        string
    Nullable    bool
    Description string
}

func NewSkillGenerator(metadata *models.Metadata, serviceID, outputDir string) *SkillGenerator {
    return &SkillGenerator{
        metadata:  metadata,
        serviceID: serviceID,
        outputDir: outputDir,
        hints:     make(map[string]interface{}),
    }
}

func (g *SkillGenerator) LoadHints(hintsFile string) error {
    // Load domain-specific hints from JSON file
    // These enhance generated skills with business context
    return nil // TODO: implement
}

func (g *SkillGenerator) Generate() error {
    // Create output directory structure
    baseDir := filepath.Join(g.outputDir, g.serviceID)
    entitiesDir := filepath.Join(baseDir, "entities")
    workflowsDir := filepath.Join(baseDir, "workflows")

    for _, dir := range []string{baseDir, entitiesDir, workflowsDir} {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("failed to create directory %s: %w", dir, err)
        }
    }

    // Generate service README
    if err := g.generateServiceReadme(baseDir); err != nil {
        return err
    }

    // Generate per-entity skills
    for _, es := range g.metadata.EntitySets {
        if err := g.generateEntitySkill(entitiesDir, es); err != nil {
            return err
        }
    }

    // Generate common workflow templates
    if err := g.generateCommonWorkflows(workflowsDir); err != nil {
        return err
    }

    return nil
}

func (g *SkillGenerator) generateServiceReadme(baseDir string) error {
    // Generate service-level README with overview and tool list
    return nil // TODO: implement with template
}

func (g *SkillGenerator) generateEntitySkill(entitiesDir string, es models.EntitySet) error {
    // Generate per-entity skill with schema, examples, and tips
    return nil // TODO: implement with template
}

func (g *SkillGenerator) generateCommonWorkflows(workflowsDir string) error {
    // Generate common workflow patterns (CRUD, search, pagination)
    return nil // TODO: implement with template
}
```

#### Skill Templates

**`internal/skills/templates/entity.md.tmpl`:**

```markdown
# {{ .Name }} Entity

## Description
{{ .Description | default "No description available in metadata." }}

## Schema

| Property | Type | Required | Description |
|----------|------|----------|-------------|
{{- range .Properties }}
| {{ .Name }} | {{ .Type }} | {{ if .Nullable }}No{{ else }}Yes{{ end }} | {{ .Description }} |
{{- end }}

## Key Fields
{{- range .Keys }}
- `{{ . }}`
{{- end }}

## Available Operations
{{- range .Operations }}
- {{ . }}
{{- end }}

## Example Queries

### List all records
\`\`\`
query_{{ .Name }} --top 10
\`\`\`

### Filter by key
\`\`\`
get_{{ .Name }} --{{ index .Keys 0 }} "value"
\`\`\`

### Search with filter
\`\`\`
query_{{ .Name }} --filter "{{ index .Properties 0 | .Name }} eq 'value'" --select "{{ .Keys | join "," }}"
\`\`\`

## Tips
- Use `get_entity_schema {{ .Name }}` to see full schema details
- Maximum {{ .MaxItems | default 100 }} items returned per query
- {{ .Hints | default "Check service documentation for business rules" }}
```

### 3.5 Integration with Solution Manager Hub

Skills generated from Solution Manager BPA services can serve as the foundation for:

1. **Cross-System Workflows**: Skills reference tools from multiple connected systems
2. **Process Mining Integration**: Link BPA data to Celonis or other mining tools
3. **Alerting Workflows**: Guide AI through exception escalation procedures
4. **Compliance Reporting**: Standard queries for audit and compliance

See [SOLMAN_HUB_ARCHITECTURE.md](SOLMAN_HUB_ARCHITECTURE.md) for the broader Solution Manager integration strategy.

### 3.6 Skills Hints File Format

**`skills-hints.json`:**

```json
{
  "service": {
    "description": "SAP Solution Manager Business Process Analytics",
    "domain": "Process Monitoring",
    "documentation_url": "https://help.sap.com/docs/bpa"
  },
  "entities": {
    "ProcessDefinitions": {
      "description": "Business process definitions configured in Solution Manager",
      "tips": [
        "Process IDs are unique across the landscape",
        "Use Status filter to find active processes only"
      ],
      "common_filters": [
        "Status eq 'ACTIVE'",
        "ProcessType eq 'STANDARD'"
      ]
    },
    "ProcessStepExceptions": {
      "description": "Exceptions and errors from process step execution",
      "tips": [
        "High ExceptionCount indicates systemic issues",
        "Correlate with ProcessMetrics for time-based patterns"
      ],
      "related_entities": ["ProcessSteps", "ProcessMetrics"]
    }
  },
  "workflows": {
    "exception-analysis": {
      "name": "Exception Analysis",
      "description": "Identify and analyze process exceptions",
      "steps": [
        "List available processes",
        "Query exception patterns",
        "Drill down to step details",
        "Analyze time correlations"
      ]
    }
  }
}
```

### 3.7 Usage Examples

```bash
# Generate skills from OData metadata
./odata-mcp --service https://solman.company.com/sap/opu/odata/sap/BPA_SRV/ \
  --generate-skills \
  --skills-output ./skills

# Generate with domain hints
./odata-mcp --service https://solman.company.com/sap/opu/odata/sap/BPA_SRV/ \
  --generate-skills \
  --skills-hints ./bpa-hints.json \
  --skills-output ./skills

# Use generated skills with Claude Code
claude --skill ./skills/bpa-srv/workflows/exception-analysis.md
```

### 3.8 Future Enhancements

1. **LLM-Assisted Skill Enhancement**: Use Claude to enrich generated skills with better descriptions
2. **Interactive Skill Discovery**: Runtime skill recommendations based on user queries
3. **Skill Composition**: Combine multiple entity skills into complex workflow skills
4. **Skill Versioning**: Track skill versions alongside metadata changes
5. **Skill Validation**: Test skills against live services to verify examples work

---

## Testing Strategy

### Unit Tests

Each new file should have corresponding `*_test.go`:

```
internal/debug/masking_test.go
internal/client/retry_test.go
internal/cache/metadata_cache_test.go
internal/bridge/lazy_loader_test.go
```

### Integration Tests

**`internal/test/retry_integration_test.go`:**
- Test against mock server returning 429, 503
- Verify backoff timing
- Verify max retries honored

**`internal/test/lazy_load_integration_test.go`:**
- Test lazy loading with Northwind service
- Verify stub tools generated
- Verify on-demand loading works
- Measure token savings

### Mock Servers

Create test helpers in `internal/test/mocks/`:
- `flaky_server.go` - Server that returns errors for retry testing
- `slow_server.go` - Server with delays for timeout testing

---

## Documentation Updates

### README.md Updates

Add sections for:
- Retry configuration
- Lazy loading mode

### New Documentation Files

- `docs/RETRY.md` - Retry behavior documentation
- `docs/LAZY_LOADING.md` - Lazy loading mode guide

### CHANGELOG.md Entries

```markdown
## [1.6.0] - TBD

### Added
- Exponential backoff retry with configurable parameters
- Credential masking in verbose output
- New CLI flags: --retry-max-attempts, --retry-initial-backoff-ms, --retry-max-backoff-ms

### Changed
- Improved error handling for transient failures
- CSRF retry now integrated with general retry logic

## [1.7.0] - TBD

### Added
- Lazy loading mode for reduced token consumption (--lazy flag)
- Metadata caching with configurable TTL
- New tools in lazy mode: list_entities, get_entity_schema
- New CLI flags: --lazy, --metadata-cache-ttl

### Changed
- Tool generation refactored to support both full and lazy modes
```

---

## Milestones & Checklist

### v1.6.0 Milestone (Foundation) ✅ COMPLETE

- [x] Implement credential masking (`internal/debug/masking.go`)
- [x] Write masking unit tests
- [x] Implement exponential backoff (`internal/client/retry.go`)
- [x] Write retry unit tests
- [x] Add CLI flags for retry configuration
- [x] Update `internal/client/client.go` to use retry config
- [x] Update documentation (README, CHANGELOG)
- [ ] Create `docs/RETRY.md` (optional)

### v1.7.0 Milestone (Token Optimization)

- [ ] Implement metadata cache (`internal/cache/metadata_cache.go`)
- [ ] Write cache unit tests
- [ ] Implement lazy loader (`internal/bridge/lazy_loader.go`)
- [ ] Write lazy loader unit tests
- [ ] Add `GetEntitySetNames()` to client
- [ ] Modify bridge to support lazy mode
- [ ] Add lazy mode tools (list_entities, get_entity_schema)
- [ ] Add CLI flags for lazy loading
- [ ] Write integration tests with Northwind
- [ ] Update documentation (README, CHANGELOG)
- [ ] Create `docs/LAZY_LOADING.md`
- [ ] Performance benchmarks

### v1.8.0 Milestone (Skill Generator)

- [ ] Create skill generator module (`internal/skills/generator.go`)
- [ ] Write generator unit tests
- [ ] Create Go templates for skill markdown files
- [ ] Add CLI flags (`--generate-skills`, `--skills-output`, `--skills-hints`)
- [ ] Implement hints file loading and merging
- [ ] Generate service README from metadata
- [ ] Generate per-entity skill files
- [ ] Generate common workflow templates
- [ ] Test with SAP BPA service metadata
- [ ] Update documentation (README, CHANGELOG)
- [ ] Create `docs/SKILL_GENERATOR.md`

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Lazy loading breaks existing workflows | Medium | High | Make it opt-in, default to full mode |
| Retry storms on service outages | Low | Medium | Cap max retries, jitter prevents thundering herd |
| Cache invalidation issues | Medium | Low | Short default TTL, manual invalidation option |
| Regex parsing misses entity sets | Low | Medium | Test against multiple OData versions |
| Generated skills lack domain context | Medium | Medium | Hints file system for domain enrichment |
| Skills become stale after metadata changes | Low | Low | Include version/timestamp in skills |
| Template complexity grows | Low | Medium | Keep templates simple, use hints for customization |

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Token reduction (lazy mode) | >80% | Compare tool list response size |
| Retry success rate | >95% | Integration tests with flaky mock |
| Credential exposure | 0 instances | Audit verbose output in tests |
| Backward compatibility | 100% | Existing tests pass without changes |
| Skill generation coverage | 100% entities | All entity sets get skill files |
| Skill usefulness | User validation | Skills correctly guide multi-step workflows |
| Hint integration | Seamless merge | Hints appear in correct skill sections |

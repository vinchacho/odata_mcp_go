// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zmcp/odata-mcp/internal/bridge"
	"github.com/zmcp/odata-mcp/internal/config"
)

// TestLazyModeIntegration tests lazy metadata mode against the real Northwind service
// Run with: INTEGRATION_TESTS=true go test -run TestLazyModeIntegration ./internal/test/
func TestLazyModeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Integration tests are opt-in: require INTEGRATION_TESTS=true to run
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test (set INTEGRATION_TESTS=true to run)")
	}

	serviceURL := "https://services.odata.org/V4/Northwind/Northwind.svc/"

	t.Run("LazyModeToolCount", func(t *testing.T) {
		// Test that lazy mode generates exactly 10 tools
		cfg := &config.Config{
			ServiceURL:   serviceURL,
			LazyMetadata: true,
		}

		b, err := bridge.NewODataMCPBridge(cfg)
		require.NoError(t, err)

		traceInfo, err := b.GetTraceInfo()
		require.NoError(t, err)

		assert.Equal(t, 10, traceInfo.TotalTools, "Lazy mode should generate exactly 10 tools")

		// Verify each expected tool exists by prefix
		expectedPrefixes := []string{
			"odata_service_info",
			"list_entities",
			"count_entities",
			"get_entity",
			"get_entity_schema",
			"create_entity",
			"update_entity",
			"delete_entity",
			"list_functions",
			"call_function",
		}

		for _, prefix := range expectedPrefixes {
			found := false
			for _, tool := range traceInfo.RegisteredTools {
				if strings.HasPrefix(tool.Name, prefix) {
					found = true
					break
				}
			}
			assert.True(t, found, "Missing tool with prefix: %s", prefix)
		}
	})

	t.Run("LazyModeReadOnly", func(t *testing.T) {
		// Test that read-only lazy mode generates 7 tools
		cfg := &config.Config{
			ServiceURL:   serviceURL,
			LazyMetadata: true,
			ReadOnly:     true,
		}

		b, err := bridge.NewODataMCPBridge(cfg)
		require.NoError(t, err)

		traceInfo, err := b.GetTraceInfo()
		require.NoError(t, err)

		assert.Equal(t, 7, traceInfo.TotalTools, "Lazy read-only mode should generate 7 tools")

		// Verify mutating tools are not present
		mutatingPrefixes := []string{
			"create_entity",
			"update_entity",
			"delete_entity",
		}

		for _, prefix := range mutatingPrefixes {
			for _, tool := range traceInfo.RegisteredTools {
				assert.False(t, strings.HasPrefix(tool.Name, prefix),
					"Read-only mode should not have tool with prefix: %s", prefix)
			}
		}
	})

	t.Run("EagerModeToolCount", func(t *testing.T) {
		// Test that eager mode generates many more tools
		cfg := &config.Config{
			ServiceURL: serviceURL,
		}

		b, err := bridge.NewODataMCPBridge(cfg)
		require.NoError(t, err)

		traceInfo, err := b.GetTraceInfo()
		require.NoError(t, err)

		// Northwind has ~26 entity sets, so we expect many more tools than 10
		assert.Greater(t, traceInfo.TotalTools, 50, "Eager mode should generate many tools (got %d)", traceInfo.TotalTools)
	})

	t.Run("LazyThresholdAutoEnable", func(t *testing.T) {
		// Test that lazy threshold auto-enables lazy mode
		cfg := &config.Config{
			ServiceURL:    serviceURL,
			LazyThreshold: 20, // Low threshold to trigger lazy mode
		}

		b, err := bridge.NewODataMCPBridge(cfg)
		require.NoError(t, err)

		traceInfo, err := b.GetTraceInfo()
		require.NoError(t, err)

		// Should be 10 tools since threshold triggers lazy mode
		assert.Equal(t, 10, traceInfo.TotalTools, "Lazy threshold should auto-enable lazy mode")
	})

	t.Run("LazyThresholdNotTriggered", func(t *testing.T) {
		// Test that a high threshold doesn't trigger lazy mode
		cfg := &config.Config{
			ServiceURL:    serviceURL,
			LazyThreshold: 1000, // High threshold, won't trigger
		}

		b, err := bridge.NewODataMCPBridge(cfg)
		require.NoError(t, err)

		traceInfo, err := b.GetTraceInfo()
		require.NoError(t, err)

		// Should be many tools since threshold doesn't trigger
		assert.Greater(t, traceInfo.TotalTools, 50, "High threshold should not trigger lazy mode")
	})
}

// TestLazyModeTokenSavings validates the token savings from lazy mode
// Run with: INTEGRATION_TESTS=true go test -run TestLazyModeTokenSavings ./internal/test/
func TestLazyModeTokenSavings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Integration tests are opt-in: require INTEGRATION_TESTS=true to run
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test (set INTEGRATION_TESTS=true to run)")
	}

	serviceURL := "https://services.odata.org/V4/Northwind/Northwind.svc/"

	// Get eager mode tool count
	eagerCfg := &config.Config{
		ServiceURL: serviceURL,
	}
	eagerBridge, err := bridge.NewODataMCPBridge(eagerCfg)
	require.NoError(t, err)
	eagerTrace, err := eagerBridge.GetTraceInfo()
	require.NoError(t, err)
	eagerToolCount := eagerTrace.TotalTools

	// Get lazy mode tool count
	lazyCfg := &config.Config{
		ServiceURL:   serviceURL,
		LazyMetadata: true,
	}
	lazyBridge, err := bridge.NewODataMCPBridge(lazyCfg)
	require.NoError(t, err)
	lazyTrace, err := lazyBridge.GetTraceInfo()
	require.NoError(t, err)
	lazyToolCount := lazyTrace.TotalTools

	// Calculate reduction ratio
	reductionRatio := float64(eagerToolCount-lazyToolCount) / float64(eagerToolCount) * 100

	t.Logf("Eager mode: %d tools", eagerToolCount)
	t.Logf("Lazy mode: %d tools", lazyToolCount)
	t.Logf("Tool count reduction: %.1f%%", reductionRatio)

	// Verify lazy mode uses exactly 10 tools
	assert.Equal(t, 10, lazyToolCount, "Lazy mode should use exactly 10 tools")

	// Verify significant reduction (at least 80%)
	assert.Greater(t, reductionRatio, 80.0,
		"Lazy mode should reduce tool count by at least 80%% (got %.1f%%)", reductionRatio)
}

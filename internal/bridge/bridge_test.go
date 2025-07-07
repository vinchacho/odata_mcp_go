package bridge

import (
	"testing"

	"github.com/zmcp/odata-mcp/internal/config"
)

func TestParameterTransformation(t *testing.T) {
	tests := []struct {
		name               string
		claudeCodeFriendly bool
		inputParam         string
		expectedOutput     string
	}{
		{
			name:               "Standard mode - keeps $ prefix",
			claudeCodeFriendly: false,
			inputParam:         "$filter",
			expectedOutput:     "$filter",
		},
		{
			name:               "Claude-friendly mode - removes $ prefix",
			claudeCodeFriendly: true,
			inputParam:         "$filter",
			expectedOutput:     "filter",
		},
		{
			name:               "Claude-friendly mode - handles select",
			claudeCodeFriendly: true,
			inputParam:         "$select",
			expectedOutput:     "select",
		},
		{
			name:               "Claude-friendly mode - handles expand",
			claudeCodeFriendly: true,
			inputParam:         "$expand",
			expectedOutput:     "expand",
		},
		{
			name:               "Claude-friendly mode - handles orderby",
			claudeCodeFriendly: true,
			inputParam:         "$orderby",
			expectedOutput:     "orderby",
		},
		{
			name:               "Claude-friendly mode - handles top",
			claudeCodeFriendly: true,
			inputParam:         "$top",
			expectedOutput:     "top",
		},
		{
			name:               "Claude-friendly mode - handles skip",
			claudeCodeFriendly: true,
			inputParam:         "$skip",
			expectedOutput:     "skip",
		},
		{
			name:               "Claude-friendly mode - handles count",
			claudeCodeFriendly: true,
			inputParam:         "$count",
			expectedOutput:     "count",
		},
		{
			name:               "Standard mode - non-OData param unchanged",
			claudeCodeFriendly: false,
			inputParam:         "customParam",
			expectedOutput:     "customParam",
		},
		{
			name:               "Claude-friendly mode - non-OData param unchanged",
			claudeCodeFriendly: true,
			inputParam:         "customParam",
			expectedOutput:     "customParam",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ClaudeCodeFriendly: tt.claudeCodeFriendly,
			}
			bridge := &ODataMCPBridge{
				config: cfg,
			}

			result := bridge.getParameterName(tt.inputParam)
			if result != tt.expectedOutput {
				t.Errorf("getParameterName() = %v, want %v", result, tt.expectedOutput)
			}
		})
	}
}

func TestParameterMapping(t *testing.T) {
	tests := []struct {
		name               string
		claudeCodeFriendly bool
		inputParam         string
		expectedOutput     string
	}{
		// Claude-friendly mode tests
		{
			name:               "Maps filter to $filter",
			claudeCodeFriendly: true,
			inputParam:         "filter",
			expectedOutput:     "$filter",
		},
		{
			name:               "Maps select to $select",
			claudeCodeFriendly: true,
			inputParam:         "select",
			expectedOutput:     "$select",
		},
		{
			name:               "Maps expand to $expand",
			claudeCodeFriendly: true,
			inputParam:         "expand",
			expectedOutput:     "$expand",
		},
		{
			name:               "Maps orderby to $orderby",
			claudeCodeFriendly: true,
			inputParam:         "orderby",
			expectedOutput:     "$orderby",
		},
		{
			name:               "Maps top to $top",
			claudeCodeFriendly: true,
			inputParam:         "top",
			expectedOutput:     "$top",
		},
		{
			name:               "Maps skip to $skip",
			claudeCodeFriendly: true,
			inputParam:         "skip",
			expectedOutput:     "$skip",
		},
		{
			name:               "Maps count to $count",
			claudeCodeFriendly: true,
			inputParam:         "count",
			expectedOutput:     "$count",
		},
		{
			name:               "Keeps $filter as is",
			claudeCodeFriendly: true,
			inputParam:         "$filter",
			expectedOutput:     "$filter",
		},
		{
			name:               "Non-OData param unchanged",
			claudeCodeFriendly: true,
			inputParam:         "customParam",
			expectedOutput:     "customParam",
		},
		// Standard mode tests
		{
			name:               "Standard mode - keeps $filter",
			claudeCodeFriendly: false,
			inputParam:         "$filter",
			expectedOutput:     "$filter",
		},
		{
			name:               "Standard mode - adds $ to filter",
			claudeCodeFriendly: false,
			inputParam:         "filter",
			expectedOutput:     "$filter",
		},
		{
			name:               "Standard mode - adds $ to select",
			claudeCodeFriendly: false,
			inputParam:         "select",
			expectedOutput:     "$select",
		},
		{
			name:               "Standard mode - non-OData unchanged",
			claudeCodeFriendly: false,
			inputParam:         "customParam",
			expectedOutput:     "customParam",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ClaudeCodeFriendly: tt.claudeCodeFriendly,
			}
			bridge := &ODataMCPBridge{
				config: cfg,
			}

			result := bridge.mapParameterToOData(tt.inputParam)
			if result != tt.expectedOutput {
				t.Errorf("mapParameterToOData() = %v, want %v", result, tt.expectedOutput)
			}
		})
	}
}
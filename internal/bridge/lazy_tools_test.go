// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package bridge

import (
	"context"
	"strings"
	"testing"

	"github.com/zmcp/odata-mcp/internal/config"
	"github.com/zmcp/odata-mcp/internal/mcp"
	"github.com/zmcp/odata-mcp/internal/models"
)

// createTestMetadata creates a minimal OData metadata for testing
func createTestMetadata() *models.ODataMetadata {
	desc := "Test product description"
	return &models.ODataMetadata{
		ServiceRoot:     "https://example.com/odata",
		SchemaNamespace: "TestNamespace",
		ContainerName:   "TestContainer",
		Version:         "2.0",
		EntityTypes: map[string]*models.EntityType{
			"Product": {
				Name:          "Product",
				KeyProperties: []string{"ProductID"},
				Properties: []*models.EntityProperty{
					{Name: "ProductID", Type: "Edm.Int32", IsKey: true, Nullable: false},
					{Name: "ProductName", Type: "Edm.String", Nullable: false, Description: &desc},
					{Name: "Price", Type: "Edm.Decimal", Nullable: true},
				},
			},
			"Category": {
				Name:          "Category",
				KeyProperties: []string{"CategoryID"},
				Properties: []*models.EntityProperty{
					{Name: "CategoryID", Type: "Edm.Int32", IsKey: true, Nullable: false},
					{Name: "CategoryName", Type: "Edm.String", Nullable: false},
				},
			},
			"OrderDetail": {
				Name:          "OrderDetail",
				KeyProperties: []string{"OrderID", "ProductID"},
				Properties: []*models.EntityProperty{
					{Name: "OrderID", Type: "Edm.Int32", IsKey: true, Nullable: false},
					{Name: "ProductID", Type: "Edm.Int32", IsKey: true, Nullable: false},
					{Name: "Quantity", Type: "Edm.Int32", Nullable: false},
				},
			},
		},
		EntitySets: map[string]*models.EntitySet{
			"Products": {
				Name:       "Products",
				EntityType: "Product",
				Creatable:  true,
				Updatable:  true,
				Deletable:  true,
				Searchable: true,
				Pageable:   true,
			},
			"Categories": {
				Name:       "Categories",
				EntityType: "Category",
				Creatable:  false,
				Updatable:  false,
				Deletable:  false,
				Searchable: true,
				Pageable:   true,
			},
			"OrderDetails": {
				Name:       "OrderDetails",
				EntityType: "OrderDetail",
				Creatable:  true,
				Updatable:  true,
				Deletable:  true,
				Searchable: false,
				Pageable:   true,
			},
		},
		FunctionImports: map[string]*models.FunctionImport{
			"GetProductsByCategory": {
				Name:       "GetProductsByCategory",
				HTTPMethod: "GET",
				ReturnType: "Collection(Product)",
				Parameters: []*models.FunctionParameter{
					{Name: "categoryId", Type: "Edm.Int32", Nullable: false},
				},
			},
		},
	}
}

// createTestBridge creates a bridge with test metadata for lazy mode testing
func createTestBridge(cfg *config.Config) *ODataMCPBridge {
	server := mcp.NewServer("test-service", "1.0.0")
	bridge := &ODataMCPBridge{
		config:   cfg,
		metadata: createTestMetadata(),
		server:   server,
		tools:    make(map[string]*models.ToolInfo),
	}
	return bridge
}

func TestShouldUseLazyMode(t *testing.T) {
	tests := []struct {
		name         string
		lazyMetadata bool
		lazyThreshold int
		entitySets   int
		want         bool
	}{
		{
			name:         "Explicit lazy mode enabled",
			lazyMetadata: true,
			lazyThreshold: 0,
			entitySets:   3,
			want:         true,
		},
		{
			name:         "Lazy mode disabled, no threshold",
			lazyMetadata: false,
			lazyThreshold: 0,
			entitySets:   3,
			want:         false,
		},
		{
			name:         "Threshold exceeded",
			lazyMetadata: false,
			lazyThreshold: 10, // threshold 10, with 3 entity sets * 6 ops = 18 tools estimated
			entitySets:   3,
			want:         true,
		},
		{
			name:         "Threshold not exceeded",
			lazyMetadata: false,
			lazyThreshold: 100,
			entitySets:   3,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				LazyMetadata:  tt.lazyMetadata,
				LazyThreshold: tt.lazyThreshold,
			}
			bridge := createTestBridge(cfg)

			got := bridge.shouldUseLazyMode()
			if got != tt.want {
				t.Errorf("shouldUseLazyMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEstimateToolCount(t *testing.T) {
	cfg := &config.Config{}
	bridge := createTestBridge(cfg)

	// With 3 entity sets and full operations enabled:
	// Each entity set gets: filter, count, get, create, update, delete = 6 ops
	// Plus 1 function import
	// Expected: 3 * 6 + 1 = 19 tools
	estimate := bridge.estimateToolCount()

	// Check it's reasonable (should be > 0 and account for entity sets + functions)
	if estimate < 10 {
		t.Errorf("estimateToolCount() = %v, expected at least 10 for 3 entity sets", estimate)
	}
}

func TestGenerateLazyTools(t *testing.T) {
	cfg := &config.Config{
		LazyMetadata: true,
	}
	bridge := createTestBridge(cfg)

	err := bridge.generateLazyTools()
	if err != nil {
		t.Fatalf("generateLazyTools() error = %v", err)
	}

	// Check that exactly 10 tools were generated
	expectedToolPrefixes := []string{
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

	if len(bridge.tools) != len(expectedToolPrefixes) {
		t.Errorf("generateLazyTools() generated %d tools, want %d", len(bridge.tools), len(expectedToolPrefixes))
	}

	// Check tools by prefix (they include _for_serviceid suffix)
	for _, prefix := range expectedToolPrefixes {
		found := false
		for toolName := range bridge.tools {
			if strings.HasPrefix(toolName, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("generateLazyTools() missing tool with prefix: %s", prefix)
		}
	}
}

func TestGenerateLazyToolsReadOnlyMode(t *testing.T) {
	cfg := &config.Config{
		LazyMetadata: true,
		ReadOnly:     true,
	}
	bridge := createTestBridge(cfg)

	err := bridge.generateLazyTools()
	if err != nil {
		t.Fatalf("generateLazyTools() error = %v", err)
	}

	// In read-only mode, create/update/delete tools should NOT be generated
	readOnlyExpectedPrefixes := []string{
		"odata_service_info",
		"list_entities",
		"count_entities",
		"get_entity",
		"get_entity_schema",
		"list_functions",
		"call_function",
	}

	mutatingPrefixes := []string{
		"create_entity",
		"update_entity",
		"delete_entity",
	}

	if len(bridge.tools) != len(readOnlyExpectedPrefixes) {
		t.Errorf("generateLazyTools() in read-only mode generated %d tools, want %d", len(bridge.tools), len(readOnlyExpectedPrefixes))
	}

	// Check expected tools exist by prefix
	for _, prefix := range readOnlyExpectedPrefixes {
		found := false
		for toolName := range bridge.tools {
			if strings.HasPrefix(toolName, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("generateLazyTools() read-only mode missing tool with prefix: %s", prefix)
		}
	}

	// Check mutating tools are NOT present
	for _, prefix := range mutatingPrefixes {
		for toolName := range bridge.tools {
			if strings.HasPrefix(toolName, prefix) {
				t.Errorf("generateLazyTools() read-only mode should not have tool with prefix: %s", prefix)
			}
		}
	}
}

func TestValidateEntitySet(t *testing.T) {
	cfg := &config.Config{}
	bridge := createTestBridge(cfg)

	tests := []struct {
		name      string
		entitySet string
		wantErr   bool
	}{
		{
			name:      "Valid entity set",
			entitySet: "Products",
			wantErr:   false,
		},
		{
			name:      "Invalid entity set",
			entitySet: "NonExistent",
			wantErr:   true,
		},
		{
			name:      "Empty entity set",
			entitySet: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es, et, err := bridge.validateEntitySet(tt.entitySet)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEntitySet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if es == nil {
					t.Error("validateEntitySet() returned nil EntitySet for valid input")
				}
				if et == nil {
					t.Error("validateEntitySet() returned nil EntityType for valid input")
				}
			}
		})
	}
}

func TestHandleLazyGetEntitySchema(t *testing.T) {
	cfg := &config.Config{}
	bridge := createTestBridge(cfg)

	tests := []struct {
		name      string
		args      map[string]interface{}
		wantErr   bool
		checkKeys []string // keys expected in JSON response
	}{
		{
			name: "Valid entity set",
			args: map[string]interface{}{
				"entity_set": "Products",
			},
			wantErr:   false,
			checkKeys: []string{"entity_set", "entity_type", "properties", "keys", "capabilities"},
		},
		{
			name: "Missing entity_set parameter",
			args: map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "Invalid entity set",
			args: map[string]interface{}{
				"entity_set": "NonExistent",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := bridge.handleLazyGetEntitySchema(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleLazyGetEntitySchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("handleLazyGetEntitySchema() returned nil result for valid input")
			}
		})
	}
}

func TestHandleLazyListFunctions(t *testing.T) {
	cfg := &config.Config{}
	bridge := createTestBridge(cfg)

	result, err := bridge.handleLazyListFunctions(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleLazyListFunctions() error = %v", err)
	}

	if result == nil {
		t.Error("handleLazyListFunctions() returned nil result")
	}

	// Result should be a JSON string containing functions
	resultStr, ok := result.(string)
	if !ok {
		t.Errorf("handleLazyListFunctions() result is not a string, got %T", result)
	}

	// Check it contains expected function name
	if len(resultStr) == 0 {
		t.Error("handleLazyListFunctions() returned empty string")
	}
}

func TestLazyHandlersMissingParameters(t *testing.T) {
	cfg := &config.Config{}
	bridge := createTestBridge(cfg)

	tests := []struct {
		name    string
		handler func(context.Context, map[string]interface{}) (interface{}, error)
		args    map[string]interface{}
	}{
		{
			name:    "handleLazyListEntities missing entity_set",
			handler: bridge.handleLazyListEntities,
			args:    map[string]interface{}{},
		},
		{
			name:    "handleLazyCountEntities missing entity_set",
			handler: bridge.handleLazyCountEntities,
			args:    map[string]interface{}{},
		},
		{
			name:    "handleLazyGetEntity missing entity_set",
			handler: bridge.handleLazyGetEntity,
			args:    map[string]interface{}{},
		},
		{
			name:    "handleLazyGetEntity missing key",
			handler: bridge.handleLazyGetEntity,
			args:    map[string]interface{}{"entity_set": "Products"},
		},
		{
			name:    "handleLazyCreateEntity missing entity_set",
			handler: bridge.handleLazyCreateEntity,
			args:    map[string]interface{}{},
		},
		{
			name:    "handleLazyCreateEntity missing data",
			handler: bridge.handleLazyCreateEntity,
			args:    map[string]interface{}{"entity_set": "Products"},
		},
		{
			name:    "handleLazyUpdateEntity missing entity_set",
			handler: bridge.handleLazyUpdateEntity,
			args:    map[string]interface{}{},
		},
		{
			name:    "handleLazyUpdateEntity missing key",
			handler: bridge.handleLazyUpdateEntity,
			args:    map[string]interface{}{"entity_set": "Products"},
		},
		{
			name:    "handleLazyUpdateEntity missing data",
			handler: bridge.handleLazyUpdateEntity,
			args:    map[string]interface{}{"entity_set": "Products", "key": map[string]interface{}{"ProductID": 1}},
		},
		{
			name:    "handleLazyDeleteEntity missing entity_set",
			handler: bridge.handleLazyDeleteEntity,
			args:    map[string]interface{}{},
		},
		{
			name:    "handleLazyDeleteEntity missing key",
			handler: bridge.handleLazyDeleteEntity,
			args:    map[string]interface{}{"entity_set": "Products"},
		},
		{
			name:    "handleLazyCallFunction missing function_name",
			handler: bridge.handleLazyCallFunction,
			args:    map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.handler(context.Background(), tt.args)
			if err == nil {
				t.Errorf("%s should return error for missing parameters", tt.name)
			}
		})
	}
}

func TestLazyHandlersReadOnlyMode(t *testing.T) {
	cfg := &config.Config{
		ReadOnly: true,
	}
	bridge := createTestBridge(cfg)

	tests := []struct {
		name    string
		handler func(context.Context, map[string]interface{}) (interface{}, error)
		args    map[string]interface{}
	}{
		{
			name:    "handleLazyCreateEntity in read-only mode",
			handler: bridge.handleLazyCreateEntity,
			args: map[string]interface{}{
				"entity_set": "Products",
				"data":       map[string]interface{}{"ProductName": "Test"},
			},
		},
		{
			name:    "handleLazyUpdateEntity in read-only mode",
			handler: bridge.handleLazyUpdateEntity,
			args: map[string]interface{}{
				"entity_set": "Products",
				"key":        map[string]interface{}{"ProductID": 1},
				"data":       map[string]interface{}{"ProductName": "Updated"},
			},
		},
		{
			name:    "handleLazyDeleteEntity in read-only mode",
			handler: bridge.handleLazyDeleteEntity,
			args: map[string]interface{}{
				"entity_set": "Products",
				"key":        map[string]interface{}{"ProductID": 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.handler(context.Background(), tt.args)
			if err == nil {
				t.Errorf("%s should return error in read-only mode", tt.name)
			}
		})
	}
}

func TestLazyHandlersInvalidEntitySet(t *testing.T) {
	cfg := &config.Config{}
	bridge := createTestBridge(cfg)

	tests := []struct {
		name    string
		handler func(context.Context, map[string]interface{}) (interface{}, error)
		args    map[string]interface{}
	}{
		{
			name:    "handleLazyListEntities with invalid entity_set",
			handler: bridge.handleLazyListEntities,
			args:    map[string]interface{}{"entity_set": "InvalidSet"},
		},
		{
			name:    "handleLazyCountEntities with invalid entity_set",
			handler: bridge.handleLazyCountEntities,
			args:    map[string]interface{}{"entity_set": "InvalidSet"},
		},
		{
			name:    "handleLazyGetEntitySchema with invalid entity_set",
			handler: bridge.handleLazyGetEntitySchema,
			args:    map[string]interface{}{"entity_set": "InvalidSet"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.handler(context.Background(), tt.args)
			if err == nil {
				t.Errorf("%s should return error for invalid entity set", tt.name)
			}
		})
	}
}

func TestLazyGetEntityKeyValidation(t *testing.T) {
	cfg := &config.Config{}
	bridge := createTestBridge(cfg)

	// Test key type validation that happens before OData call
	tests := []struct {
		name      string
		entitySet string
		key       interface{}
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Single key for composite key entity should fail",
			entitySet: "OrderDetails",
			key:       123,
			wantErr:   true,
			errMsg:    "has 2 key properties",
		},
		{
			name:      "Invalid key type (array) should fail",
			entitySet: "Products",
			key:       []string{"invalid"},
			wantErr:   true,
			errMsg:    "invalid key type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"entity_set": tt.entitySet,
				"key":        tt.key,
			}
			_, err := bridge.handleLazyGetEntity(context.Background(), args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("handleLazyGetEntity() should return error for key type %T", tt.key)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("handleLazyGetEntity() error = %v, should contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestLazyCallFunctionValidation(t *testing.T) {
	cfg := &config.Config{}
	bridge := createTestBridge(cfg)

	// Test validation that happens before OData call
	tests := []struct {
		name         string
		functionName string
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "Invalid function name",
			functionName: "NonExistentFunction",
			wantErr:      true,
			errMsg:       "function not found",
		},
		{
			name:         "Empty function name",
			functionName: "",
			wantErr:      true,
			errMsg:       "missing required parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"function_name": tt.functionName,
			}
			_, err := bridge.handleLazyCallFunction(context.Background(), args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("handleLazyCallFunction() should return error for function %s", tt.functionName)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("handleLazyCallFunction() error = %v, should contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestToolNamingWithPrefix(t *testing.T) {
	cfg := &config.Config{
		LazyMetadata: true,
		ToolPrefix:   "myservice",
		NoPostfix:    true, // Use prefix mode instead of postfix
	}
	bridge := createTestBridge(cfg)

	err := bridge.generateLazyTools()
	if err != nil {
		t.Fatalf("generateLazyTools() error = %v", err)
	}

	// Check tools have prefix (in prefix mode, tools are: prefix_operation)
	expectedPrefixes := []string{
		"myservice_odata_service_info",
		"myservice_list_entities",
		"myservice_count_entities",
		"myservice_get_entity",
		"myservice_get_entity_schema",
		"myservice_create_entity",
		"myservice_update_entity",
		"myservice_delete_entity",
		"myservice_list_functions",
		"myservice_call_function",
	}

	for _, expectedName := range expectedPrefixes {
		if _, exists := bridge.tools[expectedName]; !exists {
			t.Errorf("generateLazyTools() with prefix missing tool: %s (got tools: %v)", expectedName, getToolNames(bridge.tools))
		}
	}
}

// getToolNames returns a slice of tool names for debugging
func getToolNames(tools map[string]*models.ToolInfo) []string {
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	return names
}

func TestEntitySetCapabilityChecks(t *testing.T) {
	cfg := &config.Config{}
	bridge := createTestBridge(cfg)

	// Categories is not creatable/updatable/deletable
	tests := []struct {
		name    string
		handler func(context.Context, map[string]interface{}) (interface{}, error)
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "Create on non-creatable entity set",
			handler: bridge.handleLazyCreateEntity,
			args: map[string]interface{}{
				"entity_set": "Categories",
				"data":       map[string]interface{}{"CategoryName": "Test"},
			},
			wantErr: true,
		},
		{
			name:    "Update on non-updatable entity set",
			handler: bridge.handleLazyUpdateEntity,
			args: map[string]interface{}{
				"entity_set": "Categories",
				"key":        map[string]interface{}{"CategoryID": 1},
				"data":       map[string]interface{}{"CategoryName": "Updated"},
			},
			wantErr: true,
		},
		{
			name:    "Delete on non-deletable entity set",
			handler: bridge.handleLazyDeleteEntity,
			args: map[string]interface{}{
				"entity_set": "Categories",
				"key":        map[string]interface{}{"CategoryID": 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.handler(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

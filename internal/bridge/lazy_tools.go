// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package bridge

import (
	"context"
	"fmt"

	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/mcp"
	"github.com/zmcp/odata-mcp/internal/models"
)

// generateLazyTools creates 10 generic MCP tools for lazy metadata mode
// instead of generating per-entity tools (500+ for large SAP services).
// These tools accept entity_set as a parameter for dynamic entity resolution.
// Respects --enable/--disable operation filters and --read-only mode.
func (b *ODataMCPBridge) generateLazyTools() error {
	// 1. Service info tool (always generated)
	if err := b.generateLazyServiceInfoTool(); err != nil {
		return fmt.Errorf("failed to generate lazy service info tool: %w", err)
	}

	// 2. List entities tool (filter operation - 'F')
	if b.config.IsOperationEnabled('F') {
		if err := b.generateLazyListEntitiesTool(); err != nil {
			return fmt.Errorf("failed to generate lazy list entities tool: %w", err)
		}
	}

	// 3. Count entities tool (filter operation - 'F')
	if b.config.IsOperationEnabled('F') {
		if err := b.generateLazyCountEntitiesTool(); err != nil {
			return fmt.Errorf("failed to generate lazy count entities tool: %w", err)
		}
	}

	// 4. Get entity tool (get operation - 'G')
	if b.config.IsOperationEnabled('G') {
		if err := b.generateLazyGetEntityTool(); err != nil {
			return fmt.Errorf("failed to generate lazy get entity tool: %w", err)
		}
	}

	// 5. Get entity schema tool (always generated - metadata inspection)
	if err := b.generateLazyGetEntitySchemaTool(); err != nil {
		return fmt.Errorf("failed to generate lazy get entity schema tool: %w", err)
	}

	// 6. Create entity tool (create operation - 'C')
	if !b.config.IsReadOnly() && b.config.IsOperationEnabled('C') {
		if err := b.generateLazyCreateEntityTool(); err != nil {
			return fmt.Errorf("failed to generate lazy create entity tool: %w", err)
		}
	}

	// 7. Update entity tool (update operation - 'U')
	if !b.config.IsReadOnly() && b.config.IsOperationEnabled('U') {
		if err := b.generateLazyUpdateEntityTool(); err != nil {
			return fmt.Errorf("failed to generate lazy update entity tool: %w", err)
		}
	}

	// 8. Delete entity tool (delete operation - 'D')
	if !b.config.IsReadOnly() && b.config.IsOperationEnabled('D') {
		if err := b.generateLazyDeleteEntityTool(); err != nil {
			return fmt.Errorf("failed to generate lazy delete entity tool: %w", err)
		}
	}

	// 9. List functions tool (actions operation - 'A')
	if b.config.IsOperationEnabled('A') {
		if err := b.generateLazyListFunctionsTool(); err != nil {
			return fmt.Errorf("failed to generate lazy list functions tool: %w", err)
		}
	}

	// 10. Call function tool (actions operation - 'A')
	if b.config.IsOperationEnabled('A') {
		if err := b.generateLazyCallFunctionTool(); err != nil {
			return fmt.Errorf("failed to generate lazy call function tool: %w", err)
		}
	}

	return nil
}

// generateLazyServiceInfoTool creates the odata_service_info tool
func (b *ODataMCPBridge) generateLazyServiceInfoTool() error {
	toolName := b.formatToolName("odata_service_info", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Get comprehensive information about the OData service including metadata, entity list, and function list",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"include_metadata": map[string]interface{}{
					"type":        "boolean",
					"description": "Include detailed metadata information for all entity types and sets",
					"default":     false,
				},
			},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleServiceInfo(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   constants.OpInfo,
	}

	return nil
}

// generateLazyListEntitiesTool creates the list_entities generic tool
func (b *ODataMCPBridge) generateLazyListEntitiesTool() error {
	toolName := b.formatToolName("list_entities", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "List/filter entities from any entity set with OData query options",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_set": map[string]interface{}{
					"type":        "string",
					"description": "Name of the entity set to query (e.g., 'Products', 'Customers')",
				},
				b.getParameterName("$filter"): map[string]interface{}{
					"type":        "string",
					"description": "OData filter expression to filter results",
				},
				b.getParameterName("$select"): map[string]interface{}{
					"type":        "string",
					"description": "Comma-separated list of properties to select",
				},
				b.getParameterName("$expand"): map[string]interface{}{
					"type":        "string",
					"description": "Navigation properties to expand",
				},
				b.getParameterName("$orderby"): map[string]interface{}{
					"type":        "string",
					"description": "Properties to order by with optional asc/desc",
				},
				b.getParameterName("$top"): map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of entities to return",
				},
				b.getParameterName("$skip"): map[string]interface{}{
					"type":        "integer",
					"description": "Number of entities to skip for pagination",
				},
				b.getParameterName("$count"): map[string]interface{}{
					"type":        "boolean",
					"description": "Include total count of matching entities",
				},
			},
			"required": []string{"entity_set"},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleLazyListEntities(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   constants.OpFilter,
	}

	return nil
}

// generateLazyCountEntitiesTool creates the count_entities generic tool
func (b *ODataMCPBridge) generateLazyCountEntitiesTool() error {
	toolName := b.formatToolName("count_entities", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Get count of entities in any entity set with optional filter",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_set": map[string]interface{}{
					"type":        "string",
					"description": "Name of the entity set to count (e.g., 'Products', 'Customers')",
				},
				b.getParameterName("$filter"): map[string]interface{}{
					"type":        "string",
					"description": "OData filter expression to filter results before counting",
				},
			},
			"required": []string{"entity_set"},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleLazyCountEntities(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   constants.OpCount,
	}

	return nil
}

// generateLazyGetEntityTool creates the get_entity generic tool
func (b *ODataMCPBridge) generateLazyGetEntityTool() error {
	toolName := b.formatToolName("get_entity", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Get a single entity by key from any entity set",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_set": map[string]interface{}{
					"type":        "string",
					"description": "Name of the entity set (e.g., 'Products', 'Customers')",
				},
				"key": map[string]interface{}{
					"type":        "object",
					"description": "Key properties and values as a JSON object (e.g., {\"ProductID\": 1} or {\"OrderID\": 123, \"LineNumber\": 1})",
				},
				b.getParameterName("$select"): map[string]interface{}{
					"type":        "string",
					"description": "Comma-separated list of properties to select",
				},
				b.getParameterName("$expand"): map[string]interface{}{
					"type":        "string",
					"description": "Navigation properties to expand",
				},
			},
			"required": []string{"entity_set", "key"},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleLazyGetEntity(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   constants.OpGet,
	}

	return nil
}

// generateLazyGetEntitySchemaTool creates the get_entity_schema generic tool
func (b *ODataMCPBridge) generateLazyGetEntitySchemaTool() error {
	toolName := b.formatToolName("get_entity_schema", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Get the schema (properties, types, keys) for any entity set",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_set": map[string]interface{}{
					"type":        "string",
					"description": "Name of the entity set to get schema for (e.g., 'Products', 'Customers')",
				},
			},
			"required": []string{"entity_set"},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleLazyGetEntitySchema(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   "schema",
	}

	return nil
}

// generateLazyCreateEntityTool creates the create_entity generic tool
func (b *ODataMCPBridge) generateLazyCreateEntityTool() error {
	toolName := b.formatToolName("create_entity", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Create a new entity in any entity set",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_set": map[string]interface{}{
					"type":        "string",
					"description": "Name of the entity set to create in (e.g., 'Products', 'Customers')",
				},
				"data": map[string]interface{}{
					"type":        "object",
					"description": "Entity data as a JSON object with property names and values",
				},
			},
			"required": []string{"entity_set", "data"},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleLazyCreateEntity(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   constants.OpCreate,
	}

	return nil
}

// generateLazyUpdateEntityTool creates the update_entity generic tool
func (b *ODataMCPBridge) generateLazyUpdateEntityTool() error {
	toolName := b.formatToolName("update_entity", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Update an existing entity in any entity set",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_set": map[string]interface{}{
					"type":        "string",
					"description": "Name of the entity set (e.g., 'Products', 'Customers')",
				},
				"key": map[string]interface{}{
					"type":        "object",
					"description": "Key properties and values as a JSON object (e.g., {\"ProductID\": 1})",
				},
				"data": map[string]interface{}{
					"type":        "object",
					"description": "Updated property values as a JSON object",
				},
				"_method": map[string]interface{}{
					"type":        "string",
					"description": "HTTP method to use (PUT, PATCH, or MERGE)",
					"enum":        []string{"PUT", "PATCH", "MERGE"},
					"default":     "PUT",
				},
			},
			"required": []string{"entity_set", "key", "data"},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleLazyUpdateEntity(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   constants.OpUpdate,
	}

	return nil
}

// generateLazyDeleteEntityTool creates the delete_entity generic tool
func (b *ODataMCPBridge) generateLazyDeleteEntityTool() error {
	toolName := b.formatToolName("delete_entity", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Delete an entity from any entity set",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_set": map[string]interface{}{
					"type":        "string",
					"description": "Name of the entity set (e.g., 'Products', 'Customers')",
				},
				"key": map[string]interface{}{
					"type":        "object",
					"description": "Key properties and values as a JSON object (e.g., {\"ProductID\": 1})",
				},
			},
			"required": []string{"entity_set", "key"},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleLazyDeleteEntity(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   constants.OpDelete,
	}

	return nil
}

// generateLazyListFunctionsTool creates the list_functions generic tool
func (b *ODataMCPBridge) generateLazyListFunctionsTool() error {
	toolName := b.formatToolName("list_functions", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "List all available function imports and actions with their parameters and descriptions",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleLazyListFunctions(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   "list_functions",
	}

	return nil
}

// generateLazyCallFunctionTool creates the call_function generic tool
func (b *ODataMCPBridge) generateLazyCallFunctionTool() error {
	toolName := b.formatToolName("call_function", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Call any function import or action by name with parameters",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"function_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the function import or action to call",
				},
				"params": map[string]interface{}{
					"type":        "object",
					"description": "Function parameters as a JSON object with parameter names and values",
					"default":     map[string]interface{}{},
				},
			},
			"required": []string{"function_name"},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleLazyCallFunction(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   "call_function",
	}

	return nil
}

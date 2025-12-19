// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package bridge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zmcp/odata-mcp/internal/models"
)

// validateEntitySet validates that an entity set exists and is allowed by filters
func (b *ODataMCPBridge) validateEntitySet(entitySet string) (*models.EntitySet, *models.EntityType, error) {
	// Check if entity set exists
	es, exists := b.metadata.EntitySets[entitySet]
	if !exists {
		return nil, nil, fmt.Errorf("entity set not found: %s", entitySet)
	}

	// Check if entity set is allowed by --entities filter
	if !b.shouldIncludeEntity(entitySet) {
		return nil, nil, fmt.Errorf("entity set not allowed: %s (restricted by --entities filter)", entitySet)
	}

	// Get the entity type
	et, exists := b.metadata.EntityTypes[es.EntityType]
	if !exists {
		return nil, nil, fmt.Errorf("entity type not found for entity set %s: %s", entitySet, es.EntityType)
	}

	return es, et, nil
}

// handleLazyListEntities handles lazy mode list/filter operations
func (b *ODataMCPBridge) handleLazyListEntities(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract entity_set parameter
	entitySet, ok := args["entity_set"].(string)
	if !ok || entitySet == "" {
		return nil, fmt.Errorf("missing required parameter: entity_set")
	}

	// Validate entity set exists
	_, _, err := b.validateEntitySet(entitySet)
	if err != nil {
		return nil, err
	}

	// Remove entity_set from args before delegating
	delete(args, "entity_set")

	// Delegate to existing handler
	return b.handleEntityFilter(ctx, entitySet, args)
}

// handleLazyCountEntities handles lazy mode count operations
func (b *ODataMCPBridge) handleLazyCountEntities(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract entity_set parameter
	entitySet, ok := args["entity_set"].(string)
	if !ok || entitySet == "" {
		return nil, fmt.Errorf("missing required parameter: entity_set")
	}

	// Validate entity set exists
	_, _, err := b.validateEntitySet(entitySet)
	if err != nil {
		return nil, err
	}

	// Remove entity_set from args before delegating
	delete(args, "entity_set")

	// Delegate to existing handler
	return b.handleEntityCount(ctx, entitySet, args)
}

// handleLazyGetEntity handles lazy mode get entity operations
func (b *ODataMCPBridge) handleLazyGetEntity(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract entity_set parameter
	entitySet, ok := args["entity_set"].(string)
	if !ok || entitySet == "" {
		return nil, fmt.Errorf("missing required parameter: entity_set")
	}

	// Extract key parameter
	key, ok := args["key"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: key")
	}

	// Validate entity set and get entity type
	_, entityType, err := b.validateEntitySet(entitySet)
	if err != nil {
		return nil, err
	}

	// Handle key parameter - can be a single value or a map for composite keys
	keyMap := make(map[string]interface{})
	switch k := key.(type) {
	case map[string]interface{}:
		// Composite key provided as map
		keyMap = k
	case string, float64, int, int64, bool:
		// Single key value - map to the first (and should be only) key property
		if len(entityType.KeyProperties) != 1 {
			return nil, fmt.Errorf("single key value provided but entity has %d key properties", len(entityType.KeyProperties))
		}
		keyMap[entityType.KeyProperties[0]] = k
	default:
		return nil, fmt.Errorf("invalid key type: %T", key)
	}

	// Build merged args with key properties
	mergedArgs := make(map[string]interface{})
	for k, v := range args {
		if k != "entity_set" && k != "key" {
			mergedArgs[k] = v
		}
	}
	for k, v := range keyMap {
		mergedArgs[k] = v
	}

	// Delegate to existing handler
	return b.handleEntityGet(ctx, entitySet, entityType, mergedArgs)
}

// handleLazyGetEntitySchema returns the full schema for an entity set
func (b *ODataMCPBridge) handleLazyGetEntitySchema(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract entity_set parameter
	entitySet, ok := args["entity_set"].(string)
	if !ok || entitySet == "" {
		return nil, fmt.Errorf("missing required parameter: entity_set")
	}

	// Validate entity set and get entity type
	es, et, err := b.validateEntitySet(entitySet)
	if err != nil {
		return nil, err
	}

	// Build schema response
	schema := map[string]interface{}{
		"entity_set":  entitySet,
		"entity_type": es.EntityType,
		"namespace":   b.metadata.SchemaNamespace,
		"capabilities": map[string]interface{}{
			"creatable":  es.Creatable,
			"updatable":  es.Updatable,
			"deletable":  es.Deletable,
			"searchable": es.Searchable,
			"pageable":   es.Pageable,
		},
		"properties": make([]map[string]interface{}, 0, len(et.Properties)),
		"keys":       et.KeyProperties,
	}

	// Add property details
	properties := make([]map[string]interface{}, 0, len(et.Properties))
	for _, prop := range et.Properties {
		propSchema := map[string]interface{}{
			"name":     prop.Name,
			"type":     prop.Type,
			"nullable": prop.Nullable,
			"is_key":   prop.IsKey,
		}
		if prop.Description != nil {
			propSchema["description"] = *prop.Description
		}
		properties = append(properties, propSchema)
	}
	schema["properties"] = properties

	// Add navigation properties if any
	if len(et.NavigationProps) > 0 {
		navProps := make([]map[string]interface{}, 0, len(et.NavigationProps))
		for _, nav := range et.NavigationProps {
			navSchema := map[string]interface{}{
				"name":     nav.Name,
				"nullable": nav.Nullable,
			}
			if nav.Type != "" {
				navSchema["type"] = nav.Type
			}
			if nav.Relationship != "" {
				navSchema["relationship"] = nav.Relationship
			}
			if nav.ToRole != "" {
				navSchema["to_role"] = nav.ToRole
			}
			if nav.FromRole != "" {
				navSchema["from_role"] = nav.FromRole
			}
			if nav.Partner != "" {
				navSchema["partner"] = nav.Partner
			}
			navProps = append(navProps, navSchema)
		}
		schema["navigation_properties"] = navProps
	}

	// Format response as JSON string
	result, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to format schema: %w", err)
	}

	return string(result), nil
}

// handleLazyCreateEntity handles lazy mode create operations
func (b *ODataMCPBridge) handleLazyCreateEntity(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract entity_set parameter
	entitySet, ok := args["entity_set"].(string)
	if !ok || entitySet == "" {
		return nil, fmt.Errorf("missing required parameter: entity_set")
	}

	// Extract data parameter
	data, ok := args["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing required parameter: data")
	}

	// Check read-only mode
	if b.config.IsReadOnly() {
		return nil, fmt.Errorf("create operation not allowed in read-only mode")
	}

	// Validate entity set exists
	es, _, err := b.validateEntitySet(entitySet)
	if err != nil {
		return nil, err
	}

	// Check if entity set is creatable
	if !es.Creatable {
		return nil, fmt.Errorf("entity set %s is not creatable", entitySet)
	}

	// Delegate to existing handler
	return b.handleEntityCreate(ctx, entitySet, data)
}

// handleLazyUpdateEntity handles lazy mode update operations
func (b *ODataMCPBridge) handleLazyUpdateEntity(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract entity_set parameter
	entitySet, ok := args["entity_set"].(string)
	if !ok || entitySet == "" {
		return nil, fmt.Errorf("missing required parameter: entity_set")
	}

	// Extract key parameter
	key, ok := args["key"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: key")
	}

	// Extract data parameter
	data, ok := args["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing required parameter: data")
	}

	// Check read-only mode
	if b.config.IsReadOnly() {
		return nil, fmt.Errorf("update operation not allowed in read-only mode")
	}

	// Validate entity set and get entity type
	es, entityType, err := b.validateEntitySet(entitySet)
	if err != nil {
		return nil, err
	}

	// Check if entity set is updatable
	if !es.Updatable {
		return nil, fmt.Errorf("entity set %s is not updatable", entitySet)
	}

	// Handle key parameter - can be a single value or a map for composite keys
	keyMap := make(map[string]interface{})
	switch k := key.(type) {
	case map[string]interface{}:
		// Composite key provided as map
		keyMap = k
	case string, float64, int, int64, bool:
		// Single key value - map to the first (and should be only) key property
		if len(entityType.KeyProperties) != 1 {
			return nil, fmt.Errorf("single key value provided but entity has %d key properties", len(entityType.KeyProperties))
		}
		keyMap[entityType.KeyProperties[0]] = k
	default:
		return nil, fmt.Errorf("invalid key type: %T", key)
	}

	// Merge key and data for the handler
	mergedArgs := make(map[string]interface{})
	for k, v := range keyMap {
		mergedArgs[k] = v
	}
	for k, v := range data {
		mergedArgs[k] = v
	}

	// Include _method if provided
	if method, ok := args["_method"].(string); ok {
		mergedArgs["_method"] = method
	}

	// Delegate to existing handler
	return b.handleEntityUpdate(ctx, entitySet, entityType, mergedArgs)
}

// handleLazyDeleteEntity handles lazy mode delete operations
func (b *ODataMCPBridge) handleLazyDeleteEntity(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract entity_set parameter
	entitySet, ok := args["entity_set"].(string)
	if !ok || entitySet == "" {
		return nil, fmt.Errorf("missing required parameter: entity_set")
	}

	// Extract key parameter
	key, ok := args["key"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: key")
	}

	// Check read-only mode
	if b.config.IsReadOnly() {
		return nil, fmt.Errorf("delete operation not allowed in read-only mode")
	}

	// Validate entity set and get entity type
	es, entityType, err := b.validateEntitySet(entitySet)
	if err != nil {
		return nil, err
	}

	// Check if entity set is deletable
	if !es.Deletable {
		return nil, fmt.Errorf("entity set %s is not deletable", entitySet)
	}

	// Handle key parameter - can be a single value or a map for composite keys
	keyMap := make(map[string]interface{})
	switch k := key.(type) {
	case map[string]interface{}:
		// Composite key provided as map
		keyMap = k
	case string, float64, int, int64, bool:
		// Single key value - map to the first (and should be only) key property
		if len(entityType.KeyProperties) != 1 {
			return nil, fmt.Errorf("single key value provided but entity has %d key properties", len(entityType.KeyProperties))
		}
		keyMap[entityType.KeyProperties[0]] = k
	default:
		return nil, fmt.Errorf("invalid key type: %T", key)
	}

	// Delegate to existing handler
	return b.handleEntityDelete(ctx, entitySet, entityType, keyMap)
}

// handleLazyListFunctions returns a list of available function imports
func (b *ODataMCPBridge) handleLazyListFunctions(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	functions := make([]map[string]interface{}, 0, len(b.metadata.FunctionImports))

	for name, fn := range b.metadata.FunctionImports {
		// Skip functions that are filtered out
		if !b.shouldIncludeFunction(name) {
			continue
		}

		funcInfo := map[string]interface{}{
			"name":        name,
			"http_method": fn.HTTPMethod,
			"is_bound":    fn.IsBound,
			"is_action":   fn.IsAction,
		}

		if fn.ReturnType != "" {
			funcInfo["return_type"] = fn.ReturnType
		}

		if fn.Description != nil {
			funcInfo["description"] = *fn.Description
		}

		// Add parameters
		if len(fn.Parameters) > 0 {
			params := make([]map[string]interface{}, 0, len(fn.Parameters))
			for _, param := range fn.Parameters {
				paramInfo := map[string]interface{}{
					"name":     param.Name,
					"type":     param.Type,
					"nullable": param.Nullable,
				}
				if param.Mode != "" {
					paramInfo["mode"] = param.Mode
				}
				params = append(params, paramInfo)
			}
			funcInfo["parameters"] = params
		}

		functions = append(functions, funcInfo)
	}

	// Format response as JSON string
	result, err := json.Marshal(map[string]interface{}{
		"functions": functions,
		"count":     len(functions),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to format functions list: %w", err)
	}

	return string(result), nil
}

// handleLazyCallFunction handles lazy mode function call operations
func (b *ODataMCPBridge) handleLazyCallFunction(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Extract function_name parameter
	functionName, ok := args["function_name"].(string)
	if !ok || functionName == "" {
		return nil, fmt.Errorf("missing required parameter: function_name")
	}

	// Extract params parameter (optional, default to empty map)
	params := make(map[string]interface{})
	if p, ok := args["params"].(map[string]interface{}); ok {
		params = p
	}

	// Validate function exists
	fn, exists := b.metadata.FunctionImports[functionName]
	if !exists {
		return nil, fmt.Errorf("function not found: %s", functionName)
	}

	// Check if function is filtered out
	if !b.shouldIncludeFunction(functionName) {
		return nil, fmt.Errorf("function not available: %s", functionName)
	}

	// Check read-only mode for modifying functions
	if b.config.ReadOnly && !b.config.AllowModifyingFunctions() && b.isFunctionModifying(fn) {
		return nil, fmt.Errorf("modifying function %s not allowed in read-only mode", functionName)
	}

	// Delegate to existing handler
	return b.handleFunctionCall(ctx, functionName, fn, params)
}

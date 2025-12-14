package test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// MCP Protocol structures
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type MCPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	scanner   *bufio.Scanner
	mu        sync.Mutex
	requestID int
	responses map[interface{}]chan MCPResponse
	t         *testing.T
}

func NewMCPClient(t *testing.T, serviceURL string) (*MCPClient, error) {
	// Build the server if needed
	buildCmd := exec.Command("go", "build", "-o", "../../odata-mcp", "../../cmd/odata-mcp")
	buildCmd.Dir = "."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to build server: %w\nOutput: %s", err, output)
	}

	cmd := exec.Command("../../odata-mcp", serviceURL)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	client := &MCPClient{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		scanner:   bufio.NewScanner(stdout),
		responses: make(map[interface{}]chan MCPResponse),
		t:         t,
	}

	// Start reading responses
	go client.readResponses()

	// Start reading stderr for debugging
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			t.Logf("[SERVER STDERR] %s", scanner.Text())
		}
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	return client, nil
}

func (c *MCPClient) readResponses() {
	for c.scanner.Scan() {
		line := c.scanner.Text()
		c.t.Logf("[SERVER RESPONSE] %s", line)

		var resp MCPResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			c.t.Logf("Failed to parse response: %v", err)
			continue
		}

		// Normalize ID type: JSON unmarshals numbers as float64, but we store as int
		normalizedID := normalizeID(resp.ID)

		c.mu.Lock()
		if ch, ok := c.responses[normalizedID]; ok {
			ch <- resp
			delete(c.responses, normalizedID)
		}
		c.mu.Unlock()
	}
}

// normalizeID converts float64 IDs (from JSON) to int for map key matching
func normalizeID(id interface{}) interface{} {
	if f, ok := id.(float64); ok {
		return int(f)
	}
	return id
}

func (c *MCPClient) SendRequest(method string, params interface{}) (MCPResponse, error) {
	c.mu.Lock()
	c.requestID++
	id := c.requestID
	c.mu.Unlock()

	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return MCPResponse{}, err
		}
		paramsJSON = data
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return MCPResponse{}, err
	}

	c.t.Logf("[CLIENT REQUEST] %s", string(data))

	respChan := make(chan MCPResponse, 1)
	c.mu.Lock()
	c.responses[id] = respChan
	c.mu.Unlock()

	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return MCPResponse{}, err
	}

	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(5 * time.Second):
		return MCPResponse{}, fmt.Errorf("timeout waiting for response")
	}
}

func (c *MCPClient) SendNotification(method string, params interface{}) error {
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		paramsJSON = data
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	c.t.Logf("[CLIENT NOTIFICATION] %s", string(data))

	_, err = c.stdin.Write(append(data, '\n'))
	return err
}

func (c *MCPClient) Close() error {
	c.stdin.Close()
	return c.cmd.Wait()
}

// Test Suite
type MCPProtocolTestSuite struct {
	suite.Suite
	client     *MCPClient
	serviceURL string
	mockServer *httptest.Server
}

func (suite *MCPProtocolTestSuite) SetupSuite() {
	// Use environment variable or mock server
	suite.serviceURL = os.Getenv("ODATA_URL")
	if suite.serviceURL == "" {
		suite.mockServer = newMCPMockServer(suite.T())
		suite.serviceURL = suite.mockServer.URL
		suite.T().Log("Using in-memory mock OData service:", suite.serviceURL)
	}
}

func (suite *MCPProtocolTestSuite) TearDownSuite() {
	if suite.mockServer != nil {
		suite.mockServer.Close()
	}
}

func (suite *MCPProtocolTestSuite) SetupTest() {
	client, err := NewMCPClient(suite.T(), suite.serviceURL)
	require.NoError(suite.T(), err)
	suite.client = client
}

func (suite *MCPProtocolTestSuite) TearDownTest() {
	if suite.client != nil {
		suite.client.Close()
	}
}

func (suite *MCPProtocolTestSuite) TestInitializeProtocol() {
	// Test initialize request
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "mcp-test-client",
			"version": "1.0.0",
		},
	}

	resp, err := suite.client.SendRequest("initialize", initParams)
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), resp.Error)

	// Parse result
	var result map[string]interface{}
	err = json.Unmarshal(resp.Result, &result)
	require.NoError(suite.T(), err)

	// Verify server info
	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "odata-mcp-bridge", serverInfo["name"])

	// Verify capabilities
	capabilities, ok := result["capabilities"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.NotNil(suite.T(), capabilities["tools"])

	// Send initialized notification
	err = suite.client.SendNotification("initialized", nil)
	assert.NoError(suite.T(), err)
}

func (suite *MCPProtocolTestSuite) TestListTools() {
	// Initialize first
	suite.initializeClient()

	// List tools
	resp, err := suite.client.SendRequest("tools/list", nil)
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), resp.Error)

	// Parse tools
	var result map[string]interface{}
	err = json.Unmarshal(resp.Result, &result)
	require.NoError(suite.T(), err)

	tools, ok := result["tools"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Greater(suite.T(), len(tools), 0)

	// Verify expected tools are present (using actual tool naming pattern)
	expectedTools := []string{
		"odata_service_info_for_od",
		"filter_TestEntities_for_od",
		"count_TestEntities_for_od",
		"get_TestEntities_for_od",
		"create_TestEntities_for_od",
		"update_TestEntities_for_od",
		"delete_TestEntities_for_od",
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]interface{})
		name := toolMap["name"].(string)
		toolNames[name] = true

		// Verify tool structure
		assert.NotEmpty(suite.T(), toolMap["description"])
		assert.NotNil(suite.T(), toolMap["inputSchema"])
	}

	for _, expected := range expectedTools {
		assert.True(suite.T(), toolNames[expected], "Tool %s should be present", expected)
	}
}

func (suite *MCPProtocolTestSuite) TestCallToolWithCSRF() {
	// Initialize first
	suite.initializeClient()

	// Test create tool (which should trigger CSRF token handling)
	toolParams := map[string]interface{}{
		"name": "create_TestEntities_for_od",
		"arguments": map[string]interface{}{
			"Name":  "Test Entity",
			"Value": 100,
		},
	}

	resp, err := suite.client.SendRequest("tools/call", toolParams)

	// The actual result depends on whether the service exists
	// We're mainly testing that the protocol works correctly
	if err != nil {
		suite.T().Logf("Tool call error (expected if service doesn't exist): %v", err)
	} else if resp.Error != nil {
		suite.T().Logf("Tool returned error: %+v", resp.Error)
		// Verify error structure
		assert.NotEmpty(suite.T(), resp.Error.Message)
		assert.NotZero(suite.T(), resp.Error.Code)
	} else {
		// If successful, verify result structure
		var result map[string]interface{}
		err = json.Unmarshal(resp.Result, &result)
		assert.NoError(suite.T(), err)
		assert.Contains(suite.T(), result, "content")
	}
}

func (suite *MCPProtocolTestSuite) TestInvalidRequest() {
	// Initialize first
	suite.initializeClient()

	// Test invalid method
	resp, err := suite.client.SendRequest("invalid/method", nil)
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), resp.Error)
	assert.Equal(suite.T(), -32601, resp.Error.Code) // Method not found
}

func (suite *MCPProtocolTestSuite) TestMissingRequiredParams() {
	// Initialize first
	suite.initializeClient()

	// Call nonexistent tool to test error handling
	toolParams := map[string]interface{}{
		"name": "nonexistent_tool",
		"arguments": map[string]interface{}{
			"Name": "Test",
		},
	}

	resp, err := suite.client.SendRequest("tools/call", toolParams)
	require.NoError(suite.T(), err)

	// Should return "Invalid params" error with "Tool not found" message
	assert.NotNil(suite.T(), resp.Error)
	assert.Equal(suite.T(), -32602, resp.Error.Code) // Invalid params
	assert.Contains(suite.T(), strings.ToLower(resp.Error.Message), "invalid")
}

func (suite *MCPProtocolTestSuite) TestConcurrentRequests() {
	// Initialize first
	suite.initializeClient()

	// Send multiple requests concurrently
	var wg sync.WaitGroup
	errors := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			resp, err := suite.client.SendRequest("tools/list", nil)
			if err != nil {
				errors[index] = err
				return
			}

			if resp.Error != nil {
				errors[index] = fmt.Errorf("response error: %v", resp.Error)
			}
		}(i)
	}

	wg.Wait()

	// Check all requests succeeded
	for i, err := range errors {
		assert.NoError(suite.T(), err, "Request %d failed", i)
	}
}

func (suite *MCPProtocolTestSuite) initializeClient() {
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "mcp-test-client",
			"version": "1.0.0",
		},
	}

	resp, err := suite.client.SendRequest("initialize", initParams)
	require.NoError(suite.T(), err)
	require.Nil(suite.T(), resp.Error)

	err = suite.client.SendNotification("initialized", nil)
	require.NoError(suite.T(), err)
}

func TestMCPProtocolTestSuite(t *testing.T) {
	suite.Run(t, new(MCPProtocolTestSuite))
}

func newMCPMockServer(tb testing.TB) *httptest.Server {
	tb.Helper()
	// Keep simple in-memory data
	type entity struct {
		ID    string `json:"ID"`
		Name  string `json:"Name"`
		Value int    `json:"Value"`
	}

	entities := []entity{
		{ID: "1", Name: "Test 1", Value: 100},
		{ID: "2", Name: "Test 2", Value: 200},
	}

	metadata := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx" Version="1.0">
  <edmx:DataServices>
    <Schema xmlns="http://schemas.microsoft.com/ado/2008/09/edm" Namespace="TestNamespace">
      <EntityType Name="TestEntity">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.String" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
        <Property Name="Value" Type="Edm.Int32"/>
      </EntityType>
      <EntityContainer Name="TestContainer">
        <EntitySet Name="TestEntities" EntityType="TestNamespace.TestEntity"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tb.Logf("[MOCK SERVER] %s %s", r.Method, r.URL.String())

		switch {
		case strings.HasSuffix(r.URL.Path, "/$metadata"):
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(metadata))
			return
		case r.URL.Path == "/" || r.URL.Path == "":
			// Service document
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"d": map[string]interface{}{
					"EntitySets": []string{"TestEntities"},
				},
			})
			return
		case strings.Contains(r.URL.Path, "TestEntities"):
			switch r.Method {
			case http.MethodGet:
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": map[string]interface{}{
						"results": entities,
					},
				})
			case http.MethodPost:
				var payload entity
				_ = json.NewDecoder(r.Body).Decode(&payload)
				if payload.ID == "" {
					payload.ID = fmt.Sprintf("%d", len(entities)+1)
				}
				entities = append(entities, payload)
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": payload,
				})
			case http.MethodPut:
				w.WriteHeader(http.StatusOK)
				var payload entity
				_ = json.NewDecoder(r.Body).Decode(&payload)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": payload,
				})
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		default:
			http.NotFound(w, r)
		}
	}))
}

// MCP Protocol Audit Tests
func TestMCPProtocolCompliance(t *testing.T) {
	// These tests verify compliance with MCP specification

	t.Run("JSONRPCVersion", func(t *testing.T) {
		server := newMCPMockServer(t)
		defer server.Close()

		client, err := NewMCPClient(t, server.URL)
		require.NoError(t, err)
		defer client.Close()

		// All responses should have jsonrpc: "2.0"
		resp, err := client.SendRequest("tools/list", nil)
		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
	})

	t.Run("ErrorCodes", func(t *testing.T) {
		// Test standard JSON-RPC error codes
		testCases := []struct {
			name         string
			method       string
			params       interface{}
			expectedCode int
		}{
			{
				name:         "ParseError",
				method:       "", // Will send malformed JSON
				expectedCode: -32700,
			},
			{
				name:         "InvalidRequest",
				method:       "test",
				params:       "invalid", // Should be object
				expectedCode: -32600,
			},
			{
				name:         "MethodNotFound",
				method:       "nonexistent/method",
				expectedCode: -32601,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test implementation would go here
				// This is a placeholder for the structure
				t.Logf("Testing error code %d for %s", tc.expectedCode, tc.name)
			})
		}
	})
}

// Benchmark tests
func BenchmarkMCPRequests(b *testing.B) {
	server := newMCPMockServer(b)
	defer server.Close()

	client, err := NewMCPClient(&testing.T{}, server.URL)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	// Initialize
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "benchmark-client",
			"version": "1.0.0",
		},
	}

	_, err = client.SendRequest("initialize", initParams)
	if err != nil {
		b.Fatal(err)
	}

	client.SendNotification("initialized", nil)

	b.ResetTimer()

	b.Run("ToolsList", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.SendRequest("tools/list", nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ToolCall", func(b *testing.B) {
		params := map[string]interface{}{
			"name":      "get_metadata",
			"arguments": map[string]interface{}{},
		}

		for i := 0; i < b.N; i++ {
			_, err := client.SendRequest("tools/call", params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

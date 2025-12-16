package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zmcp/odata-mcp/internal/transport"
)

// SSETransport implements the Transport interface for Server-Sent Events
type SSETransport struct {
	addr            string
	server          *http.Server
	handler         transport.Handler
	clients         map[string]*sseClient
	mu              sync.RWMutex
	messages        chan *clientMessage
	droppedMessages int64 // Atomic counter for dropped messages
	verbose         bool  // Enable verbose logging for dropped messages
}

type sseClient struct {
	id      string
	events  chan []byte
	done    chan struct{}
	writer  http.ResponseWriter
	flusher http.Flusher
}

type clientMessage struct {
	clientID string
	message  *transport.Message
}

// NewSSE creates a new SSE transport
func NewSSE(addr string, handler transport.Handler) *SSETransport {
	return &SSETransport{
		addr:     addr,
		handler:  handler,
		clients:  make(map[string]*sseClient),
		messages: make(chan *clientMessage, 100),
	}
}

// SetVerbose enables verbose logging for dropped messages
func (t *SSETransport) SetVerbose(verbose bool) {
	t.verbose = verbose
}

// GetDroppedMessageCount returns the number of dropped messages
func (t *SSETransport) GetDroppedMessageCount() int64 {
	return atomic.LoadInt64(&t.droppedMessages)
}

// Start initializes the HTTP server and begins listening
func (t *SSETransport) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// SSE endpoint for bidirectional communication
	mux.HandleFunc("/sse", t.handleSSE)

	// Regular HTTP endpoint for request-response
	mux.HandleFunc("/rpc", t.handleRPC)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	t.server = &http.Server{
		Addr:    t.addr,
		Handler: mux,
	}

	// Start message processor
	go t.processMessages(ctx)

	// Start server
	go func() {
		if err := t.server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	<-ctx.Done()
	return t.Close()
}

// handleSSE handles SSE connections
func (t *SSETransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Check if the request accepts SSE (allow combined Accept headers like "text/event-stream, application/json")
	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		http.Error(w, "SSE not supported", http.StatusBadRequest)
		return
	}

	// Ensure we can flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client
	client := &sseClient{
		id:      fmt.Sprintf("client-%d", time.Now().UnixNano()),
		events:  make(chan []byte, 10),
		done:    make(chan struct{}),
		writer:  w,
		flusher: flusher,
	}

	// Register client
	t.mu.Lock()
	t.clients[client.id] = client
	t.mu.Unlock()

	// Send connection event
	t.sendEvent(client, "connected", map[string]string{"clientId": client.id})

	// Clean up on disconnect
	defer func() {
		t.mu.Lock()
		delete(t.clients, client.id)
		t.mu.Unlock()
		close(client.events)
		close(client.done)
	}()

	// Handle incoming messages from query parameters or POST body
	if r.Method == http.MethodPost {
		var msg transport.Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err == nil {
			t.messages <- &clientMessage{
				clientID: client.id,
				message:  &msg,
			}
		}
	}

	// Send events to client
	for {
		select {
		case event := <-client.events:
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
		case <-client.done:
			return
		case <-r.Context().Done():
			return
		}
	}
}

// handleRPC handles regular HTTP RPC requests
func (t *SSETransport) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg transport.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process the message
	ctx := r.Context()
	response, err := t.handler(ctx, &msg)
	if err != nil {
		response = &transport.Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &transport.Error{
				Code:    -32603,
				Message: err.Error(),
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// processMessages handles incoming messages from clients
func (t *SSETransport) processMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cm := <-t.messages:
			if cm.message.Method != "" && t.handler != nil {
				response, err := t.handler(ctx, cm.message)
				if err != nil {
					response = &transport.Message{
						JSONRPC: "2.0",
						ID:      cm.message.ID,
						Error: &transport.Error{
							Code:    -32603,
							Message: err.Error(),
						},
					}
				}

				// Send response to specific client
				t.mu.RLock()
				client, exists := t.clients[cm.clientID]
				t.mu.RUnlock()

				if exists && response != nil {
					data, _ := json.Marshal(response)
					select {
					case client.events <- data:
					default:
						// Client buffer full - log and count dropped message
						dropped := atomic.AddInt64(&t.droppedMessages, 1)
						if t.verbose {
							fmt.Fprintf(os.Stderr, "[SSE] Dropped response for client %s: buffer full (total dropped: %d)\n",
								cm.clientID, dropped)
						}
					}
				}
			}
		}
	}
}

// sendEvent sends an event to a specific client
func (t *SSETransport) sendEvent(client *sseClient, eventType string, data interface{}) {
	event := map[string]interface{}{
		"type": eventType,
		"data": data,
	}

	if eventData, err := json.Marshal(event); err == nil {
		select {
		case client.events <- eventData:
		default:
			// Buffer full - log and count dropped event
			dropped := atomic.AddInt64(&t.droppedMessages, 1)
			if t.verbose {
				fmt.Fprintf(os.Stderr, "[SSE] Dropped %s event for client %s: buffer full (total dropped: %d)\n",
					eventType, client.id, dropped)
			}
		}
	}
}

// BroadcastMessage sends a message to all connected clients
func (t *SSETransport) BroadcastMessage(msg *transport.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	for clientID, client := range t.clients {
		select {
		case client.events <- data:
		default:
			// Client buffer full - log and count dropped broadcast
			dropped := atomic.AddInt64(&t.droppedMessages, 1)
			if t.verbose {
				fmt.Fprintf(os.Stderr, "[SSE] Dropped broadcast for client %s: buffer full (total dropped: %d)\n",
					clientID, dropped)
			}
		}
	}

	return nil
}

// ReadMessage is not used for HTTP/SSE transport
func (t *SSETransport) ReadMessage() (*transport.Message, error) {
	return nil, fmt.Errorf("ReadMessage not implemented for HTTP/SSE transport")
}

// WriteMessage broadcasts a message to all connected clients
func (t *SSETransport) WriteMessage(msg *transport.Message) error {
	return t.BroadcastMessage(msg)
}

// Close gracefully shuts down the HTTP server
func (t *SSETransport) Close() error {
	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return t.server.Shutdown(ctx)
	}
	return nil
}

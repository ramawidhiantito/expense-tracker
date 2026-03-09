package web

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"expense-tracking/internal/core/domain"
	"expense-tracking/internal/core/ports"
	"expense-tracking/internal/infrastructure/web/templates"
)

type Server struct {
	stateManager ports.StateManager
	clients      map[chan string]bool
	mu           sync.Mutex
	port         string
}

func NewServer(stateManager ports.StateManager, port string) *Server {
	return &Server{
		stateManager: stateManager,
		clients:      make(map[chan string]bool),
		port:         port,
	}
}

// Start HTTP server
func (s *Server) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/events", s.handleEvents)

	log.Printf("Starting web server on %s", s.port)
	if err := http.ListenAndServe(s.port, mux); err != nil {
		log.Fatalf("Web server failed: %v", err)
	}
}

// Broadcasts a new expense to all connected clients via SSE
func (s *Server) BroadcastNewExpense(expense domain.Expense) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Render the single row using templ
	var buf bytes.Buffer
	err := templates.ExpenseRow(expense).Render(context.Background(), &buf)
	if err != nil {
		log.Printf("Failed to render templ row: %v", err)
		return
	}

	// For SSE, all lines in data payload must be prefixed with 'data: '
	htmlLines := bytes.Split(buf.Bytes(), []byte("\n"))
	var ssePayload bytes.Buffer
	ssePayload.WriteString("event: expense\n")
	for _, line := range htmlLines {
		ssePayload.WriteString("data: ")
		ssePayload.Write(line)
		ssePayload.WriteString("\n")
	}
	ssePayload.WriteString("\n")
	payload := ssePayload.String()

	// Send to all clients
	for clientChan := range s.clients {
		select {
		case clientChan <- payload:
		default:
			// If channel blocks, drop client
			delete(s.clients, clientChan)
			close(clientChan)
		}
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	expenses, err := s.stateManager.ListExpenses(context.Background(), 50)
	if err != nil {
		log.Printf("Error fetching initial expenses: %v", err)
		http.Error(w, "Failed to load expenses", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = templates.Index(expenses).Render(context.Background(), w)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// Set Headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Ensure flush capability
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create and register new channel
	messageChan := make(chan string, 10)

	s.mu.Lock()
	s.clients[messageChan] = true
	s.mu.Unlock()

	// Clean up client unregister logic
	defer func() {
		s.mu.Lock()
		delete(s.clients, messageChan)
		s.mu.Unlock()
		close(messageChan)
	}()

	// Send an initial heartbeat/connection success
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	// Keep connection open and push messages
	for {
		select {
		case <-r.Context().Done(): // Client disconnected
			return
		case msg := <-messageChan:
			// Push new row event
			// For htmx SSE: The entire msg including "event:" and "data:" has been formatted in BroadcastNewExpense
			// Note: Wait, BroadcastNewExpense contains newlines. We need to be careful with SSE padding.
			// The htmx ext waits for `event` and `data`.
			fmt.Fprint(w, msg)
			flusher.Flush()
		case <-time.After(30 * time.Second):
			// Keep-alive heartbeat
			fmt.Fprintf(w, "event: ping\ndata: {}\n\n")
			flusher.Flush()
		}
	}
}

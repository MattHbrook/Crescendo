package websocket

import (
	"crescendo/types"
	"log"
	"sync"
	"time"
)

// Hub interface defines the methods for managing WebSocket connections
type Hub interface {
	Run()
	BroadcastProgress(jobID, msgType, status, currentFile, speed, message string, progress float64)
	RegisterClient(client *Client)
	UnregisterClient(client *Client)
}

// hub maintains the set of active clients and broadcasts messages to them
type hub struct {
	// Registered clients mapped by job ID
	clients map[string]map[*Client]bool

	// Broadcast channel for sending messages to all clients of a job
	broadcast chan types.ProgressMessage

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() Hub {
	return &hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan types.ProgressMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main event loop
func (h *hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.jobID] == nil {
				h.clients[client.jobID] = make(map[*Client]bool)
			}
			h.clients[client.jobID][client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected for job %s", client.jobID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.jobID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.jobID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected for job %s", client.jobID)

		case message := <-h.broadcast:
			h.mu.RLock()
			// Send to specific job clients
			if clients, ok := h.clients[message.JobID]; ok {
				for client := range clients {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(clients, client)
					}
				}
				if len(clients) == 0 {
					delete(h.clients, message.JobID)
				}
			}

			// Also send to "all" clients for any job update
			if allClients, ok := h.clients["all"]; ok {
				for client := range allClients {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(allClients, client)
					}
				}
				if len(allClients) == 0 {
					delete(h.clients, "all")
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastProgress sends a progress message to all clients of a specific job
func (h *hub) BroadcastProgress(jobID, msgType, status, currentFile, speed, message string, progress float64) {
	progressMsg := types.ProgressMessage{
		JobID:       jobID,
		Type:        msgType,
		Progress:    progress,
		Status:      status,
		CurrentFile: currentFile,
		Speed:       speed,
		Message:     message,
		Timestamp:   time.Now(),
	}

	select {
	case h.broadcast <- progressMsg:
	default:
		log.Printf("WebSocket broadcast channel full, dropping message for job %s", jobID)
	}
}

// RegisterClient registers a new client with the hub
func (h *hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client from the hub
func (h *hub) UnregisterClient(client *Client) {
	h.unregister <- client
}
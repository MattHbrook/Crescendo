package websocket

import (
	"crescendo/types"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader with CORS support
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for development
		// In production, check against allowed origins
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	hub   Hub
	conn  *websocket.Conn
	send  chan types.ProgressMessage
	jobID string
}

// NewClient creates a new WebSocket client
func NewClient(hub Hub, conn *websocket.Conn, jobID string) *Client {
	return &Client{
		hub:   hub,
		conn:  conn,
		send:  make(chan types.ProgressMessage, 256),
		jobID: jobID,
	}
}

// StartPumps starts the read and write pumps for the client
func (c *Client) StartPumps() {
	go c.writePump()
	go c.readPump()
}

// readPump handles reading from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.UnregisterClient(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// writePump handles writing to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GetUpgrader returns the WebSocket upgrader
func GetUpgrader() websocket.Upgrader {
	return upgrader
}
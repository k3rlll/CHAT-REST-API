package ws

import (
	"log/slog"
	"net/http"
	"sync"

	"main/pkg/metrics"

	"github.com/gorilla/websocket"
)

type Manager struct {
	logger   *slog.Logger
	mu       sync.RWMutex
	clients  map[int64]*websocket.Conn
	upgrader websocket.Upgrader
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		logger:  logger,
		clients: make(map[int64]*websocket.Conn),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

func (m *Manager) HandleConnection(w http.ResponseWriter, r *http.Request, userID int64) (*websocket.Conn, error) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("failed to upgrade connection", "error", err)
		return nil, err
	}

	m.mu.Lock()
	m.clients[userID] = conn
	m.mu.Unlock()

	m.logger.Info("User connected via WS", "userID", userID)

	defer func() {
		if err := conn.Close(); err != nil {
			m.logger.Error("failed to close connection", "error", err)
		}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			m.logger.Error("failed to read message", "error", err)
			break
		}
	}

	return conn, nil
}

func (m *Manager) WsUnicast(userID int64, data interface{}) {
	m.mu.RLock()
	conn, ok := m.clients[userID]
	m.mu.RUnlock()
	if !ok {
		return
	}
	if err := conn.WriteJSON(data); err != nil {
		m.logger.Error("failed to write JSON message", "userID", userID, "error", err)
		m.removeClient(userID)
		conn.Close()
	}
}

func (m *Manager) AddClient(userID int64, conn *websocket.Conn) {
	metrics.ActiveWebSocketConnections.Inc()
	m.mu.Lock()
	m.clients[userID] = conn
	m.mu.Unlock()
}

func (m *Manager) removeClient(userID int64) {
	metrics.ActiveWebSocketConnections.Dec()
	m.mu.Lock()
	delete(m.clients, userID)
	m.mu.Unlock()
}

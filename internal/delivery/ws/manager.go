package ws

import (
	"context"
	"fmt"
	pb "main/internal/delivery/ws/events_proto/websocket/v1"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type Manager struct {
	Clients map[*Client]bool
	sync.RWMutex
	register   chan *Client
	unregister chan *Client
	broadcast  chan *pb.WebSocketEvent
	rooms      map[string]map[*Client]bool
	Ctx        context.Context
	rdb        *redis.Client
}

func NewManager(ctx context.Context, rdb *redis.Client) *Manager {
	return &Manager{
		Clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *pb.WebSocketEvent),
		rooms:      make(map[string]map[*Client]bool),
		Ctx:        ctx,
		rdb:        rdb,
	}
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request, userID string) {
	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		fmt.Println("roomID is required")
		http.Error(w, "roomID is required", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	conn.SetReadLimit(1024 * 1024) // 1 MB
	if err != nil {
		fmt.Println("WebSocket Accept Error:", err)
		return
	}

	client := NewClient(m, conn, roomID, userID)
	m.register <- client

	go client.Heartbeat(r.Context())

	go client.WriteMessages(r.Context())
	client.ReadMessages(r.Context())

}
func (m *Manager) Run() {
	for {
		select {
		case client := <-m.register:
			if _, ok := m.rooms[client.roomID]; !ok {
				m.rooms[client.roomID] = make(map[*Client]bool)
			}

			m.rooms[client.roomID][client] = true
		case client := <-m.unregister:
			if clientsInRoom, ok := m.rooms[client.roomID]; ok {
				if _, ok := clientsInRoom[client]; ok {
					delete(clientsInRoom, client)
					close(client.send)
					if len(clientsInRoom) == 0 {
						delete(m.rooms, client.roomID)
					}
				}
			}

		case event := <-m.broadcast:
			if clientsInRoom, ok := m.rooms[event.RoomId]; ok {
				for client := range clientsInRoom {
					if event.SenderId == client.userID {
						continue
					}
					select {
					case client.send <- event.Data:
					default:
						close(client.send)
						delete(clientsInRoom, client)
					}
				}
			}
		}
	}
}

func (m *Manager) ListenRedis() {
	subscriber := m.rdb.Subscribe(m.Ctx, "chat_events")
	defer subscriber.Close()
	ch := subscriber.Channel(
		redis.WithChannelHealthCheckInterval(30*time.Second),
		redis.WithChannelSize(1000),
	)
	for msg := range ch {
		event := &pb.WebSocketEvent{}
		err := proto.Unmarshal([]byte(msg.Payload), event)
		if err != nil {
			fmt.Println("Protobuf unmarshal error:", err)
			continue
		}

		m.broadcast <- event
	}
}

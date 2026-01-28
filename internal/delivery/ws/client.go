package ws

import (
	"context"
	"io"
	"log"
	"time"

	pb "main/internal/delivery/ws/events_proto/websocket/v1"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"
)

var client map[*Client]bool

type Client struct {
	manager *Manager
	conn    *websocket.Conn
	send    chan []byte
	roomID  string
	userID  string
}

func NewClient(manager *Manager, conn *websocket.Conn, roomID string, userID string) *Client {
	return &Client{
		manager: manager,
		conn:    conn,
		send:    make(chan []byte, 256),
		roomID:  roomID,
		userID:  userID,
	}
}

func (c *Client) ReadMessages(ctx context.Context) {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close(websocket.StatusNormalClosure, "the read message failed")
	}()
	for {
		_, reader, err := c.conn.Reader(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway ||
				websocket.CloseStatus(err) == websocket.StatusNoStatusRcvd || websocket.CloseStatus(err) == websocket.StatusAbnormalClosure {
				return
			}
			log.Println("error reading message:", err)
			break
		}
		b, err := io.ReadAll(reader)
		if err != nil {
			log.Println("read error:", err)
			continue
		}

		event := &pb.WebSocketEvent{
			SenderId:       c.userID,
			Data:           b,
			RoomId:         c.roomID,
			EventType:      "message"}

		data, err := proto.Marshal(event)
		if err != nil {
			log.Printf("Protobuf marshal error: %v", err)
			continue
		}
		err = c.manager.rdb.Publish(c.manager.ctx, "chat_events", data).Err()
		if err != nil {
			log.Printf("Redis publish error: %v", err)
		}

		log.Printf("RAW payload: '%s'", string(b))

	}
}

func (c *Client) WriteMessages(ctx context.Context) error {
	defer c.conn.Close(websocket.StatusNormalClosure, "the write message failed")
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return c.conn.Close(websocket.StatusNormalClosure, "the send channel was closed")
			}
			writeCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			err := c.conn.Write(writeCtx, websocket.MessageText, message)
			cancel()
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *Client) Heartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close(websocket.StatusNormalClosure, "heartbeat stopped")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				log.Println("ping error:", err)
				return
			}
		}
	}
}

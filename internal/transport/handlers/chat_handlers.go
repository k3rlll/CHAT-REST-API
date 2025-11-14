package handlers

import (
	"encoding/json"
	"log/slog"
	srvAuth "main/internal/service/auth"
	srvChat "main/internal/service/chat"
	srvMessage "main/internal/service/message"
	srvUser "main/internal/service/user"
	"net/http"
)

type ChatHandler struct {
	UserSrv *srvUser.UserService
	AuthSrv *srvAuth.AuthService
	MessSrv *srvMessage.MessageService
	ChatSrv *srvChat.ChatService
	logger  *slog.Logger
}

func NewChatHandler(userSrv *srvUser.UserService, authSrv *srvAuth.AuthService, messSrv *srvMessage.MessageService, chatSrv *srvChat.ChatService, logger *slog.Logger) *ChatHandler {
	return &ChatHandler{
		UserSrv: userSrv,
		AuthSrv: authSrv,
		MessSrv: messSrv,
		ChatSrv: chatSrv,
		logger:  logger,
	}
}

/*pattern: /v1/chats
method:  POST
info:    Создать чат: direct (user_id) или group (title)

succeed:
  - status code: 201 Created
  - response body: JSON { chat:{ id, type, title?, members:[...] } }

failed:
  - status code: 400 Bad Request
  - status code: 401 Unauthorized
  - status code: 404 Not Found (для direct: user не найден)
  - status code: 409 Conflict (direct уже существует)
  - status code: 422 Unprocessable Entity
  - status code: 500 Internal Server Error
  - response body: JSON error + time
*/

func (h *ChatHandler) CreateChatHandler(w http.ResponseWriter, r *http.Request) {

	var members []int

	if err := json.NewDecoder(r.Body).Decode(&members); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := h.ChatSrv.CreateChat(r.Context(), false, "title", members)
	if err != nil {
		h.logger.Error("failed to create chat", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// pattern: /v1/chats
// method:  GET
// info:    Список чатов пользователя; параметры: cursor, limit, q

// succeed:
//   - status code: 200 OK
//   - response body: JSON { items:[{ chat, last_message, pinned, muted }], next_cursor }

// failed:
//   - status code: 400 Bad Request
//   - status code: 401 Unauthorized
//   - status code: 429 Too Many Requests
//   - status code: 500 Internal Server Error
//   - response body: JSON error + time

func (h *ChatHandler) getChatsHandler(w http.ResponseWriter, r *http.Request) {
	chats, err := h.ChatSrv.ListOfChats(r.Context())
	if err != nil {
		h.logger.Error("failed to get list of chats", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b, err := json.MarshalIndent(chats, "", "  ")
	if err != nil {
		h.logger.Error("failed to marshal response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// pattern: /v1/chats/{id}
// method:  GET
// info:    Детали чата (если участник)

// succeed:
//   - status code: 200 OK
//   - response body: JSON { chat:{...}, members:[...], settings:{ pinned, muted } }

// failed:
//   - status code: 401 Unauthorized
//   - status code: 403 Forbidden (не участник)
//   - status code: 404 Not Found
//   - status code: 500 Internal Server Error
//   - response body: JSON error + time

func viewChatHandler(w http.ResponseWriter, r *http.Request) {}

// pattern: /v1/chats/{id}
// method:  DELETE
// info:    Удалить чат/Архивировать

// succeed:
//   - status code: 204 No Content
//   - response body: пусто

// failed:
//   - status code: 401 Unauthorized
//   - status code: 403 Forbidden (нельзя покинуть как единственный owner)
//   - status code: 404 Not Found
//   - status code: 409 Conflict (состояние не позволяет)
//   - status code: 500 Internal Server Error
//   - response body: JSON error + time
func (h *ChatHandler) deleteChatHandler(w http.ResponseWriter, r *http.Request) {

	var chat_id int

	if err := json.NewDecoder(r.Body).Decode(&chat_id); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.ChatSrv.DeleteChat(r.Context(), chat_id); err != nil {
		h.logger.Error("failed to delete chat", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

// pattern: /v1/chats/{id}/members
// method:  GET
// info:    Список участников чата

// succeed:
//   - status code: 200 OK
//   - response body: JSON { items:[{ user, role }], total }

// failed:
//   - status code: 401 Unauthorized
//   - status code: 403 Forbidden
//   - status code: 404 Not Found
//   - status code: 500 Internal Server Error
//   - response body: JSON error + time

func getListMembersHandler(w http.ResponseWriter, r *http.Request) {}

// pattern: /v1/chats/{id}/members
// method:  POST
// info:    Добавить участника(ов) в групповой чат

// succeed:
//   - status code: 201 Created
//   - response body: JSON { added:[user_id...], chat_id }

// failed:
//   - status code: 400 Bad Request
//   - status code: 401 Unauthorized
//   - status code: 403 Forbidden (нет прав)
//   - status code: 404 Not Found (chat/user)
//   - status code: 409 Conflict (уже участник)
//   - status code: 422 Unprocessable Entity
//   - status code: 500 Internal Server Error
//   - response body: JSON error + time

func addMembersHandler(w http.ResponseWriter, r *http.Request) {}

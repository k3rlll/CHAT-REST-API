package handlers

import "net/http"

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

func CreateChatHandler(w http.ResponseWriter, r *http.Request)

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

func getChatsHandler(w http.ResponseWriter, r *http.Request) {}

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
func deleteChatHandler(w http.ResponseWriter, r *http.Request) {}

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

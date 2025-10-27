package handlers

import "net/http"

/*pattern: /v1/me
method:  GET
info:    Получить профиль текущего пользователя

succeed:
  - status code: 200 OK
  - response body: JSON { id, username, email?, avatar, ... }

failed:
  - status code: 401 Unauthorized
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func meHandler(w http.ResponseWriter, r *http.Request) {}

/*pattern: /v1/users/{id}
method:  GET
info:    Публичный профиль пользователя по ID

succeed:
  - status code: 200 OK
  - response body: JSON { id, username, avatar, status, ... }

failed:
  - status code: 400 Bad Request
  - status code: 403 Forbidden (вы заблокированы друг у друга — по политике)
  - status code: 404 Not Found
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func userByIDHandler(w http.ResponseWriter, r *http.Request) {}

/*pattern: /v1/users
method:  GET
info:    Поиск пользователей; параметры: q, limit, cursor

succeed:
  - status code: 200 OK
  - response body: JSON { items:[...], next_cursor }

failed:
  - status code: 400 Bad Request (параметры)
  - status code: 401 Unauthorized
  - status code: 429 Too Many Requests
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/
func usersSearchHandler(w http.ResponseWriter, r *http.Request) {}

/*pattern: /v1/users/{id}/block
method:  POST
info:    Заблокировать пользователя

succeed:
  - status code: 201 Created
  - response body: JSON { user_id, blocked:true, blocked_at }

failed:
  - status code: 400 Bad Request
  - status code: 401 Unauthorized
  - status code: 404 Not Found (user)
  - status code: 409 Conflict (уже заблокирован)
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func userBlockHandler(w http.ResponseWriter, r *http.Request) {}

/*pattern: /v1/users/{id}/block
method:  DELETE
info:    Разблокировать пользователя

succeed:
  - status code: 204 No Content
  - response body: пусто

failed:
  - status code: 401 Unauthorized
  - status code: 404 Not Found (не был заблокирован)
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func userUnblockHandler(w http.ResponseWriter, r *http.Request) {}

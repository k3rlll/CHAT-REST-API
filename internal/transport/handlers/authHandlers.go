package handlers

import "net/http"

/*pattern: /v1/auth/register
method:  POST
info:    Регистрация нового пользователя

succeed:
  - status code: 201 Created
  - response body: JSON { user:{...}, access_token, refresh_token, issued_at }

failed:
  - status code: 400 Bad Request (невалидные поля)
  - status code: 409 Conflict (username/email заняты)
  - status code: 422 Unprocessable Entity (валидация)
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func registerHandler(w http.ResponseWriter, r *http.Request) {}

/*pattern: /v1/auth/login
method:  POST
info:    Логин по паролю, выдача токенов

succeed:
  - status code: 200 OK
  - response body: JSON { user:{...}, access_token, refresh_token, issued_at }

failed:
  - status code: 400 Bad Request
  - status code: 401 Unauthorized (неверные креды)
  - status code: 429 Too Many Requests
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func loginHandler(w http.ResponseWriter, r *http.Request) {}

/*pattern: /v1/auth/token/refresh
method:  POST
info:    Обновление пары токенов по refresh

succeed:
  - status code: 200 OK
  - response body: JSON { access_token, refresh_token, issued_at }

failed:
  - status code: 400 Bad Request
  - status code: 401 Unauthorized (просрочен/заблокирован/невалиден refresh)
  - status code: 429 Too Many Requests
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func refreshTokenHandler(w http.ResponseWriter, r *http.Request) {}

/*pattern: /v1/auth/logout
method:  POST
info:    Выход с текущей сессии (инвалидация refresh)

succeed:
  - status code: 204 No Content
  - response body: пусто

failed:
  - status code: 401 Unauthorized
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/
func logoutHandler(w http.ResponseWriter, r *http.Request) {}

/*pattern: /v1/auth/logout-all
method:  POST
info:    Выход со всех устройств/сессий

succeed:
  - status code: 204 No Content
  - response body: пусто

failed:
  - status code: 401 Unauthorized
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/
func logoutAllHandler(w http.ResponseWriter, r *http.Request) {}

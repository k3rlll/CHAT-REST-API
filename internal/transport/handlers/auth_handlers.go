package handlers

import (
	"encoding/json"
	db "main/internal/database"
	dto "main/internal/pkg/DTO"
	"net/http"
	"time"

	"github.com/go-playground/validator"
)

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

// type User struct {
// 	ID       string `json:"id"`
// 	Username string `json:"username"`
// 	Email    string `json:"email"`
// }

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type RegisterResponse struct {
	User         db.User `json:"user"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errDTO := dto.ErrDTO{
			Error: err.Error(),
			Time:  time.Now().Format(time.RFC3339),
		}
		http.Error(w, errDTO.Error, http.StatusBadRequest)
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		errDTO := dto.ErrDTO{
			Error: err.Error(),
			Time:  time.Now().Format(time.RFC3339),
		}
		http.Error(w, errDTO.Error, http.StatusBadRequest)
		return
	}

}

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

/*
pattern: /v1/auth/logout
method:  POST
info:    Выход с текущей сессии (инвалидация refresh)

succeed:
  - status code: 204 No Content
  - response body: пусто

failed:
  - status code: 401 Unauthorized
  - status code: 500 Internal Server Error
  - response body: JSON error + time
*/
func logoutHandler(w http.ResponseWriter, r *http.Request) {}

/*
pattern: /v1/auth/logout-all
method:  POST
info:    Выход со всех устройств/сессий

succeed:
  - status code: 204 No Content
  - response body: пусто

failed:
  - status code: 401 Unauthorized
  - status code: 500 Internal Server Error
  - response body: JSON error + time
*/
func logoutAllHandler(w http.ResponseWriter, r *http.Request) {}

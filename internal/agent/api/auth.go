// В этом файле описаны методы клиента для работы
// с эндпоинтами аутентификации: регистрация, вход, обновление токена и получение
// информации о текущем пользователе.
package api

// RegisterRequest описывает тело запроса регистрации пользователя.
//
// Email и Password передаются в JSON формате в эндпоинт /auth/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterResponse описывает ответ сервера при успешной регистрации.
//
// UserID содержит идентификатор созданного пользователя.
type RegisterResponse struct {
	UserID string `json:"user_id"`
}

// LoginRequest описывает тело запроса входа пользователя.
//
// Email и Password передаются в JSON формате в эндпоинт /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse описывает ответ сервера при успешном входе.
//
// AccessToken используется для авторизации запросов к защищённым эндпоинтам.
// RefreshToken используется для обновления пары токенов через /auth/refresh.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshRequest описывает тело запроса обновления токенов.
//
// RefreshToken передаётся в JSON формате в эндпоинт /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse описывает ответ сервера при успешном обновлении токенов.
//
// Возвращается новая пара AccessToken/RefreshToken.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Register выполняет регистрацию пользователя на сервере.
//
// Метод отправляет POST запрос на /auth/register и возвращает RegisterResponse.
// В случае ошибки возвращает непустую ошибку и пустой ответ.
func (c *Client) Register(email, password string) (RegisterResponse, error) {
	var resp RegisterResponse
	err := c.PostJSON("/auth/register", RegisterRequest{Email: email, Password: password}, &resp, "")
	return resp, err
}

// Login выполняет вход пользователя и получает пару токенов.
//
// Метод отправляет POST запрос на /auth/login и возвращает LoginResponse
// с AccessToken и RefreshToken. В случае ошибки возвращает непустую ошибку
// и пустой ответ.
func (c *Client) Login(email, password string) (LoginResponse, error) {
	var resp LoginResponse
	err := c.PostJSON("/auth/login", LoginRequest{Email: email, Password: password}, &resp, "")
	return resp, err
}

// Refresh обновляет пару токенов по refresh токену.
//
// Метод отправляет POST запрос на /auth/refresh и возвращает новую пару токенов.
// В случае ошибки возвращает непустую ошибку и пустой ответ.
func (c *Client) Refresh(refreshToken string) (RefreshResponse, error) {
	var resp RefreshResponse
	err := c.PostJSON("/auth/refresh", RefreshRequest{RefreshToken: refreshToken}, &resp, "")
	return resp, err
}

// MeResponse описывает ответ сервера с информацией о текущем пользователе.
//
// UserID содержит идентификатор пользователя, ассоциированного с переданным access токеном.
type MeResponse struct {
	UserID string `json:"user_id"`
}

// Me запрашивает информацию о текущем пользователе.
//
// Метод отправляет GET запрос на /me и использует accessToken для авторизации.
// В случае ошибки возвращает непустую ошибку и пустой ответ.
func (c *Client) Me(accessToken string) (MeResponse, error) {
	var resp MeResponse
	err := c.GetJSON("/me", &resp, accessToken)
	return resp, err
}

package sources

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hakaton/subscriptions-backend/internal/config"
)

// OAuthService обрабатывает OAuth авторизацию для почтовых провайдеров.
type OAuthService struct {
	cfg  *config.Config
	repo *Repository
}

// NewOAuthService создаёт сервис OAuth.
func NewOAuthService(cfg *config.Config, repo *Repository) *OAuthService {
	return &OAuthService{
		cfg:  cfg,
		repo: repo,
	}
}

// OAuthState хранит временное состояние для OAuth flow.
type OAuthState struct {
	UserID   string
	Provider string
	ExpiresAt time.Time
}

// generateState создаёт state с userID для OAuth.
func generateState(userID string) (string, error) {
	// Кодируем userID в state: base64(userID + separator + random)
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	
	data := userID + "|" + base64.URLEncoding.EncodeToString(random)
	return base64.URLEncoding.EncodeToString([]byte(data)), nil
}

// ParseState извлекает userID из state.
func (s *OAuthService) ParseState(state string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return "", err
	}
	
	parts := string(decoded)
	// Формат: "userID|random"
	for i := 0; i < len(parts); i++ {
		if parts[i] == '|' {
			return parts[:i], nil
		}
	}
	
	return "", fmt.Errorf("invalid state format")
}

// GetAuthorizationURL возвращает URL для редиректа на страницу авторизации провайдера.
func (s *OAuthService) GetAuthorizationURL(provider string, userID string, redirectURI string) (string, error) {
	state, err := generateState(userID)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// TODO: сохранить state в БД или сессию для проверки в callback
	// Для MVP можно использовать JWT или просто проверить формат state

	var authURL string
	var clientID string
	var scope string

	switch provider {
	case "mailru":
		clientID = s.cfg.MailruClientID
		scope = "userinfo mail.imap"
		authURL = fmt.Sprintf(
			"https://oauth.mail.ru/login?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&state=%s",
			url.QueryEscape(clientID),
			url.QueryEscape(redirectURI),
			url.QueryEscape(scope),
			url.QueryEscape(state),
		)
	case "yandex":
		clientID = s.cfg.YandexClientID
		scope = "mail.imap"
		authURL = fmt.Sprintf(
			"https://oauth.yandex.ru/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s",
			url.QueryEscape(clientID),
			url.QueryEscape(redirectURI),
			url.QueryEscape(scope),
			url.QueryEscape(state),
		)
	case "gmail":
		clientID = s.cfg.GmailClientID
		scope = "https://www.googleapis.com/auth/gmail.readonly"
		authURL = fmt.Sprintf(
			"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent&state=%s",
			url.QueryEscape(clientID),
			url.QueryEscape(redirectURI),
			url.QueryEscape(scope),
			url.QueryEscape(state),
		)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	if clientID == "" {
		return "", fmt.Errorf("OAuth not configured for provider: %s", provider)
	}

	// Сохраняем state в meta (временное решение, для продакшена нужна отдельная таблица)
	// Для MVP просто возвращаем URL, state будет в callback

	return authURL, nil
}

// ExchangeCodeForToken обменивает authorization code на access token.
func (s *OAuthService) ExchangeCodeForToken(provider string, code string, redirectURI string) (map[string]interface{}, error) {
	var tokenURL string
	var clientID string
	var clientSecret string
	var grantType string

	switch provider {
	case "mailru":
		tokenURL = "https://oauth.mail.ru/token"
		clientID = s.cfg.MailruClientID
		clientSecret = s.cfg.MailruClientSecret
		grantType = "authorization_code"
	case "yandex":
		tokenURL = "https://oauth.yandex.ru/token"
		clientID = s.cfg.YandexClientID
		clientSecret = s.cfg.YandexClientSecret
		grantType = "authorization_code"
	case "gmail":
		tokenURL = "https://oauth2.googleapis.com/token"
		clientID = s.cfg.GmailClientID
		clientSecret = s.cfg.GmailClientSecret
		grantType = "authorization_code"
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	// Формируем запрос
	data := url.Values{}
	data.Set("grant_type", grantType)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	// Для Gmail нужен другой формат
	if provider == "gmail" {
		req, err := http.NewRequest("POST", tokenURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.URL.RawQuery = data.Encode()

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("token exchange failed: %s", string(body))
		}

		return result, nil
	}

	// Для Mail.ru и Yandex используем стандартный POST
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	return result, nil
}

// GetUserEmail получает email пользователя из провайдера (для проверки).
func (s *OAuthService) GetUserEmail(provider string, accessToken string) (string, error) {
	var userInfoURL string

	switch provider {
	case "mailru":
		userInfoURL = "https://oauth.mail.ru/userinfo?access_token=" + url.QueryEscape(accessToken)
	case "yandex":
		userInfoURL = "https://login.yandex.ru/info?format=json"
	case "gmail":
		// Для Gmail используем Gmail API
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return "", err
	}

	if provider == "yandex" || provider == "gmail" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get user info: %s", string(body))
	}

	// Извлекаем email в зависимости от провайдера
	var email string
	switch provider {
	case "mailru":
		if e, ok := userInfo["email"].(string); ok {
			email = e
		}
	case "yandex":
		if e, ok := userInfo["default_email"].(string); ok {
			email = e
		} else if e, ok := userInfo["email"].(string); ok {
			email = e
		}
	case "gmail":
		if e, ok := userInfo["email"].(string); ok {
			email = e
		}
	}

	if email == "" {
		return "", fmt.Errorf("email not found in user info")
	}

	return email, nil
}

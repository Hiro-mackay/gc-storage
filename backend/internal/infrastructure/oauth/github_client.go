package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

const (
	githubTokenURL    = "https://github.com/login/oauth/access_token"
	githubUserInfoURL = "https://api.github.com/user"
	githubEmailsURL   = "https://api.github.com/user/emails"
)

// GitHubClient はGitHub OAuthクライアントの実装です
type GitHubClient struct {
	clientID     string
	clientSecret string
	redirectURL  string
	httpClient   *http.Client
}

// NewGitHubClient は新しいGitHubClientを作成します
func NewGitHubClient(clientID, clientSecret, redirectURL string) *GitHubClient {
	return &GitHubClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		httpClient:   &http.Client{},
	}
}

// githubTokenResponse はGitHubのトークンレスポンスを表します
type githubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// githubUserInfoResponse はGitHubのユーザー情報レスポンスを表します
type githubUserInfoResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// githubEmailResponse はGitHubのメール情報レスポンスを表します
type githubEmailResponse struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// ExchangeCode は認可コードをトークンに交換します
func (c *GitHubClient) ExchangeCode(ctx context.Context, code string) (*service.OAuthTokens, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("redirect_uri", c.redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github token exchange failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var tokenResp githubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// GitHubはrefresh_tokenを返さない（トークンは有効期限なし）
	return &service.OAuthTokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: "",
		ExpiresIn:    0, // GitHubトークンは有効期限なし
	}, nil
}

// GetUserInfo はアクセストークンを使用してユーザー情報を取得します
func (c *GitHubClient) GetUserInfo(ctx context.Context, accessToken string) (*service.OAuthUserInfo, error) {
	// ユーザー基本情報を取得
	userResp, err := c.getUserBasicInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	// メールアドレスを取得
	email := userResp.Email
	if email == "" {
		// プライマリメールを取得
		email, err = c.getPrimaryEmail(ctx, accessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to get email: %w", err)
		}
	}

	name := userResp.Name
	if name == "" {
		name = userResp.Login
	}

	return &service.OAuthUserInfo{
		ProviderUserID: strconv.FormatInt(userResp.ID, 10),
		Email:          email,
		Name:           name,
		AvatarURL:      userResp.AvatarURL,
	}, nil
}

// getUserBasicInfo はユーザー基本情報を取得します
func (c *GitHubClient) getUserBasicInfo(ctx context.Context, accessToken string) (*githubUserInfoResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github user info failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var userResp githubUserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode user info response: %w", err)
	}

	return &userResp, nil
}

// getPrimaryEmail はプライマリメールアドレスを取得します
func (c *GitHubClient) getPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubEmailsURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github emails failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var emails []githubEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("failed to decode emails response: %w", err)
	}

	// プライマリで確認済みのメールを探す
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	// プライマリがなければ確認済みのものを探す
	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}

	return "", fmt.Errorf("no verified email found")
}

// Provider はプロバイダー種別を返します
func (c *GitHubClient) Provider() valueobject.OAuthProvider {
	return valueobject.OAuthProviderGitHub
}

// インターフェースの実装を保証
var _ service.OAuthClient = (*GitHubClient)(nil)

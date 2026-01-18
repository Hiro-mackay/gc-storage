package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// ErrSessionNotFound はセッションが見つからないエラーを表します
var ErrSessionNotFound = errors.New("session not found")

// sessionData はRedisに保存するセッションデータを表します（内部用）
type sessionData struct {
	ID           string    `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	IPAddress    string    `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
}

// toSessionData はentity.SessionからsessionDataに変換します
func toSessionData(s *entity.Session) *sessionData {
	return &sessionData{
		ID:           s.ID,
		UserID:       s.UserID,
		RefreshToken: s.RefreshToken,
		UserAgent:    s.UserAgent,
		IPAddress:    s.IPAddress,
		CreatedAt:    s.CreatedAt,
		ExpiresAt:    s.ExpiresAt,
		LastUsedAt:   s.LastUsedAt,
	}
}

// toEntity はsessionDataからentity.Sessionに変換します
func (d *sessionData) toEntity() *entity.Session {
	return &entity.Session{
		ID:           d.ID,
		UserID:       d.UserID,
		RefreshToken: d.RefreshToken,
		UserAgent:    d.UserAgent,
		IPAddress:    d.IPAddress,
		CreatedAt:    d.CreatedAt,
		ExpiresAt:    d.ExpiresAt,
		LastUsedAt:   d.LastUsedAt,
	}
}

// SessionStore はセッションの永続化を提供します
type SessionStore struct {
	client *redis.Client
	ttl    time.Duration // デフォルト: 7日（リフレッシュトークン有効期限）
}

// NewSessionStore は新しいSessionStoreを作成します
func NewSessionStore(client *redis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{
		client: client,
		ttl:    ttl,
	}
}

// Save はセッションを保存します
func (s *SessionStore) Save(ctx context.Context, session *entity.Session) error {
	data, err := json.Marshal(toSessionData(session))
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := SessionKey(session.ID)
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = s.ttl
	}

	// パイプラインで複数操作をアトミックに実行
	pipe := s.client.TxPipeline()

	// セッションデータを保存
	pipe.Set(ctx, key, data, ttl)

	// ユーザーのセッション一覧に追加
	userSessionsKey := UserSessionsKey(session.UserID)
	pipe.SAdd(ctx, userSessionsKey, session.ID)
	pipe.Expire(ctx, userSessionsKey, 30*24*time.Hour) // 30日

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// FindByID はセッションIDでセッションを取得します
func (s *SessionStore) FindByID(ctx context.Context, sessionID string) (*entity.Session, error) {
	key := SessionKey(sessionID)

	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var sd sessionData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return sd.toEntity(), nil
}

// FindByUserID はユーザーIDで全セッションを取得します
func (s *SessionStore) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Session, error) {
	userSessionsKey := UserSessionsKey(userID)

	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessionIDs) == 0 {
		return []*entity.Session{}, nil
	}

	// パイプラインで一括取得
	pipe := s.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(sessionIDs))
	for i, sessionID := range sessionIDs {
		cmds[i] = pipe.Get(ctx, SessionKey(sessionID))
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	sessions := make([]*entity.Session, 0, len(sessionIDs))
	expiredIDs := make([]string, 0)

	for i, cmd := range cmds {
		data, err := cmd.Bytes()
		if err != nil {
			if err == redis.Nil {
				// 期限切れセッションをマーク
				expiredIDs = append(expiredIDs, sessionIDs[i])
				continue
			}
			continue
		}

		var sd sessionData
		if err := json.Unmarshal(data, &sd); err != nil {
			continue
		}
		sessions = append(sessions, sd.toEntity())
	}

	// 期限切れセッションをユーザーセッション一覧から削除（非同期）
	if len(expiredIDs) > 0 {
		go func() {
			bgCtx := context.Background()
			args := make([]interface{}, len(expiredIDs))
			for i, id := range expiredIDs {
				args[i] = id
			}
			s.client.SRem(bgCtx, userSessionsKey, args...)
		}()
	}

	return sessions, nil
}

// Delete はセッションを削除します
func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	session, err := s.FindByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil // すでに削除済み
		}
		return err
	}

	pipe := s.client.TxPipeline()
	pipe.Del(ctx, SessionKey(sessionID))
	pipe.SRem(ctx, UserSessionsKey(session.UserID), sessionID)

	_, err = pipe.Exec(ctx)
	return err
}

// DeleteByUserID はユーザーの全セッションを削除します（ログアウトオール）
func (s *SessionStore) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	userSessionsKey := UserSessionsKey(userID)

	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessionIDs) == 0 {
		return nil
	}

	pipe := s.client.TxPipeline()

	// 全セッションキーを削除
	for _, sessionID := range sessionIDs {
		pipe.Del(ctx, SessionKey(sessionID))
	}

	// ユーザーセッション一覧を削除
	pipe.Del(ctx, userSessionsKey)

	_, err = pipe.Exec(ctx)
	return err
}

// CountByUserID はユーザーのセッション数を返します
func (s *SessionStore) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.client.SCard(ctx, UserSessionsKey(userID)).Result()
}

// インターフェースの実装を保証
var _ repository.SessionRepository = (*SessionStore)(nil)

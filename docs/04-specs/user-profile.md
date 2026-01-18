# ユーザープロファイル仕様書

## 概要

本ドキュメントでは、GC Storageにおけるユーザープロファイル管理の実装仕様を定義します。

**関連ドキュメント:**
- [user.md](../03-domains/user.md) - ユーザードメイン定義
- [auth-identity.md](./auth-identity.md) - 認証・アイデンティティ仕様

---

## 1. エンティティ

### UserProfile

```go
// internal/domain/entity/user_profile.go

package entity

import (
    "encoding/json"
    "time"

    "github.com/google/uuid"
)

// UserProfileSettings はユーザー設定を定義します
type UserProfileSettings struct {
    NotificationsEnabled bool   `json:"notifications_enabled"`
    EmailNotifications   bool   `json:"email_notifications"`
    Theme                string `json:"theme,omitempty"`
}

// UserProfile はユーザープロファイルエンティティを定義します
type UserProfile struct {
    UserID      uuid.UUID
    DisplayName string
    AvatarURL   string
    Bio         string
    Locale      string
    Timezone    string
    Settings    UserProfileSettings
    UpdatedAt   time.Time
}

// NewUserProfile は新しいUserProfileを作成します
func NewUserProfile(userID uuid.UUID) *UserProfile {
    return &UserProfile{
        UserID:   userID,
        Locale:   "ja",
        Timezone: "Asia/Tokyo",
        Settings: UserProfileSettings{
            NotificationsEnabled: true,
            EmailNotifications:   true,
            Theme:                "system",
        },
        UpdatedAt: time.Now(),
    }
}

// ValidateBio はbioの長さを検証します
func (p *UserProfile) ValidateBio() bool {
    return len([]rune(p.Bio)) <= 500
}
```

---

## 2. リポジトリ

### UserProfileRepository

```go
// internal/domain/repository/user_profile_repository.go

package repository

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// UserProfileRepository はユーザープロファイルリポジトリインターフェースを定義します
type UserProfileRepository interface {
    // Create はユーザープロファイルを作成します
    Create(ctx context.Context, profile *entity.UserProfile) error

    // Update はユーザープロファイルを更新します
    Update(ctx context.Context, profile *entity.UserProfile) error

    // FindByUserID はユーザーIDでプロファイルを検索します
    FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.UserProfile, error)

    // Upsert はプロファイルを作成または更新します
    Upsert(ctx context.Context, profile *entity.UserProfile) error
}
```

---

## 3. APIエンドポイント

### 3.1 プロファイル取得

```
GET /api/v1/me/profile
```

**認証:** 必須（Bearer Token）

**レスポンス:**
```json
{
  "profile": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "display_name": "山田太郎",
    "avatar_url": "https://example.com/avatar.png",
    "bio": "自己紹介文",
    "locale": "ja",
    "timezone": "Asia/Tokyo",
    "settings": {
      "notifications_enabled": true,
      "email_notifications": true,
      "theme": "system"
    }
  },
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "yamada@example.com",
    "name": "山田太郎",
    "status": "active",
    "email_verified": true,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 3.2 プロファイル更新

```
PUT /api/v1/me/profile
```

**認証:** 必須（Bearer Token）

**リクエスト:**
```json
{
  "display_name": "新しい表示名",
  "avatar_url": "https://example.com/new-avatar.png",
  "bio": "新しい自己紹介",
  "locale": "en",
  "timezone": "UTC",
  "settings": {
    "notifications_enabled": false,
    "email_notifications": true,
    "theme": "dark"
  }
}
```

**注記:** すべてのフィールドはオプショナルです。指定されたフィールドのみ更新されます。

**レスポンス:**
```json
{
  "profile": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "display_name": "新しい表示名",
    "avatar_url": "https://example.com/new-avatar.png",
    "bio": "新しい自己紹介",
    "locale": "en",
    "timezone": "UTC",
    "settings": {
      "notifications_enabled": false,
      "email_notifications": true,
      "theme": "dark"
    }
  }
}
```

**エラーレスポンス:**

| ステータス | エラーコード | 説明 |
|-----------|------------|------|
| 401 | unauthorized | 認証エラー |
| 400 | validation_error | バリデーションエラー（bioが500文字超過など） |
| 404 | not_found | ユーザーが見つからない |

---

## 4. ユースケース

### 4.1 GetProfileQuery（プロファイル取得）

```go
// internal/usecase/profile/query/get_profile.go

package query

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GetProfileInput はプロファイル取得の入力を定義します
type GetProfileInput struct {
    UserID uuid.UUID
}

// GetProfileOutput はプロファイル取得の出力を定義します
type GetProfileOutput struct {
    Profile *entity.UserProfile
    User    *entity.User
}

// GetProfileQuery はプロファイル取得クエリです
type GetProfileQuery struct {
    profileRepo repository.UserProfileRepository
    userRepo    repository.UserRepository
}

// NewGetProfileQuery は新しいGetProfileQueryを作成します
func NewGetProfileQuery(
    profileRepo repository.UserProfileRepository,
    userRepo repository.UserRepository,
) *GetProfileQuery {
    return &GetProfileQuery{
        profileRepo: profileRepo,
        userRepo:    userRepo,
    }
}

// Execute はプロファイル取得を実行します
func (q *GetProfileQuery) Execute(ctx context.Context, input GetProfileInput) (*GetProfileOutput, error) {
    // ユーザーを取得
    user, err := q.userRepo.FindByID(ctx, input.UserID)
    if err != nil {
        return nil, apperror.NewNotFoundError("user")
    }

    // プロファイルを取得
    profile, err := q.profileRepo.FindByUserID(ctx, input.UserID)
    if err != nil {
        // プロファイルが存在しない場合はデフォルトを返す
        if apperror.IsNotFound(err) {
            profile = entity.NewUserProfile(input.UserID)
        } else {
            return nil, err
        }
    }

    return &GetProfileOutput{
        Profile: profile,
        User:    user,
    }, nil
}
```

### 4.2 UpdateProfileCommand（プロファイル更新）

```go
// internal/usecase/profile/command/update_profile.go

package command

import (
    "context"
    "time"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// UpdateProfileInput はプロファイル更新の入力を定義します
type UpdateProfileInput struct {
    UserID      uuid.UUID
    DisplayName *string
    AvatarURL   *string
    Bio         *string
    Locale      *string
    Timezone    *string
    Settings    *entity.UserProfileSettings
}

// UpdateProfileOutput はプロファイル更新の出力を定義します
type UpdateProfileOutput struct {
    Profile *entity.UserProfile
}

// UpdateProfileCommand はプロファイル更新コマンドです
type UpdateProfileCommand struct {
    profileRepo repository.UserProfileRepository
    userRepo    repository.UserRepository
}

// NewUpdateProfileCommand は新しいUpdateProfileCommandを作成します
func NewUpdateProfileCommand(
    profileRepo repository.UserProfileRepository,
    userRepo repository.UserRepository,
) *UpdateProfileCommand {
    return &UpdateProfileCommand{
        profileRepo: profileRepo,
        userRepo:    userRepo,
    }
}

// Execute はプロファイル更新を実行します
func (c *UpdateProfileCommand) Execute(ctx context.Context, input UpdateProfileInput) (*UpdateProfileOutput, error) {
    // ユーザーの存在確認
    _, err := c.userRepo.FindByID(ctx, input.UserID)
    if err != nil {
        return nil, apperror.NewNotFoundError("user")
    }

    // 既存プロファイルを取得（なければデフォルト作成）
    profile, err := c.profileRepo.FindByUserID(ctx, input.UserID)
    if err != nil {
        if apperror.IsNotFound(err) {
            profile = entity.NewUserProfile(input.UserID)
        } else {
            return nil, err
        }
    }

    // フィールドを更新
    if input.DisplayName != nil {
        profile.DisplayName = *input.DisplayName
    }
    if input.AvatarURL != nil {
        profile.AvatarURL = *input.AvatarURL
    }
    if input.Bio != nil {
        profile.Bio = *input.Bio
        // bioの長さ検証
        if !profile.ValidateBio() {
            return nil, apperror.NewValidationError("bio must not exceed 500 characters", nil)
        }
    }
    if input.Locale != nil {
        profile.Locale = *input.Locale
    }
    if input.Timezone != nil {
        profile.Timezone = *input.Timezone
    }
    if input.Settings != nil {
        profile.Settings = *input.Settings
    }

    profile.UpdatedAt = time.Now()

    // Upsertで保存
    if err := c.profileRepo.Upsert(ctx, profile); err != nil {
        return nil, apperror.NewInternalError(err)
    }

    return &UpdateProfileOutput{Profile: profile}, nil
}
```

---

## 5. OAuthログインとの連携

OAuth新規ユーザー作成時、UserProfileも同時に作成されます。

```go
// OAuthLoginCommand内での処理

// 新規ユーザー作成後
profile := entity.NewUserProfile(user.ID)
profile.DisplayName = userInfo.Name
profile.AvatarURL = userInfo.AvatarURL
if txErr = c.profileRepo.Upsert(ctx, profile); txErr != nil {
    return txErr
}
```

**注記:**
- User エンティティには AvatarURL フィールドは存在しません
- アバター画像は UserProfile.AvatarURL に保存されます
- OAuthプロバイダーから取得したアバターURLは、新規ユーザー作成時にUserProfileに保存されます

---

## 6. 受け入れ基準

### 6.1 機能要件

| 項目 | 基準 |
|------|------|
| プロファイル取得 | GET /api/v1/me/profile でプロファイルとユーザー情報を取得できる |
| プロファイル更新 | PUT /api/v1/me/profile で指定フィールドのみ更新できる |
| デフォルト値 | プロファイル未作成時はデフォルト値（locale: ja, timezone: Asia/Tokyo）で返す |
| バリデーション | bioは500文字以内に制限される |

### 6.2 セキュリティ要件

| 項目 | 基準 |
|------|------|
| 認証 | すべてのエンドポイントでJWT認証が必須 |
| 認可 | 自分のプロファイルのみ取得・更新可能 |

### 6.3 チェックリスト

- [x] プロファイル取得が正常に動作する
- [x] プロファイル未作成時にデフォルト値が返る
- [x] プロファイル更新が正常に動作する
- [x] bioの長さバリデーションが動作する
- [x] 認証なしでのアクセスが拒否される
- [x] OAuthログインでUserProfileが作成される

---

## 関連ドキュメント

- [user.md](../03-domains/user.md) - ユーザードメイン定義
- [auth-identity.md](./auth-identity.md) - 認証・アイデンティティ仕様

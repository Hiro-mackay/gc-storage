# User Profile + Settings

## Meta

| Item | Value |
|------|-------|
| Status | Ready |
| Priority | Medium |
| Tier | 1 (Auth) |
| Domain Refs | `03-domains/user.md` |
| Depends On | `features/auth-login.md` |

---

## 1. User Stories

**Primary:**
> As a logged-in user, I want to view and update my profile settings so that I can personalize my experience.

**Secondary:**
> As a user, I want to change my display name, avatar, theme, and notification preferences from the settings page.

### Context
ユーザープロファイルはUserProfileエンティティで管理される。表示名(name)はusersテーブル、その他の設定（avatar, bio, theme, locale, timezone, notifications）はuser_profilesテーブルで管理。プロファイル未作成時はデフォルト値を返す。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-U003 | nameは空文字不可、1-100文字 | `03-domains/user.md` |
| R-UP001 | bioは最大500文字 | `03-domains/user.md` |
| R-UP002 | avatar_urlは有効なURL形式 | `03-domains/user.md` |
| R-UP003 | themeは"system", "light", "dark"のいずれか | `03-domains/user.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-PROF-001 | プロファイル未作成時はデフォルト値を返す (locale: ja, timezone: Asia/Tokyo, theme: system) |
| FS-PROF-002 | プロファイル更新はUpsert方式（存在しなければ作成） |
| FS-PROF-003 | 表示名の更新はPUT /api/v1/meで行う（User entity管理） |
| FS-PROF-004 | プロファイル更新は指定フィールドのみ更新（部分更新） |
| FS-PROF-005 | OAuthログインで新規作成時、AvatarURLをプロバイダーから取得 |

### State Transitions
```
N/A (CRUD operation, no state machine)
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/me` | Required | ユーザー基本情報取得 |
| PUT | `/api/v1/me` | Required | ユーザー基本情報更新 (name等) |
| GET | `/api/v1/me/profile` | Required | プロファイル取得 |
| PUT | `/api/v1/me/profile` | Required | プロファイル更新 |

### Request / Response Details

#### `GET /api/v1/me/profile` - プロファイル取得

**Success Response (200):**
```json
{
  "profile": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "avatar_url": "https://example.com/avatar.png",
    "bio": "Hello!",
    "locale": "ja",
    "timezone": "Asia/Tokyo",
    "theme": "system",
    "notification_preferences": {
      "email_enabled": true,
      "push_enabled": true
    }
  },
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "taro@example.com",
    "name": "Taro Yamada",
    "status": "active",
    "email_verified": true,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 401 | 未認証 | `UNAUTHORIZED` |

#### `PUT /api/v1/me` - ユーザー基本情報更新

**Request Body:**
```json
{
  "name": "New Name"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| name | string | No | 1-100文字 | 表示名 |

**Success Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "taro@example.com",
  "name": "New Name",
  "status": "active",
  "email_verified": true,
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### `PUT /api/v1/me/profile` - プロファイル更新

**Request Body (all fields optional):**
```json
{
  "avatar_url": "https://example.com/new-avatar.png",
  "bio": "Updated bio",
  "locale": "en",
  "timezone": "UTC",
  "theme": "dark",
  "notification_preferences": {
    "email_enabled": false,
    "push_enabled": true
  }
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| avatar_url | string | No | 有効なURL | アバター画像URL |
| bio | string | No | max 500文字 | 自己紹介 |
| locale | string | No | - | 言語設定 |
| timezone | string | No | - | タイムゾーン |
| theme | string | No | system/light/dark | テーマ設定 |
| notification_preferences | object | No | - | 通知設定 |

**Success Response (200):**
```json
{
  "profile": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "avatar_url": "https://example.com/new-avatar.png",
    "bio": "Updated bio",
    "locale": "en",
    "timezone": "UTC",
    "theme": "dark",
    "notification_preferences": {
      "email_enabled": false,
      "push_enabled": true
    }
  }
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | bioが500文字超過 | `VALIDATION_ERROR` |
| 401 | 未認証 | `UNAUTHORIZED` |
| 404 | ユーザーが見つからない | `NOT_FOUND` |

---

## 4. Frontend UI

### Layout / Wireframe
```
[Settings Page - Profile Tab]
+-------------------------------------------+
| Settings                                  |
| [Profile] [Appearance] [Notifications]    |
|                                           |
|  Profile                                  |
|  +-------------------------------------+  |
|  | (Avatar)  [Upload new]              |  |
|  |                                     |  |
|  | Display Name                        |  |
|  | [Taro Yamada      ]                 |  |
|  |                                     |  |
|  | Email                               |  |
|  | taro@example.com (read-only)        |  |
|  |                                     |  |
|  | Bio                                 |  |
|  | [Hello!                        ]    |  |
|  |                                     |  |
|  | Locale                              |  |
|  | [Japanese      v]                   |  |
|  |                                     |  |
|  | Timezone                            |  |
|  | [Asia/Tokyo    v]                   |  |
|  |                                     |  |
|  | [Save changes]                      |  |
|  +-------------------------------------+  |
+-------------------------------------------+

[Settings Page - Appearance Tab]
+-------------------------------------------+
| Settings                                  |
| [Profile] [Appearance] [Notifications]    |
|                                           |
|  Appearance                               |
|  +-------------------------------------+  |
|  | Theme                               |  |
|  | ( ) System  ( ) Light  ( ) Dark     |  |
|  +-------------------------------------+  |
+-------------------------------------------+

[Settings Page - Notifications Tab]
+-------------------------------------------+
| Settings                                  |
| [Profile] [Appearance] [Notifications]    |
|                                           |
|  Notifications                            |
|  +-------------------------------------+  |
|  | Email notifications  [toggle on]    |  |
|  | Push notifications   [toggle on]    |  |
|  +-------------------------------------+  |
+-------------------------------------------+
```

### Components

| Component | Type | Description |
|-----------|------|-------------|
| SettingsPage | Page | 設定ページ (`/settings`) |
| ProfileTab | Tab | プロファイル設定タブ |
| AppearanceTab | Tab | テーマ設定タブ |
| NotificationsTab | Tab | 通知設定タブ |
| AvatarUpload | UI | アバターアップロード |
| ThemeSelector | UI | テーマ選択 (system/light/dark) |

### State Management

| State | Store | Type | Description |
|-------|-------|------|-------------|
| profileQuery | TanStack Query | Server | GET /me/profile データ |
| updateProfileMutation | TanStack Query | Server | PUT /me/profile API呼び出し |
| updateUserMutation | TanStack Query | Server | PUT /me API呼び出し |
| themeState | Zustand | Global | 現在のテーマ (UI即時反映) |
| activeTab | useState | Local | 現在のアクティブタブ |

### User Interactions
1. 設定ページ表示 -> プロファイルデータ取得
2. フィールド編集 -> 「Save changes」クリック -> API呼び出し -> 成功メッセージ
3. テーマ変更 -> 即時UI反映 + API保存
4. 通知設定トグル -> API保存

---

## 5. Integration Flow

### Profile Load Sequence
```
Client          Frontend        API             DB
  |                |              |               |
  |-- navigate --->|              |               |
  |   /settings    |-- GET ------>|               |
  |                |  /me/profile |-- find prof ->|
  |                |              |-- find user ->|
  |                |<-- 200 ------|               |
  |<-- render -----|              |               |
```

### Profile Update Sequence
```
Client          Frontend        API             DB
  |                |              |               |
  |-- save ------->|              |               |
  |                |-- PUT ------>|               |
  |                |  /me/profile |-- find user ->|
  |                |              |-- upsert prof>|
  |                |<-- 200 ------|               |
  |<-- success ----|              |               |
```

### Display Name Update Sequence
```
Client          Frontend        API             DB
  |                |              |               |
  |-- save ------->|              |               |
  |                |-- PUT ------>|               |
  |                |  /me         |-- find user ->|
  |                |              |-- update user>|
  |                |<-- 200 ------|               |
  |<-- success ----|              |               |
```

### Error Handling Flow
- バリデーションエラー: 400 -> フィールドごとにエラー表示
- 未認証: 401 -> ログインページへリダイレクト

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: Given ログイン状態, when /settings に遷移, then プロファイルとユーザー情報が表示される
- [ ] AC-02: Given プロファイル未作成, when GET /me/profile, then デフォルト値が返る
- [ ] AC-03: Given 有効な入力, when プロファイル更新送信, then 指定フィールドのみ更新される
- [ ] AC-04: Given 有効なname, when ユーザー情報更新送信, then nameが更新される
- [ ] AC-05: Given テーマ変更, when Dark選択, then UIが即時反映しAPIに保存される

### Validation Errors
- [ ] AC-10: Given bioが500文字超過, when プロファイル更新送信, then バリデーションエラーが返る
- [ ] AC-11: Given nameが空, when ユーザー情報更新送信, then バリデーションエラーが返る
- [ ] AC-12: Given nameが100文字超過, when ユーザー情報更新送信, then バリデーションエラーが返る

### Authorization
- [ ] AC-20: Given 未認証, when /settings にアクセス, then ログインページへリダイレクト
- [ ] AC-21: Given 未認証, when PUT /me/profile, then 401が返る

### Edge Cases
- [ ] AC-30: Given OAuthユーザー, when プロファイル取得, then アバターURLがプロバイダーから取得した値で表示される
- [ ] AC-31: Given 部分更新 (bioのみ), when プロファイル更新送信, then bio以外は変更されない

---

## 7. Test Plan

### Backend Unit Tests

| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| プロファイル取得 | GetProfileQuery | Profile + User返却 |
| プロファイル未作成時 | GetProfileQuery | デフォルト値で返却 |
| プロファイル更新（全フィールド） | UpdateProfileCommand | 全フィールド更新 |
| プロファイル更新（部分） | UpdateProfileCommand | 指定フィールドのみ更新 |
| bio長さバリデーション | UpdateProfileCommand | 500文字超過でエラー |
| ユーザー情報更新 | UpdateUserCommand | name更新 |

### Backend Integration Tests

| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| プロファイル取得 | GET /me/profile | auth session | 200, profile + user |
| プロファイル更新 | PUT /me/profile | auth session | 200, updated profile |
| bio超過 | PUT /me/profile | auth + long bio | 400 |
| ユーザー名更新 | PUT /me | auth session | 200, updated name |
| 未認証アクセス | GET /me/profile | no session | 401 |

### Frontend Tests

| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| プロファイル表示 | ProfileTab | Unit | 各フィールド表示 |
| プロファイル更新 | ProfileTab | Integration | API呼び出し、成功表示 |
| テーマ切替 | AppearanceTab | Unit | UI即時反映 |
| 通知トグル | NotificationsTab | Unit | トグル操作、API保存 |
| タブ切替 | SettingsPage | Unit | タブコンテンツ切替 |

### E2E Tests (future)

| Test | Flow | Assertions |
|------|------|------------|
| プロファイル更新フロー | settings -> edit -> save | 更新値が反映 |
| テーマ変更フロー | settings -> appearance -> dark | ダークテーマ適用 |

---

## 8. Implementation Notes

### Changed Files (Backend)

| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/entity/user_profile.go` | UserProfile entity |
| Domain | `internal/domain/repository/user_profile_repository.go` | UserProfileRepo IF |
| UseCase | `internal/usecase/profile/query/get_profile.go` | GetProfileQuery |
| UseCase | `internal/usecase/profile/command/update_profile.go` | UpdateProfileCommand |
| Interface | `internal/interface/handler/user_handler.go` | Me, UpdateMe handlers |
| Interface | `internal/interface/handler/profile_handler.go` | GetProfile, UpdateProfile |
| Interface | `internal/interface/dto/request/profile.go` | Request DTOs |
| Interface | `internal/interface/dto/response/profile.go` | Response DTOs |
| Infra | `internal/infrastructure/database/user_profile_repository.go` | Repo impl |

### Changed Files (Frontend)

| Category | File | Change |
|----------|------|--------|
| Page | `src/features/settings/pages/settings-page.tsx` | 設定ページ |
| Component | `src/features/settings/components/profile-tab.tsx` | プロファイル設定タブ |
| Component | `src/features/settings/components/appearance-tab.tsx` | テーマ設定タブ |
| Component | `src/features/settings/components/notifications-tab.tsx` | 通知設定タブ |
| Component | `src/components/ui/avatar-upload.tsx` | アバターアップロード（共通） |
| Store | `src/stores/ui-store.ts` | テーマ状態管理 (Zustand) |
| Query | `src/features/settings/api/queries.ts` | profile |
| Mutation | `src/features/settings/api/mutations.ts` | updateProfile, updateUser |

### Migration
```sql
-- user_profiles table
-- See existing migrations
```

### Considerations
- **Performance**: プロファイル取得はUser + UserProfileの2クエリだがN+1ではないため問題なし
- **UX**: テーマ変更はAPI保存前にUI即時反映（楽観的更新）
- **Data**: 表示名はusers.name、その他設定はuser_profilesで分離管理

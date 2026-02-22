# Sharing - 共有リンク + パブリックアクセス

## Meta

| Item | Value |
|------|-------|
| Status | Draft |
| Priority | Medium |
| Tier | 4 (Collab) |
| Domain Refs | `03-domains/sharing.md` |
| Depends On | `features/permission-management.md` |

---

## 1. User Stories

**Primary:**
> As a user with share permission, I want to create a share link so that anyone with the link can access the file or folder.

**Secondary:**
> As a guest, I want to access a shared resource via a link so that I can view or download files without an account.

### Context
共有リンクは認証不要でリソースにアクセスできる仕組みを提供する。パスワード保護、有効期限、アクセス回数制限をサポート。URL-safe Base62トークン（32文字以上）を使用。権限レベルはread（閲覧/ダウンロード）とwrite（read + アップロード/名前変更）の2段階。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-SL001 | tokenは全共有リンクで一意 | `03-domains/sharing.md` |
| R-SL002 | tokenはURL-safe（Base62）、32文字以上 | `03-domains/sharing.md` |
| R-SL003 | expires_at到達後はアクセス不可 | `03-domains/sharing.md` |
| R-SL004 | max_access_count到達後はアクセス不可 | `03-domains/sharing.md` |
| R-SL005 | statusがrevokedの場合はアクセス不可 | `03-domains/sharing.md` |
| R-SL006 | 作成者はfile:share/folder:share権限が必要 | `03-domains/sharing.md` |
| R-SLA001 | アクセス履歴を監査目的で記録 | `03-domains/sharing.md` |
| R-SLA002 | IPアドレスは一定期間後に匿名化 | `03-domains/sharing.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-SH001 | パスワードはbcrypt（cost=12）でハッシュ化して保存 |
| FS-SH002 | パスワードは最低4文字 |
| FS-SH003 | 無効化されたリンクは復活不可 |
| FS-SH004 | 更新・無効化は作成者のみ可能 |
| FS-SH005 | PresignedURLの有効期限は15分 |
| FS-SH006 | アクセスログの個人情報は90日後に匿名化 |

### State Transitions
```
+---------+       +---------+
|  active |------>| revoked |
+----+----+       +---------+
     |
     +----------->+---------+
                  | expired | (auto)
                  +---------+
```

---

## 3. API Contract

### Endpoints - 認証必要

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/files/:id/share` | Required | ファイル共有リンク作成 |
| POST | `/api/v1/folders/:id/share` | Required | フォルダ共有リンク作成 |
| GET | `/api/v1/files/:id/share-links` | Required | ファイル共有リンク一覧 |
| GET | `/api/v1/folders/:id/share-links` | Required | フォルダ共有リンク一覧 |
| PATCH | `/api/v1/share-links/:id` | Required | 共有リンク更新 |
| DELETE | `/api/v1/share-links/:id` | Required | 共有リンク無効化 |
| GET | `/api/v1/share-links/:id/history` | Required | アクセス履歴 |

### Endpoints - 認証不要（パブリック）

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/share/:token` | None | リンク情報取得 |
| POST | `/api/v1/share/:token/access` | None | リンクアクセス |
| GET | `/api/v1/share/:token/download` | None | ファイルダウンロード |

### Request / Response Details

#### `POST /api/v1/{files|folders}/:id/share` - 共有リンク作成

**Request Body:**
```json
{
  "permission": "read",
  "password": "optional-password",
  "expires_at": "2026-03-01T00:00:00Z",
  "max_access_count": 100
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| permission | string | Yes | oneof: read, write | 権限レベル |
| password | string | No | min:4 | パスワード保護 |
| expires_at | timestamp | No | future date | 有効期限 |
| max_access_count | int | No | min:1 | 最大アクセス回数 |

**Success Response (201):**
```json
{
  "id": "uuid", "token": "abc123...", "url": "https://domain/share/abc123...",
  "permission": "read", "has_password": false,
  "expires_at": "2026-03-01T00:00:00Z", "max_access_count": 100,
  "access_count": 0, "created_at": "timestamp"
}
```

#### `POST /api/v1/share/:token/access` - リンクアクセス

**Request Body:**
```json
{ "password": "optional-if-required" }
```

**Success Response (200):**
```json
{
  "resource_type": "file", "resource_id": "uuid",
  "resource_name": "report.pdf", "permission": "read",
  "presigned_url": "https://minio/...",
  "contents": null
}
```

For folders:
```json
{
  "resource_type": "folder", "resource_id": "uuid",
  "resource_name": "Shared Docs", "permission": "read",
  "presigned_url": null,
  "contents": [
    { "id": "uuid", "name": "sub-folder", "type": "folder" },
    { "id": "uuid", "name": "file.pdf", "type": "file", "size": 1024, "mime_type": "application/pdf" }
  ]
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | Invalid token / options | `VALIDATION_ERROR` |
| 401 | Wrong password | `UNAUTHORIZED` |
| 403 | Non-creator update/revoke | `FORBIDDEN` |
| 404 | Link or resource not found | `NOT_FOUND` |
| 410 | Expired / revoked / access limit | `GONE` |

---

## 4. Frontend UI

### Layout / Wireframe
```
Share Dialog (on file/folder):
+----------------------------------+
| Share "report.pdf"               |
+----------------------------------+
| Permission: [Read only v]        |
| Password:   [          ] (opt)   |
| Expires:    [2026-03-01] (opt)   |
| Max access: [100       ] (opt)   |
|                                  |
|         [Cancel]  [Create Link]  |
+----------------------------------+

After creation:
+----------------------------------+
| Share Link Created!              |
+----------------------------------+
| https://domain/share/abc123...   |
|                        [Copy]    |
+----------------------------------+

Public Access Page (/share/:token):
+----------------------------------+
| GC Storage - Shared File         |
+----------------------------------+
| report.pdf                       |
| 2.4 MB  |  PDF Document         |
|                                  |
|          [Download]              |
+----------------------------------+

Password Required Page:
+----------------------------------+
| This link is password protected  |
+----------------------------------+
| Password: [               ]     |
|           [Access]               |
+----------------------------------+
```

### Components
| Component | Type | Description |
|-----------|------|-------------|
| ShareDialog | Modal | 共有リンク作成/管理 |
| ShareLinkList | List | リソースの共有リンク一覧 |
| PublicAccessPage | Page | パブリックアクセスページ |
| PasswordPrompt | Form | パスワード入力フォーム |
| SharedFolderBrowser | Page | 共有フォルダ閲覧 |

### State Management
| State | Store | Type | Description |
|-------|-------|------|-------------|
| shareLinks | TanStack Query | Server | リソースの共有リンク一覧 |
| sharedResource | TanStack Query | Server | 共有リソース情報 |

---

## 5. Integration Flow

### Sequence Diagram - 共有リンク作成
```
Client          Frontend        API             Resolver      DB
  |                |              |                |            |
  |-- create ----->|              |                |            |
  |                |-- POST ----->|                |            |
  |                |              |-- hasPermission>|            |
  |                |              |<-- allowed -----|            |
  |                |              |-- gen token --->|            |
  |                |              |-- hash pwd ---->|            |
  |                |              |-- insert link ->|            |
  |                |<-- 201 ------|                |            |
  |<-- show URL ---|              |                |            |
```

### Sequence Diagram - 共有リンクアクセス
```
Guest           Frontend        API             DB          MinIO
  |                |              |                |           |
  |-- access ----->|              |                |           |
  |                |-- POST ----->|                |           |
  |                |              |-- find link -->|           |
  |                |              |-- validate --->|           |
  |                |              |-- check pwd -->|           |
  |                |              |-- incr count ->|           |
  |                |              |-- log access ->|           |
  |                |              |-- get resource>|           |
  |                |              |-- presigned ------------>|
  |                |              |<-- url ------------------|
  |                |<-- 200 ------|                |           |
  |<-- show file --|              |                |           |
```

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: file:share権限を持つユーザーがファイル共有リンクを作成できる
- [ ] AC-02: folder:share権限を持つユーザーがフォルダ共有リンクを作成できる
- [ ] AC-03: read/write権限レベルを選択できる
- [ ] AC-04: トークンでリソースにアクセスできる
- [ ] AC-05: ファイルはPresignedURLでダウンロードできる
- [ ] AC-06: フォルダは内容一覧を取得できる
- [ ] AC-07: フォルダ共有時、配下のファイルをダウンロードできる
- [ ] AC-08: リソースの共有リンク一覧を取得できる

### Optional Features
- [ ] AC-10: パスワードを設定できる
- [ ] AC-11: 有効期限を設定できる
- [ ] AC-12: 最大アクセス回数を設定できる
- [ ] AC-13: 設定を後から変更できる
- [ ] AC-14: 共有リンクを無効化できる

### Validation / Access Control
- [ ] AC-20: パスワード設定時は正しいパスワードが必要
- [ ] AC-21: 有効期限切れのリンクはアクセス不可（410 GONE）
- [ ] AC-22: アクセス回数上限到達のリンクはアクセス不可
- [ ] AC-23: 無効化されたリンクはアクセス不可
- [ ] AC-24: 更新・無効化は作成者のみ可能
- [ ] AC-25: 無効化したリンクは復活不可

### Access Logging
- [ ] AC-30: すべてのアクセスがログに記録される
- [ ] AC-31: IP、User-Agent、ユーザーID（認証時）を記録
- [ ] AC-32: 90日後にIPアドレスが匿名化される

### Edge Cases
- [ ] AC-40: URL-safeな32文字以上のトークンが生成される
- [ ] AC-41: 削除されたリソースの共有リンクはアクセス不可
- [ ] AC-42: パスワード未設定のリンクはパスワードなしでアクセス可能

---

## 7. Test Plan

### Backend Unit Tests
| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| Create link (file) | CreateShareLinkCommand | Link created, token generated |
| Create link (no permission) | CreateShareLinkCommand | FORBIDDEN |
| Access link | AccessShareLinkQuery | Resource info + presigned URL |
| Access with password | AccessShareLinkQuery | Password validated |
| Access wrong password | AccessShareLinkQuery | UNAUTHORIZED |
| Access expired link | AccessShareLinkQuery | GONE |
| Access revoked link | AccessShareLinkQuery | GONE |
| Access limit reached | AccessShareLinkQuery | GONE |
| Download via share | GetDownloadViaShareQuery | Presigned URL returned |
| Download folder file | GetDownloadViaShareQuery | File in shared folder |
| Revoke link | RevokeShareLinkCommand | Status = revoked |
| Revoke (non-creator) | RevokeShareLinkCommand | FORBIDDEN |
| Update link | UpdateShareLinkCommand | Options updated |
| List links | ListShareLinksQuery | Links for resource |

### Backend Integration Tests
| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| Create share link | POST /files/:id/share | Auth + share perm | 201, link in DB |
| Access share link | POST /share/:token/access | Valid token | 200, resource info |
| Download file | GET /share/:token/download | Valid token | 200, presigned URL |
| Revoke link | DELETE /share-links/:id | Creator auth | 204 |

### Frontend Tests
| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| Share dialog | ShareDialog | Integration | Link created, URL shown |
| Password prompt | PasswordPrompt | Unit | Input validation |
| Public access page | PublicAccessPage | Integration | Resource displayed |
| Folder browser | SharedFolderBrowser | Unit | Contents listed |

### E2E Tests (future)
| Test | Flow | Assertions |
|------|------|------------|
| Full share flow | Create -> Copy URL -> Access -> Download | File downloaded |
| Password flow | Create with pwd -> Access -> Enter pwd -> Download | Protected access |

---

## 8. Implementation Notes

### Changed Files (Backend)
| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/sharing/share_link.go` | ShareLink entity |
| Domain | `internal/domain/sharing/share_token.go` | ShareToken VO |
| Domain | `internal/domain/sharing/share_link_access.go` | ShareLinkAccess entity |
| Domain | `internal/domain/sharing/share_link_options.go` | ShareLinkOptions VO |
| UseCase | `internal/usecase/sharing/create_share_link.go` | Create command |
| UseCase | `internal/usecase/sharing/access_share_link.go` | Access query |
| UseCase | `internal/usecase/sharing/download_via_share.go` | Download query |
| UseCase | `internal/usecase/sharing/revoke_share_link.go` | Revoke command |
| UseCase | `internal/usecase/sharing/update_share_link.go` | Update command |
| UseCase | `internal/usecase/sharing/list_share_links.go` | List query |
| Interface | `internal/interface/handler/share_handler.go` | HTTP handlers |
| Interface | `internal/interface/dto/share.go` | DTOs |
| Infra | `internal/infrastructure/repository/share_link_repository.go` | DB impl |
| Infra | `internal/infrastructure/repository/share_link_access_repository.go` | DB impl |
| Job | `internal/job/share_link_expiry.go` | Expiry job |
| Job | `internal/job/access_log_anonymize.go` | Anonymize job |

### Changed Files (Frontend)
| Category | File | Change |
|----------|------|--------|
| Route | `src/app/routes/share/$token.tsx` | Public access page |
| Component | `src/components/sharing/share-dialog.tsx` | Share dialog |
| Component | `src/components/sharing/share-link-list.tsx` | Link list |
| Component | `src/components/sharing/password-prompt.tsx` | Password form |
| Component | `src/components/sharing/shared-folder-browser.tsx` | Folder browser |
| Feature | `src/features/sharing/api.ts` | API client functions |
| Feature | `src/features/sharing/hooks.ts` | TanStack Query hooks |

### Migration
```sql
CREATE TABLE share_links (
    id UUID PRIMARY KEY,
    token VARCHAR(64) NOT NULL UNIQUE,
    resource_type VARCHAR(10) NOT NULL,
    resource_id UUID NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    permission VARCHAR(10) NOT NULL,
    password_hash VARCHAR(255),
    expires_at TIMESTAMPTZ,
    max_access_count INT,
    access_count INT NOT NULL DEFAULT 0,
    status VARCHAR(10) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE share_link_accesses (
    id UUID PRIMARY KEY,
    share_link_id UUID NOT NULL REFERENCES share_links(id) ON DELETE CASCADE,
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address VARCHAR(45),
    user_agent TEXT,
    user_id UUID REFERENCES users(id),
    action VARCHAR(10) NOT NULL
);

CREATE INDEX idx_share_links_token ON share_links(token);
CREATE INDEX idx_share_links_resource ON share_links(resource_type, resource_id);
CREATE INDEX idx_share_link_accesses_link_id ON share_link_accesses(share_link_id);
```

### Considerations
- **Performance**: Token lookup indexed; presigned URL generated per-access
- **Security**: Tokens are cryptographically random; passwords bcrypt-hashed
- **Privacy**: Access log IP anonymization after 90 days via background job
- **Public endpoints**: /share/:token/* routes do not require authentication

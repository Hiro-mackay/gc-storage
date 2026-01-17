# GC Storage セキュリティ設計書

## 概要

本ドキュメントでは、GC Storageのセキュリティアーキテクチャ、認証・認可フロー、データ保護について説明します。

---

## 1. セキュリティアーキテクチャ

### 1.1 セキュリティレイヤー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Security Architecture                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  Layer 1: Network Security                                                  │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────────┐             │
│  │  TLS 1.3        │  │  WAF            │  │  DDoS Protection │             │
│  │  Termination    │  │  (Cloudflare)   │  │                  │             │
│  └─────────────────┘  └─────────────────┘  └──────────────────┘             │
├─────────────────────────────────────────────────────────────────────────────┤
│  Layer 2: Application Security                                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────────┐             │
│  │  JWT Auth       │  │  Rate Limit     │  │  Input           │             │
│  │                 │  │                 │  │  Validation      │             │
│  └─────────────────┘  └─────────────────┘  └──────────────────┘             │
├─────────────────────────────────────────────────────────────────────────────┤
│  Layer 3: Authorization (Hybrid PBAC + ReBAC)                               │
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────────┐             │
│  │  PBAC           │  │  ReBAC           │  │  Permission     │             │
│  │  (Permission)   │  │  (Relationship)  │  │  Inheritance    │             │
│  └─────────────────┘  └──────────────────┘  └─────────────────┘             │
├─────────────────────────────────────────────────────────────────────────────┤
│  Layer 4: Data Security                                                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐              │
│  │  Encryption     │  │  Data           │  │  Backup         │              │
│  │  at Rest        │  │  Masking        │  │  Encryption     │              │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘              │
├─────────────────────────────────────────────────────────────────────────────┤
│  Layer 5: Audit & Monitoring                                                │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐              │
│  │  Audit Logs     │  │  SIEM           │  │  Alerting       │              │
│  │                 │  │  Integration    │  │                 │              │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 セキュリティ原則

| 原則 | 実装 |
|------|------|
| 最小権限の原則 | 必要最小限の権限のみ付与 |
| 多層防御 | 複数のセキュリティレイヤー |
| ゼロトラスト | すべてのリクエストを検証 |
| 暗号化 | 転送時・保存時の暗号化 |
| 監査可能性 | 全操作のログ記録 |

---

## 2. 認証 (Authentication)

### 2.1 JWT認証フロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          JWT Authentication Flow                             │
└─────────────────────────────────────────────────────────────────────────────┘

【ログインフロー】

┌────────┐          ┌──────────┐          ┌──────────┐          ┌────────┐
│ Client │          │   API    │          │   DB     │          │ Redis  │
└────┬───┘          └────┬─────┘          └────┬─────┘          └───┬────┘
     │                   │                     │                    │
     │  1. POST /login   │                     │                    │
     │  (email, pass)    │                     │                    │
     │──────────────────▶│                     │                    │
     │                   │  2. Verify User     │                    │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  3. User Data       │                    │
     │                   │◀────────────────────│                    │
     │                   │                     │                    │
     │                   │  4. Generate Tokens │                    │
     │                   │  (Access + Refresh) │                    │
     │                   │                     │                    │
     │                   │  5. Store Refresh   │                    │
     │                   │─────────────────────────────────────────▶│
     │                   │                     │                    │
     │  6. Return Tokens │                     │                    │
     │◀──────────────────│                     │                    │
     │                   │                     │                    │

【APIリクエストフロー】

┌────────┐          ┌──────────┐          ┌──────────┐
│ Client │          │   API    │          │  Redis   │
└────┬───┘          └────┬─────┘          └────┬─────┘
     │                   │                     │
     │  1. Request       │                     │
     │  Authorization:   │                     │
     │  Bearer <token>   │                     │
     │──────────────────▶│                     │
     │                   │                     │
     │                   │  2. Validate JWT    │
     │                   │  (Signature, Exp)   │
     │                   │                     │
     │                   │  3. Check Blacklist │
     │                   │────────────────────▶│
     │                   │                     │
     │                   │  4. Not Blacklisted │
     │                   │◀────────────────────│
     │                   │                     │
     │  5. Response      │                     │
     │◀──────────────────│                     │
     │                   │                     │

【トークンリフレッシュフロー】

┌────────┐          ┌──────────┐          ┌──────────┐          ┌────────┐
│ Client │          │   API    │          │   DB     │          │ Redis  │
└────┬───┘          └────┬─────┘          └────┬─────┘          └───┬────┘
     │                   │                     │                    │
     │  1. POST /refresh │                     │                    │
     │  (refresh_token)  │                     │                    │
     │──────────────────▶│                     │                    │
     │                   │                     │                    │
     │                   │  2. Verify Session  │                    │
     │                   │────────────────────────────────────────▶ │
     │                   │                     │                    │
     │                   │  3. Session Valid   │                    │
     │                   │◀──────────────────────────────────────── │
     │                   │                     │                    │
     │                   │  4. Generate New Tokens                  │
     │                   │                     │                    │
     │                   │  5. Rotate Refresh  │                    │
     │                   │────────────────────────────────────────▶ │
     │                   │                     │                    │
     │  6. New Tokens    │                     │                    │
     │◀──────────────────│                     │                    │
```

### 2.2 JWT構造

```go
// Access Token Claims
type AccessTokenClaims struct {
    jwt.RegisteredClaims
    UserID    string `json:"uid"`
    Email     string `json:"email"`
    SessionID string `json:"sid"`
}

// Access Token 有効期限: 15分
// Refresh Token 有効期限: 7日
```

**Access Token ペイロード例:**
```json
{
  "iss": "gc-storage",
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "aud": ["gc-storage-api"],
  "exp": 1705312200,
  "iat": 1705311300,
  "uid": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "sid": "session-id-123"
}
```

### 2.3 OAuth 2.0 認証フロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          OAuth 2.0 Authorization Code Flow                   │
└─────────────────────────────────────────────────────────────────────────────┘

┌────────┐     ┌──────────┐    ┌───────────────┐     ┌────────────────┐
│ Client │     │   API    │    │ OAuth Provider│     │    Database    │
└────┬───┘     └────┬─────┘    │(Google/GitHub)│     └───────┬────────┘
     │              │          └───────┬───────┘             │
     │  1. GET      │                  │                     │
     │  /oauth/google                  │                     │
     │─────────────▶│                  │                     │
     │              │                  │                     │
     │  2. Redirect │                  │                     │
     │  to Google   │                  │                     │
     │◀─────────────│                  │                     │
     │              │                  │                     │
     │  3. User Login on Google        │                     │
     │───────────────────────────────▶ │                     │
     │              │                  │                     │
     │  4. Callback │                  │                     │
     │  with code   │                  │                     │
     │─────────────▶│                  │                     │
     │              │                  │                     │
     │              │  5. Exchange code│                     │
     │              │  for tokens      │                     │
     │              │─────────────────▶│                     │
     │              │                  │                     │
     │              │  6. Access Token │                     │
     │              │◀─────────────────│                     │
     │              │                  │                     │
     │              │  7. Get User Info│                     │
     │              │─────────────────▶│                     │
     │              │                  │                     │
     │              │  8. User Profile │                     │
     │              │◀─────────────────│                     │
     │              │                  │                     │
     │              │  9. Create/Update User                 │
     │              │───────────────────────────────────────▶│
     │              │                  │                     │
     │              │  10. Generate JWTs                     │
     │              │                  │                     │
     │  11. Return  │                  │                     │
     │  JWTs        │                  │                     │
     │◀─────────────│                  │                     │
```

### 2.4 OAuth設定

```go
// Google OAuth Config
type GoogleOAuthConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
    Scopes       []string
}

// Google Scopes
var googleScopes = []string{
    "openid",
    "email",
    "profile",
}

// GitHub OAuth Config
type GitHubOAuthConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
    Scopes       []string
}

// GitHub Scopes
var githubScopes = []string{
    "user:email",
    "read:user",
}
```

### 2.5 パスワードポリシー

| 項目 | 要件 |
|------|------|
| 最小文字数 | 8文字 |
| 最大文字数 | 256文字 |
| 必須文字種 | 英大文字、英小文字、数字のうち2種以上 |
| 禁止パターン | メールアドレス、連続文字、一般的なパスワード |
| ハッシュアルゴリズム | bcrypt (cost: 12) |

```go
// パスワードバリデーション
func ValidatePassword(password string) error {
    if len(password) < 8 {
        return errors.New("password must be at least 8 characters")
    }
    if len(password) > 256 {
        return errors.New("password must not exceed 256 characters")
    }

    var hasUpper, hasLower, hasDigit bool
    for _, char := range password {
        switch {
        case unicode.IsUpper(char):
            hasUpper = true
        case unicode.IsLower(char):
            hasLower = true
        case unicode.IsDigit(char):
            hasDigit = true
        }
    }

    typesCount := 0
    if hasUpper { typesCount++ }
    if hasLower { typesCount++ }
    if hasDigit { typesCount++ }

    if typesCount < 2 {
        return errors.New("password must contain at least 2 of: uppercase, lowercase, digit")
    }

    return nil
}
```

---

## 3. 認可 (Authorization)

### 3.1 認可モデル概要

**PBAC（Policy-Based）+ ReBAC（Relationship-Based）のハイブリッドモデル**を採用します。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Hybrid Authorization Model                               │
│                      PBAC + ReBAC                                           │
└─────────────────────────────────────────────────────────────────────────────┘

【なぜハイブリッドか？】
┌─────────────────────────────────────────────────────────────────────────────┐
│  このプロジェクトには2種類の認可要件が存在する：                                   │
│                                                                             │
│  1. 「何ができるか」→ Permission（PBAC）                                       │
│     例: file:read, file:delete, permission:grant                            │
│                                                                             │
│  2. 「どう繋がっているか」→ Relationship（ReBAC）                               │
│     例: folder階層、group所属、所有関係                                         │
└─────────────────────────────────────────────────────────────────────────────┘

【設計原則】
┌─────────────────────────────────────────────────────────────────────────────┐
│  • PBAC: 「このPermissionを持っているか？」で最終判定                             │
│  • ReBAC: 「どの関係性を通じてPermissionを得ているか？」を解決                      │
│  • Role = Permission の集合（割り当ての便宜のため）                              │
│  • 関係性の連鎖を辿ってPermissionを収集し、最終的にPermissionで判定                │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 ReBAC: 関係性モデル

このプロジェクトにおける関係性（Relationship）を定義します。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Relationship Types                                  │
└─────────────────────────────────────────────────────────────────────────────┘

【エンティティ間の関係性】

┌─────────────────┬─────────────────┬─────────────────┬───────────────────────┐
│   Subject       │   Relation      │   Object        │   意味                 │
├─────────────────┼─────────────────┼─────────────────┼───────────────────────┤
│ user:alice      │ owner           │ file:doc1       │ aliceはdoc1の所有者    │
│ user:alice      │ member          │ group:eng       │ aliceはengグループの一員│
│ group:eng       │ viewer          │ folder:project  │ engはprojectの閲覧者   │
│ folder:project  │ parent          │ folder:docs     │ projectはdocsの親      │
│ folder:docs     │ parent          │ file:spec.pdf   │ docsはspec.pdfの親     │
└─────────────────┴─────────────────┴─────────────────┴───────────────────────┘

【関係性の種類】

1. 所有関係 (Ownership)
   user ──owner──▶ file/folder
   └── リソースの作成者、完全な制御権を持つ

2. グループ所属 (Membership)
   user ──member──▶ group
   └── グループの一員として、グループの権限を継承

3. 権限付与 (Permission Grant)
   user/group ──viewer/editor/manager──▶ file/folder
   └── 特定のRoleによる権限付与

4. 階層関係 (Hierarchy)
   folder ──parent──▶ folder/file
   └── フォルダ構造による親子関係、権限の継承元
```

### 3.3 関係性グラフと権限解決

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Relationship Graph Example                               │
└─────────────────────────────────────────────────────────────────────────────┘

【シナリオ】
user:alice が file:spec.pdf を閲覧できるか？

【関係性グラフ】

  user:alice ───member───▶ group:engineering
                                  │
                                  │ viewer (Role)
                                  ▼
                           folder:Projects
                                  │
                                  │ parent
                                  ▼
                           folder:ProjectA
                                  │
                                  │ parent
                                  ▼
                            file:spec.pdf    ◀─── 対象リソース

【権限解決の流れ】

1. file:spec.pdf への直接権限をチェック → なし
2. 親フォルダ folder:ProjectA への権限をチェック → なし
3. さらに親 folder:Projects への権限をチェック
   → group:engineering が viewer Role を持つ
4. user:alice は group:engineering の member か？ → Yes
5. viewer Role を Permission に展開 → [file:read, folder:read, ...]
6. file:read を持っているか？ → Yes ✓

【結果】Allow（viewer Role 経由で file:read Permission を保持）
```

### 3.4 Relationship Tuple 設計

Google Zanzibar スタイルの Tuple 形式で関係性を表現します。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Relationship Tuples                                 │
└─────────────────────────────────────────────────────────────────────────────┘

【Tuple 形式】
(subject, relation, object)

【例】
(user:alice, owner, file:report.pdf)
(user:alice, member, group:engineering)
(group:engineering, viewer, folder:projects)
(folder:projects, parent, folder:projectA)
(folder:projectA, parent, file:spec.pdf)

【データベース設計】

-- 関係性テーブル（汎用）
CREATE TABLE relationships (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subject_type    VARCHAR(50) NOT NULL,    -- 'user', 'group', 'folder'
    subject_id      UUID NOT NULL,
    relation        VARCHAR(50) NOT NULL,    -- 'owner', 'member', 'viewer', 'parent'
    object_type     VARCHAR(50) NOT NULL,    -- 'file', 'folder', 'group'
    object_id       UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_relationship UNIQUE (subject_type, subject_id, relation, object_type, object_id)
);

CREATE INDEX idx_relationships_subject ON relationships(subject_type, subject_id);
CREATE INDEX idx_relationships_object ON relationships(object_type, object_id);
CREATE INDEX idx_relationships_relation ON relationships(relation);

-- 高速検索用: 特定オブジェクトへの全関係性
CREATE INDEX idx_relationships_object_lookup
    ON relationships(object_type, object_id, relation);
```

### 3.5 Permission 解決アルゴリズム（ReBAC対応）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                  Permission Resolution with ReBAC                           │
└─────────────────────────────────────────────────────────────────────────────┘

【アルゴリズム概要】

hasPermission(user, resource, requiredPermission):
    1. permissions = collectPermissions(user, resource)
    2. return requiredPermission in permissions

collectPermissions(user, resource):
    permissions = {}

    # Step 1: 所有者チェック
    if isOwner(user, resource):
        return ownerPermissions  # 全権限

    # Step 2: 直接付与された権限
    for grant in getDirectGrants(user, resource):
        permissions.addAll(grant.role.permissions)
        if grant.permission:
            permissions.add(grant.permission)

    # Step 3: グループ経由の権限（ReBAC: membership relation）
    for group in getGroups(user):  # user ──member──▶ group
        for grant in getDirectGrants(group, resource):
            permissions.addAll(grant.role.permissions)

    # Step 4: 階層経由の権限（ReBAC: parent relation）
    for ancestor in getAncestors(resource):  # resource ◀──parent── ancestor
        # ユーザー直接
        for grant in getDirectGrants(user, ancestor):
            permissions.addAll(grant.role.permissions)
        # グループ経由
        for group in getGroups(user):
            for grant in getDirectGrants(group, ancestor):
                permissions.addAll(grant.role.permissions)

    return permissions
```

```go
// ReBAC対応の権限解決サービス
type PermissionResolver struct {
    relationshipRepo repository.RelationshipRepository
}

// HasPermission: ユーザーが特定リソースに対してPermissionを持つか判定
func (r *PermissionResolver) HasPermission(
    ctx context.Context,
    userID uuid.UUID,
    resourceType string,
    resourceID uuid.UUID,
    requiredPermission Permission,
) (bool, error) {
    permissions, err := r.CollectPermissions(ctx, userID, resourceType, resourceID)
    if err != nil {
        return false, err
    }
    return permissions.Has(requiredPermission), nil
}

// CollectPermissions: 関係性を辿って全Permissionを収集
func (r *PermissionResolver) CollectPermissions(
    ctx context.Context,
    userID uuid.UUID,
    resourceType string,
    resourceID uuid.UUID,
) (PermissionSet, error) {
    permissions := NewPermissionSet()

    // Step 1: 所有者チェック (user ──owner──▶ resource)
    isOwner, err := r.relationshipRepo.Exists(ctx, Tuple{
        SubjectType: "user",
        SubjectID:   userID,
        Relation:    "owner",
        ObjectType:  resourceType,
        ObjectID:    resourceID,
    })
    if err != nil {
        return nil, err
    }
    if isOwner {
        permissions.AddAll(RoleOwner.Permissions())
        return permissions, nil
    }

    // Step 2: 直接付与 (user ──{role}──▶ resource)
    directGrants, err := r.relationshipRepo.FindByObject(ctx, resourceType, resourceID, "user", userID)
    if err != nil {
        return nil, err
    }
    for _, grant := range directGrants {
        if role := Role(grant.Relation); role.IsValid() {
            permissions.AddAll(role.Permissions())
        }
    }

    // Step 3: グループ経由 (user ──member──▶ group ──{role}──▶ resource)
    groups, err := r.relationshipRepo.FindRelated(ctx, "user", userID, "member", "group")
    if err != nil {
        return nil, err
    }
    for _, groupID := range groups {
        groupGrants, err := r.relationshipRepo.FindByObject(ctx, resourceType, resourceID, "group", groupID)
        if err != nil {
            continue
        }
        for _, grant := range groupGrants {
            if role := Role(grant.Relation); role.IsValid() {
                permissions.AddAll(role.Permissions())
            }
        }
    }

    // Step 4: 階層経由 (resource ◀──parent── ancestor)
    ancestors, err := r.getAncestors(ctx, resourceType, resourceID)
    if err != nil {
        return nil, err
    }
    for _, ancestor := range ancestors {
        // 祖先への直接権限
        ancestorGrants, err := r.relationshipRepo.FindByObject(ctx, ancestor.Type, ancestor.ID, "user", userID)
        if err != nil {
            continue
        }
        for _, grant := range ancestorGrants {
            if role := Role(grant.Relation); role.IsValid() {
                permissions.AddAll(role.Permissions())
            }
        }

        // 祖先へのグループ経由権限
        for _, groupID := range groups {
            groupGrants, err := r.relationshipRepo.FindByObject(ctx, ancestor.Type, ancestor.ID, "group", groupID)
            if err != nil {
                continue
            }
            for _, grant := range groupGrants {
                if role := Role(grant.Relation); role.IsValid() {
                    permissions.AddAll(role.Permissions())
                }
            }
        }
    }

    return permissions, nil
}

// getAncestors: parent関係を辿って全祖先を取得
func (r *PermissionResolver) getAncestors(
    ctx context.Context,
    resourceType string,
    resourceID uuid.UUID,
) ([]Resource, error) {
    var ancestors []Resource

    currentType := resourceType
    currentID := resourceID

    for {
        // parent関係を探す (ancestor ──parent──▶ current)
        parents, err := r.relationshipRepo.FindRelatedReverse(ctx, currentType, currentID, "parent")
        if err != nil || len(parents) == 0 {
            break
        }

        parent := parents[0] // 単一の親を想定
        ancestors = append(ancestors, parent)
        currentType = parent.Type
        currentID = parent.ID
    }

    return ancestors, nil
}
```

### 3.6 Permission 定義

権限は `{resource}:{action}` 形式で定義します。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Permission Catalog                                  │
└─────────────────────────────────────────────────────────────────────────────┘

【ファイル操作】
┌────────────────────┬───────────────────────────────────────────────────────┐
│    Permission      │                    Description                        │
├────────────────────┼───────────────────────────────────────────────────────┤
│ file:read          │ ファイルの閲覧・ダウンロード                              │
│ file:write         │ ファイルのアップロード・更新                              │
│ file:delete        │ ファイルの削除（ゴミ箱へ移動）                            │
│ file:restore       │ ゴミ箱からの復元                                        │
│ file:permanent_delete │ 完全削除                                           │
│ file:move          │ ファイルの移動                                          │
│ file:rename        │ ファイル名の変更                                        │
│ file:share         │ 共有リンクの作成                                        │
└────────────────────┴───────────────────────────────────────────────────────┘

【フォルダ操作】
┌────────────────────┬───────────────────────────────────────────────────────┐
│    Permission      │                    Description                        │
├────────────────────┼───────────────────────────────────────────────────────┤
│ folder:read        │ フォルダ内容の閲覧                                      │
│ folder:create      │ サブフォルダの作成                                       │
│ folder:delete      │ フォルダの削除                                          │
│ folder:move        │ フォルダの移動                                          │
│ folder:rename      │ フォルダ名の変更                                        │
│ folder:share       │ 共有リンクの作成                                        │
└────────────────────┴───────────────────────────────────────────────────────┘

【権限管理】
┌────────────────────┬───────────────────────────────────────────────────────┐
│    Permission      │                    Description                        │
├────────────────────┼───────────────────────────────────────────────────────┤
│ permission:read    │ リソースの権限設定を閲覧                                 │
│ permission:grant   │ 他ユーザー/グループへの権限付与                           │
│ permission:revoke  │ 権限の取り消し                                          │
└────────────────────┴───────────────────────────────────────────────────────┘

【グループ管理】
┌────────────────────┬───────────────────────────────────────────────────────┐
│    Permission      │                    Description                        │
├────────────────────┼───────────────────────────────────────────────────────┤
│ group:read         │ グループ情報の閲覧                                      │
│ group:update       │ グループ設定の変更                                       │
│ group:delete       │ グループの削除                                          │
│ group:member:read  │ メンバー一覧の閲覧                                       │
│ group:member:add   │ メンバーの追加                                          │
│ group:member:remove│ メンバーの削除                                          │
│ group:member:role  │ メンバーのロール変更                                     │
└────────────────────┴───────────────────────────────────────────────────────┘
```

### 3.7 Role 定義（Permission の集合）

Roleは権限管理の便宜のために使用しますが、**認可判定には使用しません**。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Role Definitions                               │
└─────────────────────────────────────────────────────────────────────────────┘

【リソースロール（ファイル/フォルダ単位）】

┌─────────┬───────────────────────────────────────────────────────────────────┐
│  Role   │                        Permissions                                │
├─────────┼───────────────────────────────────────────────────────────────────┤
│ viewer  │ file:read, folder:read                                           │
├─────────┼───────────────────────────────────────────────────────────────────┤
│ editor  │ viewer +                                                          │
│         │ file:write, file:rename, file:move,                               │
│         │ folder:create, folder:rename, folder:move                         │
├─────────┼───────────────────────────────────────────────────────────────────┤
│ manager │ editor +                                                          │
│         │ file:delete, file:restore, file:share,                            │
│         │ folder:delete, folder:share,                                      │
│         │ permission:read, permission:grant, permission:revoke              │
├─────────┼───────────────────────────────────────────────────────────────────┤
│ owner   │ manager + file:permanent_delete + 譲渡不可の完全制御               │
└─────────┴───────────────────────────────────────────────────────────────────┘

【グループロール】

┌─────────┬───────────────────────────────────────────────────────────────────┐
│  Role   │                        Permissions                                │
├─────────┼───────────────────────────────────────────────────────────────────┤
│ member  │ group:read, group:member:read                                     │
├─────────┼───────────────────────────────────────────────────────────────────┤
│ admin   │ member +                                                          │
│         │ group:update, group:member:add, group:member:remove               │
├─────────┼───────────────────────────────────────────────────────────────────┤
│ owner   │ admin +                                                           │
│         │ group:delete, group:member:role                                   │
└─────────┴───────────────────────────────────────────────────────────────────┘
```

### 3.8 Permission Grant テーブル設計

**Note:** `relationships` テーブル（3.4参照）と併用して使用します。`relationships` テーブルは所有関係・グループ所属・フォルダ階層などの関係性を管理し、`permission_grants` テーブルはRole/Permissionの付与を管理します。

```sql
-- Permission付与テーブル
CREATE TABLE permission_grants (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_type   VARCHAR(50) NOT NULL,      -- 'file', 'folder', 'group'
    resource_id     UUID NOT NULL,
    grantee_type    VARCHAR(20) NOT NULL,      -- 'user', 'group'
    grantee_id      UUID NOT NULL,
    role            VARCHAR(50),               -- 'viewer', 'editor', 'manager' (NULL if direct permission)
    permission      VARCHAR(100),              -- 'file:read' (NULL if role-based)
    granted_by      UUID NOT NULL REFERENCES users(id),
    granted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- roleかpermissionのどちらかは必須
    CONSTRAINT role_or_permission CHECK (role IS NOT NULL OR permission IS NOT NULL),

    -- 同一granteeへの重複付与防止
    CONSTRAINT unique_grant UNIQUE (resource_type, resource_id, grantee_type, grantee_id, role, permission)
);

CREATE INDEX idx_permission_grants_resource ON permission_grants(resource_type, resource_id);
CREATE INDEX idx_permission_grants_grantee ON permission_grants(grantee_type, grantee_id);
```

### 3.9 Policy Enforcement Point (PEP)

**ミドルウェアでのPermissionチェック（Roleではなく常にPermissionで判定）**

```go
// PermissionMiddleware: Policy Enforcement Point
type PermissionMiddleware struct {
    pdp *PolicyDecisionPoint
}

// RequirePermission: 特定のPermissionを要求するミドルウェア
// ★ Roleではなく、Permissionで認可チェックを行う
func (m *PermissionMiddleware) RequirePermission(
    resourceType string,
    requiredPermission Permission,
    resourceIDParam string,
) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            userID := getUserIDFromContext(c)
            resourceID, err := uuid.Parse(c.Param(resourceIDParam))
            if err != nil {
                return apperror.NewValidationError(err)
            }

            // ★ Permission で判定（Role では判定しない）
            hasPermission, err := m.pdp.HasPermission(
                c.Request().Context(),
                userID,
                resourceType,
                resourceID,
                requiredPermission,
            )
            if err != nil {
                return apperror.NewInternalError(err)
            }

            if !hasPermission {
                return apperror.NewForbiddenError(
                    fmt.Sprintf("permission %s required for %s", requiredPermission, resourceType),
                )
            }

            return next(c)
        }
    }
}

// RequireAnyPermission: いずれかのPermissionを持っていればOK
func (m *PermissionMiddleware) RequireAnyPermission(
    resourceType string,
    requiredPermissions []Permission,
    resourceIDParam string,
) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            userID := getUserIDFromContext(c)
            resourceID, err := uuid.Parse(c.Param(resourceIDParam))
            if err != nil {
                return apperror.NewValidationError(err)
            }

            permissions, err := m.pdp.ResolvePermissions(
                c.Request().Context(),
                userID,
                resourceType,
                resourceID,
            )
            if err != nil {
                return apperror.NewInternalError(err)
            }

            for _, required := range requiredPermissions {
                if permissions.Has(required) {
                    return next(c)
                }
            }

            return apperror.NewForbiddenError("insufficient permissions")
        }
    }
}
```

### 3.10 API エンドポイントでの使用例

```go
// ルーティング設定
files := api.Group("/files")

// GET /files/:id - file:read が必要
files.GET("/:id", fileHandler.GetFile,
    permMiddleware.RequirePermission("file", PermFileRead, "id"))

// PUT /files/:id - file:write が必要
files.PUT("/:id", fileHandler.UpdateFile,
    permMiddleware.RequirePermission("file", PermFileWrite, "id"))

// PATCH /files/:id/rename - file:rename が必要
files.PATCH("/:id/rename", fileHandler.RenameFile,
    permMiddleware.RequirePermission("file", PermFileRename, "id"))

// DELETE /files/:id - file:delete が必要
files.DELETE("/:id", fileHandler.DeleteFile,
    permMiddleware.RequirePermission("file", PermFileDelete, "id"))

// DELETE /files/:id/permanent - file:permanent_delete が必要
files.DELETE("/:id/permanent", fileHandler.PermanentDeleteFile,
    permMiddleware.RequirePermission("file", PermFilePermanentDelete, "id"))

// POST /files/:id/share - file:share が必要
files.POST("/:id/share", fileHandler.CreateShareLink,
    permMiddleware.RequirePermission("file", PermFileShare, "id"))

// GET /files/:id/permissions - permission:read が必要
files.GET("/:id/permissions", fileHandler.GetPermissions,
    permMiddleware.RequirePermission("file", PermPermissionRead, "id"))

// POST /files/:id/permissions - permission:grant が必要
files.POST("/:id/permissions", fileHandler.GrantPermission,
    permMiddleware.RequirePermission("file", PermPermissionGrant, "id"))
```

### 3.11 設計のメリット（Hybrid PBAC + ReBAC）

| メリット | 説明 |
|---------|------|
| 柔軟性 | 新しい機能追加時にPermissionを追加するだけで対応可能 |
| 最小権限 | 必要なPermissionのみを付与可能（Roleより細粒度） |
| 監査性 | どのPermissionで許可/拒否されたかが明確 |
| 拡張性 | Roleの変更がPermission定義に影響しない |
| 予測可能性 | 「このPermissionがあれば許可」というシンプルなロジック |
| 階層対応 | ReBAC により親フォルダの権限が自動的に子に継承 |
| グループ管理 | ユーザーをグループに追加するだけで権限が自動適用 |
| 関係性の表現 | 「誰が」「何に」「どう繋がっているか」を明示的にモデル化 |

**PBAC と ReBAC の役割分担:**

| 観点 | PBAC | ReBAC |
|------|------|-------|
| 回答する問い | 「何ができるか」(Permission) | 「どう繋がっているか」(Relationship) |
| データ構造 | Permission 定義、Role → Permission マッピング | Relationship Tuple（subject, relation, object） |
| 評価タイミング | 最終判定 | Permission 収集時の経路探索 |
| 利点 | 判定ロジックがシンプル | 階層・所属関係を自然にモデル化 |

---

## 4. 共有リンク認証

### 4.1 共有リンクフロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Share Link Authentication                           │
└─────────────────────────────────────────────────────────────────────────────┘

【共有リンクアクセス】

┌────────┐          ┌──────────┐          ┌──────────┐          ┌────────┐
│ Client │          │   API    │          │   DB     │          │ MinIO  │
└────┬───┘          └────┬─────┘          └────┬─────┘          └───┬────┘
     │                   │                     │                     │
     │  1. GET /share/{token}                  │                     │
     │──────────────────▶│                     │                     │
     │                   │                     │                     │
     │                   │  2. Lookup Token    │                     │
     │                   │────────────────────▶│                     │
     │                   │                     │                     │
     │                   │  3. Share Link Data │                     │
     │                   │◀────────────────────│                     │
     │                   │                     │                     │
     │                   │  4. Validate:       │                     │
     │                   │  - Expiration       │                     │
     │                   │  - Access Count     │                     │
     │                   │  - Password (if set)│                     │
     │                   │                     │                     │
     │                   │  5. Increment Count │                     │
     │                   │────────────────────▶│                     │
     │                   │                     │                     │
     │                   │  6. Generate        │                     │
     │                   │  Presigned URL      │                     │
     │                   │────────────────────────────────────────▶│
     │                   │                     │                     │
     │  7. Return URL    │                     │                     │
     │◀──────────────────│                     │                     │
```

### 4.2 共有リンク検証

```go
type ShareLinkValidator struct {
    repo repository.ShareLinkRepository
}

func (v *ShareLinkValidator) Validate(ctx context.Context, token string, password *string) (*entity.ShareLink, error) {
    link, err := v.repo.FindByToken(ctx, token)
    if err != nil {
        return nil, apperror.NewNotFoundError("share link")
    }

    // 有効期限チェック
    if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
        return nil, apperror.NewGoneError("share link expired")
    }

    // アクセス回数チェック
    if link.MaxAccessCount != nil && link.AccessCount >= *link.MaxAccessCount {
        return nil, apperror.NewGoneError("share link access limit reached")
    }

    // パスワードチェック
    if link.PasswordHash != nil {
        if password == nil {
            return nil, apperror.NewUnauthorizedError("password required")
        }
        if !bcrypt.CompareHashAndPassword([]byte(*link.PasswordHash), []byte(*password)) {
            return nil, apperror.NewUnauthorizedError("invalid password")
        }
    }

    return link, nil
}
```

---

## 5. データ保護

### 5.1 暗号化

| 対象 | 方式 | 詳細 |
|------|------|------|
| 転送時 | TLS 1.3 | HTTPS強制 |
| パスワード | bcrypt | cost: 12 |
| JWTシークレット | HS256 / RS256 | 256bit以上のシークレット |
| 機密データ | AES-256-GCM | DB内の機密フィールド |
| バックアップ | AES-256 | 暗号化されたバックアップ |

### 5.2 機密データのマスキング

```go
// ログ出力時のマスキング
type SensitiveString string

func (s SensitiveString) MarshalJSON() ([]byte, error) {
    return json.Marshal("[REDACTED]")
}

func (s SensitiveString) String() string {
    return "[REDACTED]"
}

// 使用例
type LogEntry struct {
    UserID   string          `json:"user_id"`
    Email    string          `json:"email"`
    Password SensitiveString `json:"password,omitempty"`
    Token    SensitiveString `json:"token,omitempty"`
}
```

### 5.3 Presigned URLのセキュリティ

```go
// Presigned URL生成ポリシー
const (
    PresignedURLExpiry     = 15 * time.Minute  // 通常の有効期限
    PresignedURLMaxExpiry  = 1 * time.Hour     // マルチパート用の最大有効期限
)

// 追加のセキュリティ対策
// - URLは一度使用後に無効化するオプション
// - IP制限オプション
// - Content-Type制限
```

---

## 6. 入力検証

### 6.1 バリデーションルール

```go
// リクエストバリデーション
type CreateFolderRequest struct {
    Name     string  `json:"name" validate:"required,min=1,max=255,foldername"`
    ParentID *string `json:"parent_id" validate:"omitempty,uuid"`
}

// カスタムバリデータ
func init() {
    validate := validator.New()

    // フォルダ名バリデーション
    validate.RegisterValidation("foldername", func(fl validator.FieldLevel) bool {
        name := fl.Field().String()
        // 禁止文字: / \ : * ? " < > |
        invalidChars := regexp.MustCompile(`[/\\:*?"<>|]`)
        return !invalidChars.MatchString(name)
    })

    // ファイル名バリデーション
    validate.RegisterValidation("filename", func(fl validator.FieldLevel) bool {
        name := fl.Field().String()
        invalidChars := regexp.MustCompile(`[/\\:*?"<>|]`)
        if invalidChars.MatchString(name) {
            return false
        }
        // 隠しファイルや特殊ファイル名のチェック
        if strings.HasPrefix(name, ".") || name == "." || name == ".." {
            return false
        }
        return true
    })
}
```

### 6.2 SQLインジェクション対策

```go
// パラメータ化クエリの使用
func (r *FileRepository) Search(ctx context.Context, query SearchQuery) ([]*entity.File, error) {
    sql := `
        SELECT id, name, size, mime_type
        FROM files
        WHERE owner_id = $1
        AND status = 'active'
    `
    args := []interface{}{query.OwnerID}
    argIndex := 2

    if query.Query != "" {
        sql += fmt.Sprintf(" AND name ILIKE $%d", argIndex)
        args = append(args, "%"+query.Query+"%")
        argIndex++
    }

    if query.MimeType != nil {
        sql += fmt.Sprintf(" AND mime_type = $%d", argIndex)
        args = append(args, *query.MimeType)
        argIndex++
    }

    // 絶対に文字列連結でユーザー入力を埋め込まない
    rows, err := r.db.QueryContext(ctx, sql, args...)
    // ...
}
```

### 6.3 XSS対策

```go
// HTMLエスケープ
func EscapeHTML(s string) string {
    return html.EscapeString(s)
}

// Content-Type設定
func SetSecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        next.ServeHTTP(w, r)
    })
}
```

---

## 7. レート制限

### 7.1 制限設定

```go
// レート制限設定
type RateLimitConfig struct {
    // 認証エンドポイント
    AuthLogin    RateLimit // 10 req/min/IP
    AuthSignup   RateLimit // 5 req/min/IP
    AuthOAuth    RateLimit // 20 req/min/IP

    // API エンドポイント
    APIDefault   RateLimit // 1000 req/min/user
    APIUpload    RateLimit // 100 req/min/user
    APIDownload  RateLimit // 500 req/min/user
    APISearch    RateLimit // 30 req/min/user
}

type RateLimit struct {
    Requests int
    Window   time.Duration
}
```

### 7.2 実装

```go
// Redis ベースのレート制限
type RedisRateLimiter struct {
    client *redis.Client
}

func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit RateLimit) (bool, int, time.Time, error) {
    now := time.Now()
    windowStart := now.Truncate(limit.Window)
    windowKey := fmt.Sprintf("ratelimit:%s:%d", key, windowStart.Unix())

    // Luaスクリプトでアトミックに処理
    script := redis.NewScript(`
        local current = redis.call('INCR', KEYS[1])
        if current == 1 then
            redis.call('EXPIRE', KEYS[1], ARGV[1])
        end
        return current
    `)

    current, err := script.Run(ctx, r.client, []string{windowKey}, int(limit.Window.Seconds())).Int()
    if err != nil {
        return false, 0, time.Time{}, err
    }

    remaining := limit.Requests - current
    if remaining < 0 {
        remaining = 0
    }

    resetAt := windowStart.Add(limit.Window)

    return current <= limit.Requests, remaining, resetAt, nil
}
```

---

## 8. 監査ログ

### 8.1 監査イベント

| カテゴリ | イベント | 記録内容 |
|---------|---------|---------|
| 認証 | auth.login | user_id, ip, user_agent, success |
| 認証 | auth.logout | user_id, session_id |
| 認証 | auth.token_refresh | user_id, session_id |
| ファイル | file.upload | file_id, name, size, user_id |
| ファイル | file.download | file_id, user_id |
| ファイル | file.delete | file_id, user_id |
| フォルダ | folder.create | folder_id, name, parent_id |
| 権限 | permission.grant | resource, grantee, permission |
| 権限 | permission.revoke | resource, grantee |
| 共有 | share.create | resource, token |
| 共有 | share.access | token, ip |

### 8.2 監査ログ実装

```go
// 監査ログサービス
type AuditLogger struct {
    repo repository.AuditLogRepository
}

func (l *AuditLogger) Log(ctx context.Context, event AuditEvent) error {
    log := &entity.AuditLog{
        ID:           uuid.New(),
        UserID:       event.UserID,
        Action:       event.Action,
        ResourceType: event.ResourceType,
        ResourceID:   event.ResourceID,
        Details:      event.Details,
        IPAddress:    event.IPAddress,
        UserAgent:    event.UserAgent,
        CreatedAt:    time.Now(),
    }
    return l.repo.Create(ctx, log)
}

// 使用例
func (uc *DownloadUseCase) Execute(ctx context.Context, input DownloadInput) (*DownloadOutput, error) {
    // ... ダウンロード処理 ...

    // 監査ログ記録
    uc.auditLogger.Log(ctx, AuditEvent{
        UserID:       input.UserID,
        Action:       "file.download",
        ResourceType: "file",
        ResourceID:   input.FileID,
        Details: map[string]interface{}{
            "file_name": file.Name,
            "file_size": file.Size,
        },
        IPAddress: ctx.Value("ip_address").(string),
        UserAgent: ctx.Value("user_agent").(string),
    })

    return output, nil
}
```

---

## 9. セキュリティヘッダー

```go
// セキュリティヘッダーミドルウェア
func SecurityHeaders() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // XSS対策
            c.Response().Header().Set("X-Content-Type-Options", "nosniff")
            c.Response().Header().Set("X-XSS-Protection", "1; mode=block")

            // クリックジャッキング対策
            c.Response().Header().Set("X-Frame-Options", "DENY")

            // HTTPS強制
            c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

            // CSP
            c.Response().Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")

            // Referrer Policy
            c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

            // Permissions Policy
            c.Response().Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

            return next(c)
        }
    }
}
```

---

## 10. セキュリティチェックリスト

### 10.1 開発時チェック

- [ ] 入力バリデーションの実装
- [ ] SQLパラメータ化クエリの使用
- [ ] HTMLエスケープの実装
- [ ] 認証・認可のテスト
- [ ] 機密データのログ出力除外
- [ ] エラーメッセージの情報漏洩チェック

### 10.2 デプロイ時チェック

- [ ] TLS証明書の設定
- [ ] 環境変数からのシークレット読み込み
- [ ] ファイアウォールルールの設定
- [ ] レート制限の有効化
- [ ] 監査ログの有効化
- [ ] バックアップの暗号化

### 10.3 定期チェック

- [ ] 依存パッケージの脆弱性スキャン
- [ ] アクセスログの異常検知
- [ ] 失敗したログイン試行の監視
- [ ] 証明書の有効期限確認
- [ ] シークレットのローテーション

---

## 関連ドキュメント

- [アーキテクチャ設計](./ARCHITECTURE.md)
- [API設計](./API.md)
- [バックエンド設計](./BACKEND.md)
- [インフラストラクチャ設計](./INFRASTRUCTURE.md)

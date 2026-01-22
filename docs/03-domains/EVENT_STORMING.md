# GC Storage イベントストーミング

## 概要

本ドキュメントは、GC Storageのドメインモデルを明らかにするためのイベントストーミング結果をまとめたものです。
ドメインイベント、コマンド、アクター、集約、境界づけられたコンテキストを定義し、ドメイン間の関係性を明確にします。

---

## 1. アクター（Actor）

システムを利用する主体を定義します。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Actors                                          │
└─────────────────────────────────────────────────────────────────────────────┘

┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│     Guest        │  │   Authenticated  │  │Group Contributor │
│   （ゲスト）       │  │      User        │  │（グループ貢献者）  │
│                  │  │  （認証済みユーザー）│  │                  │
│ • 共有リンクアクセス │  │ • ファイル操作    │  │ • メンバー招待    │
│ • サインアップ     │  │ • フォルダ管理    │  │ • 権限付与        │
│ • ログイン        │  │ • 共有リンク作成  │  │ (Contributor以下) │
└──────────────────┘  └──────────────────┘  └──────────────────┘

┌──────────────────┐  ┌──────────────────┐
│   Group Owner    │  │     System       │
│（グループオーナー） │  │   （システム）     │
│                  │  │                  │
│ • グループ削除    │  │ • バッチ処理      │
│ • オーナー譲渡    │  │ • ゴミ箱自動削除  │
│ • メンバー削除    │  │ • セッション管理  │
│ • ロール変更      │  │                  │
│ • 設定変更        │  │                  │
└──────────────────┘  └──────────────────┘
```

| アクター | 説明 | 主な操作 |
|---------|------|---------|
| Guest | 未認証のユーザー | 共有リンク経由のアクセス、アカウント作成、ログイン |
| Authenticated User | ログイン済みユーザー | ファイル・フォルダ操作、共有、グループ参加 |
| Group Contributor | グループのContributorロールを持つユーザー | メンバー招待（Contributor以下）、権限付与 |
| Group Owner | グループのOwnerロールを持つユーザー | グループ削除、オーナー譲渡、ロール変更、メンバー削除、設定変更 |
| System | 自動処理を行うシステム | スケジュールタスク、セッション失効、ゴミ箱クリーンアップ |

---

## 2. ドメインイベント（Domain Event）

過去形で記述される、システム内で発生した重要な出来事です。

### 2.1 Identity Context（認証・ユーザー管理）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Identity Context - Domain Events                          │
└─────────────────────────────────────────────────────────────────────────────┘

【ユーザー登録・認証】
┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ UserRegistered      │  │ UserLoggedIn        │  │ UserLoggedOut       │
│                     │  │                     │  │                     │
│ ユーザーが登録された   │  │ ユーザーがログインした │  │ ユーザーがログアウトした│
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ OAuthAccountLinked  │  │ PasswordChanged     │  │ PasswordResetRequested│
│                     │  │                     │  │                     │
│ OAuth連携が追加された │  │ パスワードが変更された │  │ パスワードリセットが要求│
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ PasswordReset       │  │ EmailVerified       │  │ SessionCreated      │
│                     │  │                     │  │                     │
│ パスワードがリセット   │  │ メールが確認された   │  │ セッションが作成された │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ SessionRevoked      │  │ ProfileUpdated      │  │ UserDeactivated     │
│                     │  │                     │  │                     │
│ セッションが失効した   │  │ プロフィールが更新    │  │ ユーザーが無効化      │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘
```

| イベント | トリガー | 影響 |
|---------|---------|------|
| UserRegistered | ユーザー登録完了 | ルートフォルダ作成、確認メール送信 |
| UserLoggedIn | 認証成功 | セッション作成、監査ログ記録 |
| UserLoggedOut | ログアウト実行 | セッション失効 |
| OAuthAccountLinked | OAuth認証成功 | 外部アカウント紐付け |
| PasswordChanged | パスワード変更 | 全セッション失効（オプション） |
| PasswordResetRequested | リセット要求 | リセットトークン発行、メール送信 |
| PasswordReset | リセット完了 | 新パスワード設定 |
| EmailVerified | メール確認完了 | アカウント有効化 |
| SessionCreated | ログイン成功 | JWT発行 |
| SessionRevoked | ログアウト/失効 | トークン無効化 |
| ProfileUpdated | プロフィール変更 | ユーザー情報更新 |
| UserDeactivated | アカウント無効化 | アクセス禁止 |

### 2.2 Storage Context（ファイル・フォルダ管理）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Storage Context - Domain Events                           │
└─────────────────────────────────────────────────────────────────────────────┘

【ファイル操作】
┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ FileUploadStarted   │  │ FileUploaded        │  │ FileDownloaded      │
│                     │  │                     │  │                     │
│ ファイルアップロード   │  │ ファイルがアップロード │  │ ファイルがダウンロード │
│ が開始された         │  │ された               │  │ された               │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ FileRenamed         │  │ FileMoved           │  │ FileCopied          │
│                     │  │                     │  │                     │
│ ファイル名が変更された │  │ ファイルが移動された  │  │ ファイルがコピーされた │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ FileTrashed         │  │ FileRestored        │  │ FilePermanentlyDeleted│
│                     │  │                     │  │                     │
│ ファイルがゴミ箱へ    │  │ ファイルが復元された  │  │ ファイルが完全削除    │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐
│ FileVersionCreated  │  │ FileVersionRestored │
│                     │  │                     │
│ 新バージョンが作成    │  │ 旧バージョンに復元   │
└─────────────────────┘  └─────────────────────┘

【フォルダ操作】
┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ FolderCreated       │  │ FolderRenamed       │  │ FolderMoved         │
│                     │  │                     │  │                     │
│ フォルダが作成された  │  │ フォルダ名が変更     │  │ フォルダが移動された  │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ FolderTrashed       │  │ FolderRestored      │  │ FolderPermanentlyDeleted│
│                     │  │                     │  │                     │
│ フォルダがゴミ箱へ    │  │ フォルダが復元された  │  │ フォルダが完全削除    │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘
```

| イベント | トリガー | 影響 |
|---------|---------|------|
| FileUploadStarted | アップロード開始 | Presigned URL発行、状態: pending |
| FileUploaded | アップロード完了 | 状態: active、メタデータ保存 |
| FileDownloaded | ダウンロード実行 | 監査ログ、アクセスカウント |
| FileRenamed | 名前変更 | メタデータ更新 |
| FileMoved | 移動実行 | 親フォルダ変更、権限再計算 |
| FileCopied | コピー実行 | 新ファイル作成 |
| FileTrashed | ゴミ箱移動 | 状態: trashed |
| FileRestored | 復元実行 | 状態: active |
| FilePermanentlyDeleted | 完全削除 | ストレージ削除、メタデータ削除 |
| FileVersionCreated | 上書きアップロード | 旧バージョン保存 |
| FileVersionRestored | バージョン復元 | 指定バージョンを最新に |
| FolderCreated | フォルダ作成 | 階層構造更新 |
| FolderRenamed | 名前変更 | メタデータ更新 |
| FolderMoved | 移動実行 | 階層構造更新、権限再計算 |
| FolderTrashed | ゴミ箱移動 | 子要素も連動 |
| FolderRestored | 復元実行 | 子要素も連動 |
| FolderPermanentlyDeleted | 完全削除 | 子要素も連動削除 |

### 2.3 Authorization Context（権限管理）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                  Authorization Context - Domain Events                       │
└─────────────────────────────────────────────────────────────────────────────┘

【権限操作】
┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ PermissionGranted   │  │ PermissionRevoked   │  │ PermissionInherited │
│                     │  │                     │  │                     │
│ 権限が付与された      │  │ 権限が取り消された    │  │ 権限が継承された      │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐
│ RoleAssigned        │  │ RoleRemoved         │
│                     │  │                     │
│ ロールが割り当てられた │  │ ロールが削除された    │
└─────────────────────┘  └─────────────────────┘

【所有権】
┌─────────────────────┐  ┌─────────────────────┐
│ OwnershipTransferred│  │ OwnershipClaimed    │
│                     │  │                     │
│ 所有権が譲渡された    │  │ 所有権が主張された    │
└─────────────────────┘  └─────────────────────┘
```

| イベント | トリガー | 影響 |
|---------|---------|------|
| PermissionGranted | 権限付与 | アクセス許可追加 |
| PermissionRevoked | 権限取消 | アクセス許可削除 |
| PermissionInherited | 親からの継承 | 子要素へ権限適用 |
| RoleAssigned | ロール付与 | 権限セット適用 |
| RoleRemoved | ロール削除 | 権限セット削除 |
| OwnershipTransferred | 所有権譲渡 | 新オーナー設定 |
| OwnershipClaimed | 所有権主張 | オーナーなしリソースの所有 |

### 2.4 Sharing Context（共有機能）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     Sharing Context - Domain Events                          │
└─────────────────────────────────────────────────────────────────────────────┘

【共有リンク】
┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ ShareLinkCreated    │  │ ShareLinkAccessed   │  │ ShareLinkRevoked    │
│                     │  │                     │  │                     │
│ 共有リンクが作成された │  │ 共有リンクにアクセス  │  │ 共有リンクが無効化    │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ ShareLinkExpired    │  │ ShareLinkUpdated    │  │ ShareLinkPasswordSet│
│                     │  │                     │  │                     │
│ 共有リンクが期限切れ  │  │ 共有リンクが更新     │  │ パスワードが設定      │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘
```

| イベント | トリガー | 影響 |
|---------|---------|------|
| ShareLinkCreated | 共有リンク作成 | トークン発行、アクセス許可 |
| ShareLinkAccessed | リンクアクセス | アクセスカウント更新、監査ログ |
| ShareLinkRevoked | リンク無効化 | アクセス禁止 |
| ShareLinkExpired | 有効期限到達 | アクセス禁止 |
| ShareLinkUpdated | 設定変更 | 期限・パスワード等更新 |
| ShareLinkPasswordSet | パスワード設定 | 認証必須化 |

### 2.5 Collaboration Context（グループ・協同作業）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                  Collaboration Context - Domain Events                       │
└─────────────────────────────────────────────────────────────────────────────┘

【グループ管理】
┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ GroupCreated        │  │ GroupUpdated        │  │ GroupDeleted        │
│                     │  │                     │  │                     │
│ グループが作成された  │  │ グループが更新された  │  │ グループが削除された  │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

【メンバー管理】
┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ MemberInvited       │  │ MemberJoined        │  │ MemberLeft          │
│                     │  │                     │  │                     │
│ メンバーが招待された  │  │ メンバーが参加した    │  │ メンバーが脱退した    │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│ MemberRemoved       │  │ MemberRoleChanged   │  │ InvitationAccepted  │
│                     │  │                     │  │                     │
│ メンバーが削除された  │  │ メンバーロールが変更  │  │ 招待が承諾された      │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐
│ InvitationDeclined  │  │ InvitationExpired   │
│                     │  │                     │
│ 招待が辞退された      │  │ 招待が期限切れ        │
└─────────────────────┘  └─────────────────────┘

【所有権譲渡】
┌─────────────────────┐
│ GroupOwnershipTransferred│
│                     │
│ グループ所有権が譲渡   │
└─────────────────────┘
```

| イベント | トリガー | 影響 |
|---------|---------|------|
| GroupCreated | グループ作成 | オーナー設定、グループフォルダ作成 |
| GroupUpdated | 設定変更 | 名前・説明更新 |
| GroupDeleted | グループ削除 | メンバー解放、リソース処理 |
| MemberInvited | 招待送信 | 招待トークン発行、通知 |
| MemberJoined | 参加完了 | メンバーシップ作成 |
| MemberLeft | 自主脱退 | メンバーシップ削除 |
| MemberRemoved | 強制削除 | メンバーシップ削除 |
| MemberRoleChanged | ロール変更 | 権限更新 |
| InvitationAccepted | 招待承諾 | MemberJoined発火 |
| InvitationDeclined | 招待辞退 | 招待無効化 |
| InvitationExpired | 期限切れ | 招待無効化 |
| GroupOwnershipTransferred | オーナー譲渡 | 新オーナー設定 |

---

## 3. コマンド（Command）

イベントを発生させるためのアクションです。

### 3.1 Identity Context

| コマンド | アクター | 発生イベント |
|---------|---------|-------------|
| RegisterUser | Guest | UserRegistered |
| LoginWithCredentials | Guest | UserLoggedIn / LoginFailed |
| LoginWithOAuth | Guest | UserLoggedIn, OAuthAccountLinked |
| Logout | Authenticated User | UserLoggedOut |
| ChangePassword | Authenticated User | PasswordChanged |
| RequestPasswordReset | Guest | PasswordResetRequested |
| ResetPassword | Guest | PasswordReset |
| VerifyEmail | Guest | EmailVerified |
| UpdateProfile | Authenticated User | ProfileUpdated |
| DeactivateAccount | Authenticated User | UserDeactivated |
| LinkOAuthAccount | Authenticated User | OAuthAccountLinked |
| RevokeSession | Authenticated User | SessionRevoked |

### 3.2 Storage Context

| コマンド | アクター | 発生イベント |
|---------|---------|-------------|
| InitiateUpload | Authenticated User | FileUploadStarted |
| CompleteUpload | Authenticated User | FileUploaded |
| DownloadFile | Authenticated User | FileDownloaded |
| RenameFile | Authenticated User | FileRenamed |
| MoveFile | Authenticated User | FileMoved |
| CopyFile | Authenticated User | FileCopied |
| TrashFile | Authenticated User | FileTrashed |
| RestoreFile | Authenticated User | FileRestored |
| PermanentlyDeleteFile | Authenticated User | FilePermanentlyDeleted |
| UploadNewVersion | Authenticated User | FileVersionCreated |
| RestoreVersion | Authenticated User | FileVersionRestored |
| CreateFolder | Authenticated User | FolderCreated |
| RenameFolder | Authenticated User | FolderRenamed |
| MoveFolder | Authenticated User | FolderMoved |
| TrashFolder | Authenticated User | FolderTrashed |
| RestoreFolder | Authenticated User | FolderRestored |
| PermanentlyDeleteFolder | Authenticated User | FolderPermanentlyDeleted |
| CleanupTrash | System | FilePermanentlyDeleted, FolderPermanentlyDeleted |

### 3.3 Authorization Context

| コマンド | アクター | 発生イベント |
|---------|---------|-------------|
| GrantPermission | Contributor/Content Manager/Owner | PermissionGranted |
| RevokePermission | Contributor/Content Manager/Owner | PermissionRevoked |
| AssignRole | Contributor/Content Manager/Owner | RoleAssigned |
| RemoveRole | Contributor/Content Manager/Owner | RoleRemoved |
| TransferOwnership | Owner | OwnershipTransferred |

### 3.4 Sharing Context

| コマンド | アクター | 発生イベント |
|---------|---------|-------------|
| CreateShareLink | Authenticated User | ShareLinkCreated |
| AccessShareLink | Guest/Authenticated User | ShareLinkAccessed |
| RevokeShareLink | Authenticated User | ShareLinkRevoked |
| UpdateShareLink | Authenticated User | ShareLinkUpdated |
| SetShareLinkPassword | Authenticated User | ShareLinkPasswordSet |
| ExpireShareLinks | System | ShareLinkExpired |

### 3.5 Collaboration Context

| コマンド | アクター | 発生イベント |
|---------|---------|-------------|
| CreateGroup | Authenticated User | GroupCreated |
| UpdateGroup | Group Owner | GroupUpdated |
| DeleteGroup | Group Owner | GroupDeleted |
| InviteMember | Group Contributor/Owner | MemberInvited |
| AcceptInvitation | Authenticated User | InvitationAccepted, MemberJoined |
| DeclineInvitation | Authenticated User | InvitationDeclined |
| LeaveGroup | Authenticated User | MemberLeft |
| RemoveMember | Group Owner | MemberRemoved |
| ChangeMemberRole | Group Owner | MemberRoleChanged |
| TransferGroupOwnership | Group Owner | GroupOwnershipTransferred |
| ExpireInvitations | System | InvitationExpired |

---

## 4. 集約（Aggregate）

トランザクション整合性の境界を定義します。

### 4.1 Identity Context

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              User Aggregate                                  │
└─────────────────────────────────────────────────────────────────────────────┘

                        ┌──────────────────┐
                        │      User        │ ◀─── Aggregate Root
                        │                  │
                        │ • id: UUID       │
                        │ • email: Email   │
                        │ • name: string   │
                        │ • password_hash  │
                        │ • status         │
                        │ • created_at     │
                        │ • updated_at     │
                        └────────┬─────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
              ▼                  ▼                  ▼
    ┌──────────────────┐ ┌──────────────┐ ┌──────────────────┐
    │  OAuthAccount    │ │   Session    │ │   UserProfile    │
    │                  │ │              │ │                  │
    │ • provider       │ │ • id         │ │ • avatar_url     │
    │ • provider_id    │ │ • user_id    │ │ • display_name   │
    │ • access_token   │ │ • token      │ │ • bio            │
    │ • refresh_token  │ │ • expires_at │ │ • settings       │
    └──────────────────┘ └──────────────┘ └──────────────────┘
```

**不変条件:**
- emailは一意でなければならない
- 1ユーザーにつき同一providerのOAuthAccountは1つのみ
- パスワード認証またはOAuth認証の少なくとも1つは必須

### 4.2 Storage Context

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              File Aggregate                                  │
└─────────────────────────────────────────────────────────────────────────────┘

                        ┌──────────────────┐
                        │      File        │ ◀─── Aggregate Root
                        │                  │
                        │ • id: UUID       │
                        │ • name: string   │
                        │ • folder_id      │
                        │ • owner_id       │
                        │ • created_by     │
                        │ • size: int64    │
                        │ • mime_type      │
                        │ • storage_key    │
                        │ • status         │
                        │ • current_version│
                        │ • created_at     │
                        │ • updated_at     │
                        └────────┬─────────┘
                                 │
                    ┌────────────┴────────────┐
                    │                         │
                    ▼                         ▼
          ┌──────────────────┐     ┌──────────────────┐
          │   FileVersion    │     │   FileMetadata   │
          │                  │     │                  │
          │ • version_num    │     │ • checksum_md5   │
          │ • size           │     │ • checksum_sha256│
          │ • storage_key    │     │ • width          │
          │ • created_by     │     │ • height         │
          │ • created_at     │     │ • duration       │
          └──────────────────┘     │ • custom_props   │
                                   └──────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                             Folder Aggregate                                 │
└─────────────────────────────────────────────────────────────────────────────┘

                        ┌──────────────────┐
                        │     Folder       │ ◀─── Aggregate Root
                        │                  │
                        │ • id: UUID       │
                        │ • name: string   │
                        │ • parent_id      │
                        │ • owner_id       │
                        │ • created_by     │
                        │ • depth          │
                        │ • status         │
                        │ • created_at     │
                        │ • updated_at     │
                        └──────────────────┘
```

**File不変条件:**
- storage_keyは一意でなければならない
- バージョン番号は連続していなければならない
- ファイル名は同一フォルダ内で一意
- ステータス遷移: pending → active → trashed → deleted

**Folder不変条件:**
- 自身を子孫に持つフォルダへの移動は不可（循環参照防止）
- ルートフォルダのparent_idはNULL
- pathは正規化された形式で保持
- フォルダ名は同一親フォルダ内で一意

### 4.3 Authorization Context

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Permission Grant Aggregate                          │
└─────────────────────────────────────────────────────────────────────────────┘

                     ┌──────────────────────┐
                     │  PermissionGrant     │ ◀─── Aggregate Root
                     │                      │
                     │ • id: UUID           │
                     │ • resource_type      │
                     │ • resource_id        │
                     │ • grantee_type       │ (user/group)
                     │ • grantee_id         │
                     │ • role               │ (viewer/contributor/content_manager)
                     │ • granted_by         │
                     │ • granted_at         │
                     └──────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                          Relationship Aggregate                              │
└─────────────────────────────────────────────────────────────────────────────┘

                     ┌──────────────────────┐
                     │    Relationship      │ ◀─── Aggregate Root
                     │                      │
                     │ • id: UUID           │
                     │ • subject_type       │
                     │ • subject_id         │
                     │ • relation           │ (owner/member/parent/viewer/contributor/content_manager)
                     │ • object_type        │
                     │ • object_id          │
                     │ • created_at         │
                     └──────────────────────┘
```

**不変条件:**
- 同一(subject, relation, object)の組み合わせは一意
- ownerは1リソースにつき1つのみ
- グループ所有のリソースにはユーザーownerは設定不可

### 4.4 Sharing Context

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            ShareLink Aggregate                               │
└─────────────────────────────────────────────────────────────────────────────┘

                        ┌──────────────────┐
                        │    ShareLink     │ ◀─── Aggregate Root
                        │                  │
                        │ • id: UUID       │
                        │ • token          │
                        │ • resource_type  │
                        │ • resource_id    │
                        │ • created_by     │
                        │ • permission     │ (read/write)
                        │ • password_hash  │
                        │ • expires_at     │
                        │ • max_access_cnt │
                        │ • access_count   │
                        │ • status         │
                        │ • created_at     │
                        │ • updated_at     │
                        └──────────────────┘
```

**不変条件:**
- tokenは一意でなければならない
- access_countはmax_access_cntを超えられない
- expires_at到達後はアクセス不可
- statusがrevokedの場合はアクセス不可

### 4.5 Collaboration Context

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Group Aggregate                                 │
└─────────────────────────────────────────────────────────────────────────────┘

                        ┌──────────────────┐
                        │     Group        │ ◀─── Aggregate Root
                        │                  │
                        │ • id: UUID       │
                        │ • name: string   │
                        │ • description    │
                        │ • owner_id       │
                        │ • status         │
                        │ • created_at     │
                        │ • updated_at     │
                        └────────┬─────────┘
                                 │
              ┌──────────────────┴──────────────────┐
              │                                     │
              ▼                                     ▼
    ┌──────────────────┐               ┌──────────────────┐
    │   Membership     │               │   Invitation     │
    │                  │               │                  │
    │ • group_id       │               │ • group_id       │
    │ • user_id        │               │ • email          │
    │ • role           │               │ • token          │
    │ • joined_at      │               │ • role           │
    │                  │               │ • invited_by     │
    │                  │               │ • expires_at     │
    │                  │               │ • status         │
    └──────────────────┘               └──────────────────┘
```

**不変条件:**
- グループには必ず1人のownerが存在する
- ownerはグループから脱退できない（譲渡が必要）
- 同一ユーザーは同一グループに1つのMembershipのみ
- 招待トークンは一意でなければならない

---

## 5. 境界づけられたコンテキスト（Bounded Context）

### 5.1 コンテキスト定義

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Bounded Contexts                                    │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│           Identity Context              │
│                                         │
│  責務:                                   │
│  • ユーザー登録・認証                      │
│  • セッション管理                          │
│  • OAuth連携                             │
│  • パスワード管理                          │
│                                         │
│  集約: User                              │
│  公開言語: UserID, Email                  │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│           Storage Context               │
│                                         │
│  責務:                                   │
│  • ファイルのアップロード・ダウンロード       │
│  • フォルダ階層管理                        │
│  • バージョン管理                          │
│  • ゴミ箱管理                             │
│                                         │
│  集約: File, Folder                      │
│  公開言語: FileID, FolderID, StorageKey   │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│         Authorization Context           │
│                                         │
│  責務:                                   │
│  • 権限の付与・取り消し                    │
│  • ロール管理                             │
│  • 権限継承の解決                          │
│  • アクセス判定                           │
│                                         │
│  集約: PermissionGrant, Relationship     │
│  公開言語: Permission, Role              │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│           Sharing Context               │
│                                         │
│  責務:                                   │
│  • 共有リンクの作成・管理                   │
│  • アクセス検証                           │
│  • 有効期限管理                           │
│                                         │
│  集約: ShareLink                         │
│  公開言語: ShareToken                    │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│        Collaboration Context            │
│                                         │
│  責務:                                   │
│  • グループの作成・管理                    │
│  • メンバー招待・管理                      │
│  • グループ内ロール管理                    │
│                                         │
│  集約: Group                             │
│  公開言語: GroupID, MembershipID          │
└─────────────────────────────────────────┘
```

### 5.2 コンテキストマップ

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Context Map                                       │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌─────────────────┐
                              │    Identity     │
                              │    Context      │
                              └────────┬────────┘
                                       │
                                       │ UserID
                                       │ (Published Language)
                    ┌──────────────────┼──────────────────┐
                    │                  │                  │
                    ▼                  ▼                  ▼
          ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
          │   Storage       │ │ Authorization   │ │ Collaboration   │
          │   Context       │ │   Context       │ │   Context       │
          └────────┬────────┘ └────────┬────────┘ └────────┬────────┘
                   │                   │                   │
                   │ FileID, FolderID  │ Permission        │ GroupID
                   │                   │                   │
                   └───────────────────┼───────────────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │    Sharing      │
                              │    Context      │
                              └─────────────────┘

【関係性の種類】

Identity ──U/D──▶ Storage
    Identity（上流）がUserIDを提供
    Storage（下流）がファイル所有者として使用

Identity ──U/D──▶ Authorization
    Identity（上流）がUserIDを提供
    Authorization（下流）が権限付与の主体として使用

Identity ──U/D──▶ Collaboration
    Identity（上流）がUserIDを提供
    Collaboration（下流）がグループメンバーとして使用

Storage ──U/D──▶ Authorization
    Storage（上流）がFileID, FolderIDを提供
    Authorization（下流）が権限対象として使用

Storage ──U/D──▶ Sharing
    Storage（上流）がFileID, FolderIDを提供
    Sharing（下流）が共有対象として使用

Authorization ──U/D──▶ Sharing
    Authorization（上流）がPermission判定を提供
    Sharing（下流）がリンク経由のアクセス検証に使用

Collaboration ──U/D──▶ Authorization
    Collaboration（上流）がGroupID, Membershipを提供
    Authorization（下流）がグループ経由の権限解決に使用

凡例:
U/D = Upstream/Downstream（上流/下流）
──▶ = 依存の方向
```

### 5.3 コンテキスト間の統合パターン

| 統合パターン | 使用箇所 | 説明 |
|-------------|---------|------|
| Published Language | Identity → 全コンテキスト | UserID, Emailを共通言語として公開 |
| Shared Kernel | Storage ↔ Authorization | FileID, FolderIDを共有 |
| Customer/Supplier | Storage → Sharing | StorageがShareに必要なリソース情報を提供 |
| Anti-Corruption Layer | 外部OAuth → Identity | 外部プロバイダーの形式を内部形式に変換 |

---

## 6. ドメインサービス（Domain Service）

集約をまたぐ操作を担当するサービスです。

### 6.1 Storage Context

```go
// FileUploadService: ファイルアップロードを調整
type FileUploadService interface {
    // InitiateUpload: アップロードセッション開始
    InitiateUpload(ctx context.Context, cmd InitiateUploadCommand) (*UploadSession, error)

    // CompleteUpload: アップロード完了処理
    CompleteUpload(ctx context.Context, sessionID string) (*File, error)
}

// FolderHierarchyService: フォルダ階層操作を調整
type FolderHierarchyService interface {
    // MoveFolder: 循環参照チェックを含むフォルダ移動
    MoveFolder(ctx context.Context, folderID, targetParentID uuid.UUID) error

    // GetAncestors: 祖先フォルダを取得
    GetAncestors(ctx context.Context, folderID uuid.UUID) ([]*Folder, error)
}
```

### 6.2 Authorization Context

```go
// PermissionResolver: 権限解決サービス
type PermissionResolver interface {
    // HasPermission: 権限判定
    HasPermission(ctx context.Context, userID uuid.UUID, resourceType string, resourceID uuid.UUID, permission Permission) (bool, error)

    // CollectPermissions: 全権限収集
    CollectPermissions(ctx context.Context, userID uuid.UUID, resourceType string, resourceID uuid.UUID) (PermissionSet, error)
}

// PermissionInheritanceService: 権限継承を解決
type PermissionInheritanceService interface {
    // PropagatePermission: 権限を子要素に伝播
    PropagatePermission(ctx context.Context, resourceType string, resourceID uuid.UUID) error
}
```

### 6.3 Collaboration Context

```go
// GroupMembershipService: グループメンバーシップを管理
type GroupMembershipService interface {
    // InviteMember: メンバー招待
    InviteMember(ctx context.Context, groupID uuid.UUID, email string, role GroupRole) (*Invitation, error)

    // AcceptInvitation: 招待承諾
    AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*Membership, error)
}
```

---

## 7. ユビキタス言語（Ubiquitous Language）

各コンテキストで使用する用語を統一します。

### 7.1 Identity Context

| 用語 | 定義 |
|------|------|
| User | システムに登録されたユーザー |
| Session | ユーザーの認証状態を表すセッション |
| Credentials | 認証情報（メール/パスワード） |
| OAuthAccount | 外部認証プロバイダーとの連携情報 |
| EmailVerification | メールアドレス確認プロセス |

### 7.2 Storage Context

| 用語 | 定義 |
|------|------|
| File | アップロードされたファイルのメタデータ |
| FileVersion | ファイルの特定バージョン |
| Folder | ファイルを整理するためのコンテナ |
| RootFolder | ユーザーまたはグループの最上位フォルダ |
| Trash | 論理削除されたアイテムの一時保管場所 |
| StorageKey | MinIO内のオブジェクトを識別するキー |

### 7.3 Authorization Context

| 用語 | 定義 |
|------|------|
| Permission | 特定のリソースに対する操作権限（file:read等） |
| Role | 権限のセット（viewer, contributor, content_manager） |
| PermissionGrant | 権限付与の記録 |
| Relationship | エンティティ間の関係性（owner, member, parent） |
| Inheritance | 親フォルダからの権限継承 |

### 7.4 Sharing Context

| 用語 | 定義 |
|------|------|
| ShareLink | リソースへのアクセスを許可する共有リンク |
| ShareToken | 共有リンクを識別する一意のトークン |
| AccessLimit | 共有リンクの最大アクセス回数 |
| Expiration | 共有リンクの有効期限 |

### 7.5 Collaboration Context

| 用語 | 定義 |
|------|------|
| Group | ユーザーの集まり |
| Membership | ユーザーのグループへの所属 |
| GroupRole | グループ内での役割（viewer, contributor, owner） |
| Invitation | グループへの招待 |
| GroupOwner | グループの所有者（1名のみ） |

---

## 8. PRD機能とイベントのマッピング

PRD.mdで定義された各機能がどのイベントでカバーされるかを確認します。

### 8.1 ファイル管理機能

| PRD機能 | 対応イベント |
|---------|-------------|
| ファイルアップロード | FileUploadStarted, FileUploaded |
| ファイルダウンロード | FileDownloaded |
| マルチパートアップロード | FileUploadStarted (multipart), FileUploaded |
| ファイル削除 | FileTrashed, FilePermanentlyDeleted |
| ファイル復元 | FileRestored |
| ファイル名変更 | FileRenamed |
| ファイル移動 | FileMoved |
| バージョン管理 | FileVersionCreated, FileVersionRestored |

### 8.2 フォルダ管理機能

| PRD機能 | 対応イベント |
|---------|-------------|
| フォルダ作成 | FolderCreated |
| フォルダ削除 | FolderTrashed, FolderPermanentlyDeleted |
| フォルダ移動 | FolderMoved |
| フォルダ名変更 | FolderRenamed |
| フォルダ復元 | FolderRestored |

### 8.3 ユーザー・認証機能

| PRD機能 | 対応イベント |
|---------|-------------|
| サインアップ | UserRegistered, EmailVerified |
| ログイン | UserLoggedIn, SessionCreated |
| ログアウト | UserLoggedOut, SessionRevoked |
| OAuth連携 | OAuthAccountLinked |
| パスワードリセット | PasswordResetRequested, PasswordReset |

### 8.4 グループ管理機能

| PRD機能 | 対応イベント |
|---------|-------------|
| グループ作成 | GroupCreated |
| メンバー招待 | MemberInvited, InvitationAccepted, MemberJoined |
| メンバー削除 | MemberRemoved |
| ロール変更 | MemberRoleChanged |
| グループ削除 | GroupDeleted |
| オーナー譲渡 | GroupOwnershipTransferred |

### 8.5 権限・共有機能

| PRD機能 | 対応イベント |
|---------|-------------|
| 権限付与 | PermissionGranted, RoleAssigned |
| 権限取消 | PermissionRevoked, RoleRemoved |
| 権限継承 | PermissionInherited |
| 共有リンク作成 | ShareLinkCreated |
| 共有リンクアクセス | ShareLinkAccessed |
| パスワード保護 | ShareLinkPasswordSet |
| 期限付きURL | ShareLinkExpired |

---

## 9. 次のステップ

このイベントストーミングの結果に基づいて、以下の順序でドメインファイルを作成します:

1. **user.md** - Identity Contextのユーザー・認証
2. **group.md** - Collaboration Contextのグループ管理
3. **folder.md** - Storage Contextのフォルダ操作
4. **file.md** - Storage Contextのファイル管理
5. **permission.md** - Authorization Contextの権限管理
6. **sharing.md** - Sharing Contextの共有機能

---

## 関連ドキュメント

- [PRD](../PRD.md) - プロダクト要件定義
- [セキュリティ設計](../02-architecture/SECURITY.md) - 認証・認可設計
- [データベース設計](../02-architecture/DATABASE.md) - スキーマ設計
- [システムアーキテクチャ](../02-architecture/SYSTEM.md) - システム全体構成

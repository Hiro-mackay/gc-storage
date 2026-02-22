# Group ドメイン

## 概要

Groupドメインは、ユーザーの集まりであるグループの作成、メンバー管理、ロール管理を担当します。
Collaboration Contextの中核となるドメインで、グループ単位でのファイル共有・権限管理の基盤を提供します。

### 設計方針

- **グループとフォルダは分離**: グループ作成時にフォルダは作成しない
- **グループはリソース共有の単位**: ストレージ構造とは独立して、共有の「受け皿」として機能
- **権限はPermissionGrantで管理**: グループにフォルダ/ファイルへのロールを付与して共有を実現

---

## エンティティ

### Group（集約ルート）

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | グループの一意識別子 |
| name | GroupName | Yes | グループ名（値オブジェクト） |
| description | string | No | グループの説明（最大500文字） |
| owner_id | UUID | Yes | オーナーのユーザーID |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**注記:** グループは論理削除をサポートしません。削除は物理削除のみです。

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-G001 | グループには必ず1人のownerが存在する |
| R-G002 | nameは空文字不可、1-100文字 |
| R-G003 | descriptionは最大500文字 |
| R-G004 | ownerはグループから脱退できない（所有権譲渡が必要） |
| R-G005 | グループ作成時にフォルダは作成しない |

---

### Membership

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | メンバーシップの一意識別子 |
| group_id | UUID | Yes | グループID |
| user_id | UUID | Yes | ユーザーID |
| role | GroupRole | Yes | グループ内ロール |
| joined_at | timestamp | Yes | 参加日時 |

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-M001 | 同一ユーザーは同一グループに1つのMembershipのみ |
| R-M002 | グループオーナーのMembershipはrole=ownerで固定 |
| R-M003 | ownerロールのMembershipは削除不可（所有権譲渡でのみ変更） |

---

### Invitation

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 招待の一意識別子 |
| group_id | UUID | Yes | グループID |
| email | Email | Yes | 招待先メールアドレス |
| token | string | Yes | 招待トークン（一意） |
| role | GroupRole | Yes | 付与予定のロール（デフォルト: viewer） |
| invited_by | UUID | Yes | 招待者のユーザーID |
| expires_at | timestamp | Yes | 有効期限 |
| status | InvitationStatus | Yes | 招待状態 |
| created_at | timestamp | Yes | 作成日時 |

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-I001 | tokenは全招待で一意 |
| R-I002 | expires_atを過ぎた招待は自動でexpired |
| R-I003 | 既にメンバーのユーザーへの招待は不可 |
| R-I004 | 同一グループ・同一メールへの有効な招待は1つのみ |
| R-I005 | roleにownerは指定不可（所有権譲渡は別フロー） |
| R-I006 | デフォルトのroleはviewer |
| R-I007 | 招待者は自分より低いロールのみ指定可能（`CanAssign()`） |

**ステータス遷移:**
```
┌─────────┐     ┌──────────┐
│ pending │────▶│ accepted │
└────┬────┘     └──────────┘
     │
     ├─────────▶┌──────────┐
     │          │ declined │
     │          └──────────┘
     │
     ├─────────▶┌───────────┐
     │          │ cancelled │
     │          └───────────┘
     │
     └─────────▶┌──────────┐
                │ expired  │
                └──────────┘
```

| ステータス | 説明 |
|-----------|------|
| pending | 招待中 |
| accepted | 承諾済み |
| declined | 辞退済み |
| cancelled | 取り消し済み（招待者による取り消し） |
| expired | 期限切れ |

---

## 値オブジェクト

### GroupName

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | グループ名文字列 |

**要件:**

| ID | 要件 |
|----|------|
| R-GN001 | 1-100文字 |
| R-GN002 | 先頭・末尾の空白はトリム |
| R-GN003 | 空文字は不可 |

### GroupRole

グループ内でのロール階層: `owner > contributor > viewer`

**ロールレベル:**
- owner: 3
- contributor: 2
- viewer: 1

| 値 | 説明 | 権限 |
|-----|------|------|
| viewer | 閲覧者 | グループ情報閲覧、共有リソースアクセス |
| contributor | 投稿者 | メンバー招待（自分より下位のロールのみ付与可能） |
| owner | オーナー | 全権限、グループ削除、設定変更、オーナー譲渡 |

**権限マトリクス:**

| 操作 | viewer | contributor | owner |
|------|:------:|:-----------:|:-----:|
| グループ情報閲覧 | Yes | Yes | Yes |
| 共有リソースアクセス | Yes | Yes | Yes |
| メンバー招待 (`CanInvite()`) | No | Yes | Yes |
| メンバー管理 (`CanManageMembers()`) | No | No | Yes |
| グループ設定変更 | No | No | Yes |
| グループ削除 | No | No | Yes |
| 所有権譲渡 | No | No | Yes |

**招待時のロール付与制限 (`CanAssign()`):**

招待者は自分のレベルより**低い**ロールのみ付与可能（ownerロールは所有権譲渡でのみ変更可能）

| 招待者のロール | 付与可能なロール |
|----------------|------------------|
| owner (Lv3) | viewer, contributor |
| contributor (Lv2) | viewer のみ |
| viewer (Lv1) | なし（招待不可） |

### InvitationStatus

| 値 | 説明 |
|-----|------|
| pending | 招待中 |
| accepted | 承諾済み |
| declined | 辞退済み |
| expired | 期限切れ |

---

## 定数

| 定数名 | 値 | 説明 |
|--------|-----|------|
| GroupNameMaxLength | 100 | グループ名最大長 |
| GroupDescriptionMaxLength | 500 | 説明最大長 |
| InvitationTokenLength | 32 | 招待トークン長（バイト） |
| InvitationExpiry | 7日 | 招待有効期間 |

---

## 操作フロー

### グループ作成

```
1. クライアント → API: CreateGroup（name, description）
2. API:
   - グループ名バリデーション
   - Groupエンティティ作成（status=active）
   - 作成者をownerとしてMembership作成
   ※ フォルダは作成しない
3. API → クライアント: 作成されたGroup返却
```

### グループ更新

```
1. クライアント → API: UpdateGroup（groupId, name?, description?）
2. API:
   - グループ存在・ステータス確認
   - 操作者の権限確認（ownerのみ）
   - グループ情報更新
3. API → クライアント: 更新されたGroup返却
```

### グループ削除

**注記:** グループは物理削除されます。

```
1. クライアント → API: DeleteGroup（groupId）
2. API:
   - グループ存在確認
   - 操作者の権限確認（ownerのみ）
3. トランザクション内:
   - 全招待を削除
   - 全メンバーシップを削除
   - グループへのPermissionGrantを削除
   - グループを物理削除
4. API → クライアント: 成功レスポンス
```

### メンバー招待

```
1. クライアント → API: InviteMember（groupId, email, role?）
2. API:
   - グループ存在・ステータス確認
   - 操作者の権限確認（contributor以上）
   - role未指定の場合はviewerをデフォルト設定
   - roleがownerでないことを確認
   - 招待者のロール以下であることを確認
   - 既存メンバーでないことを確認
   - 同一メールへの有効な招待がないことを確認
   - Invitation作成（status=pending、token生成、expires_at設定）
   - 招待メール送信
3. API → クライアント: 作成されたInvitation返却
```

### 招待承諾

```
1. クライアント → API: AcceptInvitation（token）
2. API:
   - tokenで招待検索
   - 招待の有効性確認（pending && 期限内）
   - 操作者のメール一致確認
   - 既存メンバーでないことを確認
3. トランザクション内:
   - Invitation.status = accepted
   - Membership作成（role=招待時のrole）
4. API → クライアント: 成功レスポンス（グループ情報含む）
```

### 招待辞退

```
1. クライアント → API: DeclineInvitation（token）
2. API:
   - tokenで招待検索
   - 招待がpendingであることを確認
   - 操作者のメール一致確認
   - Invitation.status = declined
3. API → クライアント: 成功レスポンス
```

### 招待取消

```
1. クライアント → API: CancelInvitation（invitationId）
2. API:
   - 招待存在確認
   - 操作者の権限確認（owner）
   - 招待がpendingであることを確認
   - 招待削除
3. API → クライアント: 成功レスポンス
```

### メンバー削除

```
1. クライアント → API: RemoveMember（groupId, userId）
2. API:
   - メンバーシップ存在確認
   - 対象がownerでないことを確認
   - 操作者の権限確認（ownerのみ）
   - Membership削除
3. API → クライアント: 成功レスポンス
```

### グループ脱退

```
1. クライアント → API: LeaveGroup（groupId）
2. API:
   - メンバーシップ存在確認
   - 操作者がownerでないことを確認
   - Membership削除
3. API → クライアント: 成功レスポンス
```

### ロール変更

```
1. クライアント → API: ChangeRole（groupId, userId, newRole）
2. API:
   - メンバーシップ存在確認
   - newRoleがownerでないことを確認
   - 操作者の権限確認（ownerのみ）
   - Membership.role更新
3. API → クライアント: 成功レスポンス
```

### 所有権譲渡

```
1. クライアント → API: TransferOwnership（groupId, newOwnerId）
2. API:
   - グループ存在・ステータス確認
   - 操作者がownerであることを確認
   - 新オーナーがグループメンバーであることを確認
3. トランザクション内:
   - Group.owner_id = newOwnerId
   - 新オーナーのMembership.role = owner
   - 旧オーナーのMembership.role = contributor
4. API → クライアント: 成功レスポンス
```

### 期限切れ招待の処理

```
1. バックグラウンドジョブ: ExpireInvitations（定期実行）
2. 処理:
   - expires_at < 現在時刻 かつ status=pending の招待を検索
   - 該当招待のstatus = expiredに更新
```

---

## リポジトリ

### GroupRepository

| 操作 | 説明 |
|-----|------|
| Create | グループ作成 |
| FindByID | ID検索 |
| Update | 更新 |
| Delete | 削除 |
| FindByOwnerID | オーナーIDで検索 |
| FindByMemberID | メンバーIDで所属グループ取得 |
| ExistsByName | 名前の重複チェック |

### MembershipRepository

| 操作 | 説明 |
|-----|------|
| Create | メンバーシップ作成 |
| FindByID | ID検索 |
| Update | 更新 |
| Delete | 削除 |
| FindByGroupID | グループのメンバーシップ一覧 |
| FindByGroupIDWithUsers | ユーザー情報付きメンバーシップ一覧 |
| FindByUserID | ユーザーのメンバーシップ一覧 |
| FindByGroupAndUser | グループ・ユーザーで検索 |
| Exists | 存在チェック |
| CountByGroupID | グループのメンバー数 |
| DeleteByGroupID | グループの全メンバーシップ削除 |

### InvitationRepository

| 操作 | 説明 |
|-----|------|
| Create | 招待作成 |
| FindByID | ID検索 |
| Update | 更新 |
| Delete | 削除 |
| FindByToken | トークンで検索 |
| FindPendingByGroupID | グループの有効な招待一覧 |
| FindPendingByEmail | メールアドレスへの有効な招待一覧 |
| FindPendingByGroupAndEmail | グループ・メールで有効な招待検索 |
| DeleteByGroupID | グループの全招待削除 |
| ExpireOld | 期限切れ招待を一括更新 |

---

## 不変条件

### オーナー制約

| ID | 不変条件 |
|----|---------|
| I-GO001 | グループには必ず1人のownerが存在する |
| I-GO002 | ownerロールのMembershipは削除不可 |
| I-GO003 | ownerはグループから脱退不可 |
| I-GO004 | ownerの変更は所有権譲渡によってのみ可能 |

### メンバーシップ制約

| ID | 不変条件 |
|----|---------|
| I-GM001 | 同一ユーザーは同一グループに1つのMembershipのみ |
| I-GM002 | メンバーシップ作成時、グループが存在すること |

### 招待制約

| ID | 不変条件 |
|----|---------|
| I-GI001 | tokenは全招待で一意 |
| I-GI002 | 同一グループ・同一メールへの有効な招待は1つのみ |
| I-GI003 | ownerロールでの招待は不可 |
| I-GI004 | 既存メンバーへの招待は不可 |
| I-GI005 | 招待者は自分より低いロールのみ指定可能（`CanAssign()`） |

### グループ削除制約（物理削除）

| ID | 不変条件 |
|----|---------|
| I-GD001 | 削除はownerのみ可能 |
| I-GD002 | 削除時、全メンバーシップと招待も物理削除 |
| I-GD003 | 削除時、グループへのPermissionGrantも削除 |

---

## ユースケース

| ユースケース | アクター | 概要 |
|------------|--------|------|
| CreateGroup | User | グループ作成（作成者がowner） |
| UpdateGroup | Owner | グループ名・説明の変更 |
| DeleteGroup | Owner | グループ削除 |
| InviteMember | Contributor/Owner | メンバー招待（デフォルトはviewer） |
| AcceptInvitation | User | 招待承諾 |
| DeclineInvitation | User | 招待辞退 |
| CancelInvitation | Owner | 招待取消 |
| RemoveMember | Owner | メンバー削除 |
| LeaveGroup | Viewer/Contributor | グループ脱退 |
| ChangeRole | Owner | メンバーロール変更 |
| TransferOwnership | Owner | 所有権譲渡 |
| GetGroup | Member | グループ詳細取得 |
| ListMyGroups | User | 所属グループ一覧 |
| ListMembers | Member | メンバー一覧表示 |
| ListInvitations | Owner | 招待一覧表示 |
| ListPendingInvitations | User | 自分への招待一覧 |

---

## ドメインイベント

| イベント | トリガー | ペイロード |
|---------|---------|-----------|
| GroupCreated | グループ作成 | groupId, name, ownerId |
| GroupUpdated | グループ更新 | groupId, changedFields |
| GroupDeleted | グループ削除 | groupId, deletedBy |
| MemberInvited | メンバー招待 | groupId, email, role, invitedBy |
| InvitationAccepted | 招待承諾 | invitationId, groupId, userId |
| InvitationDeclined | 招待辞退 | invitationId, groupId, userId |
| InvitationExpired | 招待期限切れ | invitationId, groupId |
| InvitationCancelled | 招待取消 | invitationId, groupId, cancelledBy |
| MemberJoined | メンバー参加 | groupId, userId, role |
| MemberLeft | メンバー脱退 | groupId, userId |
| MemberRemoved | メンバー削除 | groupId, userId, removedBy |
| MemberRoleChanged | ロール変更 | groupId, userId, oldRole, newRole, changedBy |
| GroupOwnershipTransferred | 所有権譲渡 | groupId, previousOwnerId, newOwnerId |

---

## 関係性

### エンティティ関係図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Group Domain ERD                                    │
└─────────────────────────────────────────────────────────────────────────────┘

      ┌──────────────────┐
      │      users       │ (Identity Context)
      │    (external)    │
      └────────┬─────────┘
               │
               │ owner_id, user_id, invited_by
               │
               ▼
      ┌──────────────────┐
      │     groups       │
      ├──────────────────┤
      │ id               │
      │ name             │
      │ description      │
      │ owner_id (FK)    │──────────────┐
      │ created_at       │              │
      │ updated_at       │              │
      └────────┬─────────┘              │
               │                        │
    ┌──────────┴──────────┐             │
    │                     │             │
    ▼                     ▼             │
┌──────────────┐   ┌──────────────┐     │
│  memberships │   │  invitations │     │
├──────────────┤   ├──────────────┤     │
│ id           │   │ id           │     │
│ group_id(FK) │   │ group_id(FK) │     │
│ user_id (FK) │───│ email        │     │
│ role         │   │ token        │     │
│ joined_at    │   │ role         │     │
└──────────────┘   │ invited_by   │─────┘
                   │ expires_at   │
                   │ status       │
                   │ created_at   │
                   └──────────────┘
```

### 関係性ルール

| 関係 | カーディナリティ | 説明 |
|-----|----------------|------|
| Group - Owner (User) | N:1 | 各グループは1人のオーナーを持つ |
| Group - Membership | 1:N | 1グループは複数のメンバーシップを持つ |
| Group - Invitation | 1:N | 1グループは複数の招待を持てる |
| User - Membership | 1:N | 1ユーザーは複数グループに所属可能 |
| Invitation - Inviter (User) | N:1 | 各招待は1人の招待者を持つ |

---

## 他コンテキストとの連携

### Identity Context（上流）

- UserIDの参照
- ユーザー情報の取得（表示名、メール）
- 招待時のメールアドレス検証

### Authorization Context（下流）

- グループメンバーシップに基づく権限解決
- Relationship Tuple: `(user, member, group)` の作成・削除
- グループロールに基づくリソースアクセス制御
- グループへのPermissionGrant管理

### Storage Context（下流）

- グループにフォルダ/ファイルへのロールを付与
- PermissionGrantでリソースへのアクセス権を管理
- ※ グループ作成時にフォルダは作成しない

---

## 関連ドキュメント

- [イベントストーミング](./EVENT_STORMING.md) - ドメインイベント定義
- [ユーザードメイン](./user.md) - ユーザー管理
- [権限ドメイン](./permission.md) - 権限管理
- [フォルダドメイン](./folder.md) - フォルダ管理
- [グループ管理仕様](../04-specs/features/group-management.md) - グループ CRUD + メンバー管理
- [招待フロー仕様](../04-specs/features/group-invitation.md) - 招待フロー

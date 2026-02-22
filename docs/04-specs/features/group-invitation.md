# Group Invitation - 招待フロー

## Meta

| Item | Value |
|------|-------|
| Status | Draft |
| Priority | High |
| Tier | 4 (Collab) |
| Domain Refs | `03-domains/group.md` |
| Depends On | `features/group-management.md` |

---

## 1. User Stories

**Primary:**
> As a group contributor/owner, I want to invite users by email so that they can join my group.

**Secondary:**
> As a user, I want to accept or decline group invitations so that I can choose which groups to join.

### Context
招待フローはメール経由でグループ参加を可能にする。招待トークンは32バイトで7日間有効。ロール付与には階層制限があり、招待者は自分より低いロールのみ付与可能。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-I001 | tokenは全招待で一意 | `03-domains/group.md` |
| R-I002 | expires_atを過ぎた招待は自動でexpired | `03-domains/group.md` |
| R-I003 | 既にメンバーのユーザーへの招待は不可 | `03-domains/group.md` |
| R-I004 | 同一グループ・同一メールへの有効な招待は1つのみ | `03-domains/group.md` |
| R-I005 | roleにownerは指定不可 | `03-domains/group.md` |
| R-I006 | デフォルトのroleはviewer | `03-domains/group.md` |
| R-I007 | 招待者は自分より低いロールのみ指定可能（CanAssign） | `03-domains/group.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-GI001 | 招待メールにグループ名と招待URLを含む |
| FS-GI002 | 承諾時にメールアドレスの一致を確認する |
| FS-GI003 | 招待取消はownerのみ可能 |
| FS-GI004 | 期限切れ招待はバックグラウンドジョブで処理 |

### State Transitions
```
         +-----------+
         |  pending  |
         +-----+-----+
               |
    +----------+----------+-----------+
    |          |          |           |
    v          v          v           v
+--------+ +--------+ +----------+ +--------+
|accepted| |declined| |cancelled | |expired |
+--------+ +--------+ +----------+ +--------+
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/groups/:id/invitations` | Required | メンバー招待 |
| GET | `/api/v1/groups/:id/invitations` | Required | グループの招待一覧 |
| DELETE | `/api/v1/groups/:id/invitations/:invitationId` | Required | 招待取消 |
| POST | `/api/v1/invitations/:token/accept` | Required | 招待承諾 |
| POST | `/api/v1/invitations/:token/decline` | Required | 招待辞退 |
| GET | `/api/v1/invitations/pending` | Required | 自分への招待一覧 |

### Request / Response Details

#### `POST /api/v1/groups/:id/invitations` - メンバー招待

**Request Body:**
```json
{ "email": "user@example.com", "role": "viewer" }
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| email | string | Yes | valid email | 招待先メールアドレス |
| role | string | No | oneof: viewer, contributor | 付与ロール（default: viewer） |

**Success Response (201):**
```json
{
  "id": "uuid", "email": "user@example.com",
  "role": "viewer", "expiresAt": "timestamp", "status": "pending"
}
```

#### `POST /api/v1/invitations/:token/accept` - 招待承諾

**Success Response (200):**
```json
{ "groupId": "uuid", "groupName": "Engineering Team", "role": "viewer" }
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | Expired / invalid invitation | `VALIDATION_ERROR` |
| 401 | Not authenticated | `UNAUTHORIZED` |
| 403 | Email mismatch / insufficient permission | `FORBIDDEN` |
| 404 | Invitation not found | `NOT_FOUND` |
| 409 | Already a member / duplicate invitation | `CONFLICT` |

---

## 4. Frontend UI

### Layout / Wireframe
```
Invite Member Dialog:
+----------------------------------+
| Invite Member                    |
+----------------------------------+
| Email: [user@example.com      ] |
| Role:  [viewer v]               |
|                                  |
|         [Cancel]  [Send Invite]  |
+----------------------------------+

Pending Invitations Page:
+----------------------------------+
| Pending Invitations              |
+----------------------------------+
| Engineering Team  viewer         |
|   Invited by Alice, expires 7d   |
|            [Accept] [Decline]    |
+----------------------------------+

Group Invitations List (Owner view):
+----------------------------------+
| Invitations (3)                  |
+----------------------------------+
| bob@example.com   viewer pending |
|                        [Cancel]  |
+----------------------------------+
```

### Components
| Component | Type | Description |
|-----------|------|-------------|
| InviteMemberDialog | Modal | メール + ロール入力フォーム |
| PendingInvitationsPage | Page | 自分への招待一覧 |
| InvitationListPanel | Panel | グループの招待一覧（owner用） |
| InvitationAcceptPage | Page | 招待承諾/辞退ページ |

### State Management
| State | Store | Type | Description |
|-------|-------|------|-------------|
| groupInvitations | TanStack Query | Server | グループの招待一覧 |
| pendingInvitations | TanStack Query | Server | 自分への招待一覧 |

---

## 5. Integration Flow

### Sequence Diagram - メンバー招待
```
Client          Frontend        API             DB          Email
  |                |              |                |           |
  |-- invite ----->|              |                |           |
  |                |-- POST ----->|                |           |
  |                |              |-- validate --->|           |
  |                |              |-- insert inv ->|           |
  |                |<-- 201 ------|                |           |
  |                |              |-- send mail ------------>|
  |<-- update UI --|              |                |           |
```

### Sequence Diagram - 招待承諾
```
Client          Frontend        API             DB
  |                |              |                |
  |-- accept ----->|              |                |
  |                |-- POST ----->|                |
  |                |              |-- begin tx --->|
  |                |              |-- find invite->|
  |                |              |-- check email->|
  |                |              |-- create memb->|
  |                |              |-- update inv ->|
  |                |              |-- commit ----->|
  |                |<-- 200 ------|                |
  |<-- redirect -->|              |                |
```

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: contributor/ownerがメンバーを招待できる
- [ ] AC-02: 招待メールが送信される
- [ ] AC-03: デフォルト招待ロールはviewerになる
- [ ] AC-04: 招待リンクから承諾できる
- [ ] AC-05: 招待を辞退できる
- [ ] AC-06: ownerが招待を取消できる
- [ ] AC-07: 自分への招待一覧を取得できる
- [ ] AC-08: グループの招待一覧を取得できる

### Validation Errors
- [ ] AC-10: ownerロールでの招待は拒否される
- [ ] AC-11: 期限切れ招待は承諾できない

### Authorization
- [ ] AC-20: viewerは招待できない
- [ ] AC-21: contributorはviewerのみ付与可能
- [ ] AC-22: ownerはviewer/contributorを付与可能
- [ ] AC-23: メールアドレスが一致しないユーザーは承諾できない

### Edge Cases
- [ ] AC-30: 同じメールへの重複招待は拒否される
- [ ] AC-31: 既存メンバーへの招待は拒否される
- [ ] AC-32: 7日後に招待が期限切れになる
- [ ] AC-33: 承諾後、招待のstatusがacceptedに更新される
- [ ] AC-34: 辞退後、招待のstatusがdeclinedに更新される

---

## 7. Test Plan

### Backend Unit Tests
| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| Invite member (owner) | InviteMemberCommand | Invitation created, email sent |
| Invite with owner role | InviteMemberCommand | VALIDATION_ERROR |
| Invite existing member | InviteMemberCommand | CONFLICT error |
| Invite duplicate email | InviteMemberCommand | CONFLICT error |
| Contributor invites viewer | InviteMemberCommand | Success |
| Contributor invites contributor | InviteMemberCommand | FORBIDDEN |
| Accept invitation | AcceptInvitationCommand | Membership created, status=accepted |
| Accept expired invitation | AcceptInvitationCommand | VALIDATION_ERROR |
| Accept with wrong email | AcceptInvitationCommand | FORBIDDEN |
| Decline invitation | DeclineInvitationCommand | Status=declined |
| Cancel invitation | CancelInvitationCommand | Invitation deleted |

### Backend Integration Tests
| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| Invite member | POST /groups/:id/invitations | Owner auth | 201, invitation in DB |
| Accept invite | POST /invitations/:token/accept | Matching email | 200, membership created |
| List pending | GET /invitations/pending | Invited user | 200, pending list |

### Frontend Tests
| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| Invite form validation | InviteMemberDialog | Unit | Email required, role options |
| Accept/decline buttons | InvitationAcceptPage | Integration | API calls made |
| Pending list | PendingInvitationsPage | Unit | Invitations displayed |

---

## 8. Implementation Notes

### Changed Files (Backend)
| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/entity/invitation.go` | Invitation entity |
| Domain | `internal/domain/valueobject/invitation_status.go` | InvitationStatus VO |
| UseCase | `internal/usecase/collaboration/command/invite_member.go` | Invite command |
| UseCase | `internal/usecase/collaboration/command/accept_invitation.go` | Accept command |
| UseCase | `internal/usecase/collaboration/command/decline_invitation.go` | Decline command |
| UseCase | `internal/usecase/collaboration/command/cancel_invitation.go` | Cancel command |
| UseCase | `internal/usecase/collaboration/query/list_invitations.go` | List query |
| UseCase | `internal/usecase/collaboration/query/list_pending_invitations.go` | Pending query |
| Interface | `internal/interface/handler/group_handler.go` | Invitation handlers |
| Infra | `internal/infrastructure/repository/invitation_repository.go` | DB impl |
| Job | `internal/job/invitation_expiry.go` | Expiry background job |

### Changed Files (Frontend)
| Category | File | Change |
|----------|------|--------|
| Route | `src/app/routes/invitations/index.tsx` | Pending invitations page |
| Route | `src/app/routes/invite/$token.tsx` | Invitation accept/decline page |
| Component | `src/components/groups/invite-member-dialog.tsx` | Invite form |
| Component | `src/components/groups/invitation-list.tsx` | Invitation list |
| Feature | `src/features/invitations/api.ts` | API client functions |
| Feature | `src/features/invitations/hooks.ts` | TanStack Query hooks |

### Considerations
- **Security**: Invitation token is 32 bytes, cryptographically random
- **Email**: Invitation email sent asynchronously (goroutine)
- **Expiry**: Background job runs hourly to expire old invitations

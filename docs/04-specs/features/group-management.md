# Group Management - グループCRUD + メンバー管理

## Meta

| Item | Value |
|------|-------|
| Status | Draft |
| Priority | High |
| Tier | 4 (Collab) |
| Domain Refs | `03-domains/group.md` |
| Depends On | `features/auth-registration.md` |

---

## 1. User Stories

**Primary:**
> As a user, I want to create a group so that I can organize team members for collaborative file sharing.

**Secondary:**
> As a group owner, I want to manage group members and their roles so that I can control access to shared resources.

### Context
グループはリソース共有の「受け皿」として機能する。グループにフォルダ/ファイルへのロールを PermissionGrant で付与して共有を実現する。グループ作成時にフォルダは作成しない。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-G001 | グループには必ず1人のownerが存在する | `03-domains/group.md` |
| R-G002 | nameは空文字不可、1-100文字 | `03-domains/group.md` |
| R-G003 | descriptionは最大500文字 | `03-domains/group.md` |
| R-G004 | ownerはグループから脱退できない（所有権譲渡が必要） | `03-domains/group.md` |
| R-G005 | グループ作成時にフォルダは作成しない | `03-domains/group.md` |
| R-M001 | 同一ユーザーは同一グループに1つのMembershipのみ | `03-domains/group.md` |
| R-M002 | グループオーナーのMembershipはrole=ownerで固定 | `03-domains/group.md` |
| R-M003 | ownerロールのMembershipは削除不可 | `03-domains/group.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-GM001 | グループ更新はownerのみ可能 |
| FS-GM002 | グループ削除（物理削除）はownerのみ可能 |
| FS-GM003 | 削除時にメンバーシップ・招待・PermissionGrantも削除 |
| FS-GM004 | メンバー削除はownerのみ可能 |
| FS-GM005 | ロール変更はownerのみ可能、ownerロールへの変更不可 |
| FS-GM006 | 自分自身のロールは変更不可 |
| FS-GM007 | 所有権譲渡後、旧ownerはcontributorに降格 |

### State Transitions
```
GroupRole hierarchy: viewer (Lv1) < contributor (Lv2) < owner (Lv3)

Ownership Transfer:
  current_owner -> contributor
  target_member -> owner
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/groups` | Required | グループ作成 |
| GET | `/api/v1/groups` | Required | 所属グループ一覧 |
| GET | `/api/v1/groups/:id` | Required | グループ詳細 |
| PATCH | `/api/v1/groups/:id` | Required | グループ更新 |
| DELETE | `/api/v1/groups/:id` | Required | グループ削除 |
| GET | `/api/v1/groups/:id/members` | Required | メンバー一覧 |
| DELETE | `/api/v1/groups/:id/members/:userId` | Required | メンバー削除 |
| PATCH | `/api/v1/groups/:id/members/:userId/role` | Required | ロール変更 |
| POST | `/api/v1/groups/:id/leave` | Required | グループ脱退 |
| POST | `/api/v1/groups/:id/transfer` | Required | 所有権譲渡 |

### Request / Response Details

#### `POST /api/v1/groups` - グループ作成

**Request Body:**
```json
{ "name": "Engineering Team", "description": "Optional description" }
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| name | string | Yes | min:1, max:100 | グループ名 |
| description | string | No | max:500 | グループ説明 |

**Success Response (201):**
```json
{
  "id": "uuid", "name": "Engineering Team", "description": "...",
  "ownerId": "uuid", "role": "owner", "createdAt": "timestamp"
}
```

#### `PATCH /api/v1/groups/:id/members/:userId/role` - ロール変更

**Request Body:**
```json
{ "role": "contributor" }
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| role | string | Yes | oneof: viewer, contributor | 新しいロール |

#### `POST /api/v1/groups/:id/transfer` - 所有権譲渡

**Request Body:**
```json
{ "newOwnerId": "uuid" }
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | Validation error / self-transfer | `VALIDATION_ERROR` |
| 401 | Not authenticated | `UNAUTHORIZED` |
| 403 | Not owner / insufficient permission | `FORBIDDEN` |
| 404 | Group or member not found | `NOT_FOUND` |

---

## 4. Frontend UI

### Layout / Wireframe
```
Groups List Page:
+----------------------------------+
| My Groups              [+Create] |
+----------------------------------+
| [icon] Engineering Team  owner   |
| [icon] Design Team   contributor |
| [icon] Project Alpha    viewer   |
+----------------------------------+

Group Detail Page:
+----------------------------------+
| <- Back  Engineering Team  [...]  |
| Description: ...                  |
| Members (5)           [+Invite]  |
+----------------------------------+
| Alice (owner)           [...]    |
| Bob (contributor)       [...]    |
| Charlie (viewer)        [...]    |
+----------------------------------+
```

### Components
| Component | Type | Description |
|-----------|------|-------------|
| GroupListPage | Page | 所属グループ一覧 |
| GroupDetailPage | Page | グループ詳細 + メンバー管理 |
| CreateGroupDialog | Modal | グループ作成フォーム |
| MemberListPanel | Panel | メンバー一覧 + ロール表示 |
| RoleChangeDropdown | Dropdown | ロール変更UI |
| TransferOwnershipDialog | Modal | 所有権譲渡確認 |

### State Management
| State | Store | Type | Description |
|-------|-------|------|-------------|
| groups | TanStack Query | Server | 所属グループ一覧 |
| groupDetail | TanStack Query | Server | グループ詳細 |
| members | TanStack Query | Server | メンバー一覧 |

---

## 5. Integration Flow

### Sequence Diagram - グループ作成
```
Client          Frontend        API             DB
  |                |              |                |
  |-- create ----->|              |                |
  |                |-- POST ----->|                |
  |                |              |-- begin tx --->|
  |                |              |-- insert group>|
  |                |              |-- insert memb->|
  |                |              |-- commit ----->|
  |                |<-- 201 ------|                |
  |<-- update UI --|              |                |
```

### Sequence Diagram - 所有権譲渡
```
Client          Frontend        API             DB
  |                |              |                |
  |-- transfer --->|              |                |
  |                |-- POST ----->|                |
  |                |              |-- begin tx --->|
  |                |              |-- update new ->|
  |                |              |-- update old ->|
  |                |              |-- update grp ->|
  |                |              |-- commit ----->|
  |                |<-- 200 ------|                |
  |<-- update UI --|              |                |
```

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: グループを作成でき、作成者がownerになる
- [ ] AC-02: 所属グループ一覧を取得できる
- [ ] AC-03: グループ詳細（メンバー数含む）を取得できる
- [ ] AC-04: ownerがグループ名・説明を更新できる
- [ ] AC-05: ownerがグループを物理削除できる
- [ ] AC-06: メンバー一覧（ユーザー情報付き）を取得できる
- [ ] AC-07: ownerがメンバーを削除できる
- [ ] AC-08: メンバーがグループから脱退できる
- [ ] AC-09: ownerがメンバーのロールを変更できる
- [ ] AC-10: ownerが他のメンバーに所有権を譲渡できる

### Validation Errors
- [ ] AC-11: 空のグループ名では作成できない
- [ ] AC-12: 101文字以上のグループ名では作成できない
- [ ] AC-13: ownerロールへの変更は拒否される

### Authorization
- [ ] AC-20: owner以外はグループを更新できない
- [ ] AC-21: owner以外はメンバーを削除できない
- [ ] AC-22: owner以外はロールを変更できない
- [ ] AC-23: 非メンバーはグループ詳細を閲覧できない

### Edge Cases
- [ ] AC-30: ownerはグループから脱退できない
- [ ] AC-31: ownerのMembershipは削除できない
- [ ] AC-32: 自分自身のロールは変更できない
- [ ] AC-33: 譲渡後、旧ownerはcontributorになる
- [ ] AC-34: 非メンバーへの所有権譲渡は拒否される
- [ ] AC-35: 削除時にメンバーシップ・招待・PermissionGrantも削除される

---

## 7. Test Plan

### Backend Unit Tests
| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| Create group | CreateGroupCommand | Group + owner Membership created |
| Update group (owner) | UpdateGroupCommand | Name/description updated |
| Update group (non-owner) | UpdateGroupCommand | FORBIDDEN error |
| Delete group | DeleteGroupCommand | Memberships, invitations, grants deleted |
| Remove member | RemoveMemberCommand | Membership deleted |
| Remove owner | RemoveMemberCommand | Error: cannot remove owner |
| Leave group | LeaveGroupCommand | Membership deleted |
| Leave group (owner) | LeaveGroupCommand | Error: owner cannot leave |
| Change role | ChangeRoleCommand | Role updated |
| Change role to owner | ChangeRoleCommand | Error: use transfer |
| Transfer ownership | TransferOwnershipCommand | Roles swapped, group updated |

### Backend Integration Tests
| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| Create group | POST /api/v1/groups | Auth user | 201, group in DB |
| List groups | GET /api/v1/groups | User with memberships | 200, correct list |
| Delete group | DELETE /api/v1/groups/:id | Owner | 204, cascade delete |

### Frontend Tests
| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| Group list rendering | GroupListPage | Unit | Groups displayed with roles |
| Create group form | CreateGroupDialog | Integration | Submit creates group |
| Role change | RoleChangeDropdown | Unit | Dropdown options correct |

---

## 8. Implementation Notes

### Changed Files (Backend)
| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/entity/group.go` | Group entity |
| Domain | `internal/domain/entity/membership.go` | Membership entity |
| Domain | `internal/domain/valueobject/group_name.go` | GroupName VO |
| Domain | `internal/domain/valueobject/group_role.go` | GroupRole VO |
| UseCase | `internal/usecase/collaboration/command/create_group.go` | Create command |
| UseCase | `internal/usecase/collaboration/command/delete_group.go` | Delete command |
| UseCase | `internal/usecase/collaboration/command/remove_member.go` | Remove command |
| UseCase | `internal/usecase/collaboration/command/leave_group.go` | Leave command |
| UseCase | `internal/usecase/collaboration/command/change_role.go` | Role change |
| UseCase | `internal/usecase/collaboration/command/transfer_ownership.go` | Transfer |
| UseCase | `internal/usecase/collaboration/query/get_group.go` | Get query |
| UseCase | `internal/usecase/collaboration/query/list_my_groups.go` | List query |
| UseCase | `internal/usecase/collaboration/query/list_members.go` | Members query |
| Interface | `internal/interface/handler/group_handler.go` | HTTP handlers |
| Interface | `internal/interface/dto/request/group.go` | Request DTOs |
| Interface | `internal/interface/dto/response/group.go` | Response DTOs |
| Infra | `internal/infrastructure/repository/group_repository.go` | DB impl |
| Infra | `internal/infrastructure/repository/membership_repository.go` | DB impl |

### Changed Files (Frontend)
| Category | File | Change |
|----------|------|--------|
| Route | `src/app/routes/groups/index.tsx` | Groups list page |
| Route | `src/app/routes/groups/$groupId.tsx` | Group detail page |
| Component | `src/components/groups/create-group-dialog.tsx` | Create dialog |
| Component | `src/components/groups/member-list.tsx` | Member list panel |
| Feature | `src/features/groups/api.ts` | API client functions |
| Feature | `src/features/groups/hooks.ts` | TanStack Query hooks |

### Considerations
- **Performance**: ListMyGroups does N+1 queries; optimize with JOIN query
- **Security**: All endpoints require session_id cookie authentication
- **Cascade**: Group deletion cascades to memberships, invitations, PermissionGrants

# Permission Management - PBAC+ReBAC + 権限UI

## Meta

| Item | Value |
|------|-------|
| Status | Draft |
| Priority | High |
| Tier | 5 (Cross-cutting) |
| Domain Refs | `03-domains/permission.md` |
| Depends On | `features/group-management.md` |

---

## 1. User Stories

**Primary:**
> As a resource owner/contributor, I want to grant roles on files and folders to users or groups so that I can share access.

**Secondary:**
> As a resource owner, I want to revoke granted permissions so that I can control who has access.

### Context
認可モデルはPBAC+ReBAC hybrid。PBACで最終判定（「このPermissionを持っているか？」）、ReBACで解決（「どの関係性を通じてPermissionを得ているか？」）。ロール階層: Owner > Content Manager > Contributor > Viewer。フォルダ階層を通じた権限継承、グループメンバーシップ経由の権限解決をサポートする。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-PG001 | roleまたはpermissionのいずれかは必須 | `03-domains/permission.md` |
| R-PG002 | 同一(resource, grantee, role, permission)は一意 | `03-domains/permission.md` |
| R-PG003 | ownerロールは直接付与不可（所有権譲渡で管理） | `03-domains/permission.md` |
| R-PG004 | 付与者はpermission:grant権限が必要 | `03-domains/permission.md` |
| R-PG005 | 付与者は自分のロール以下のみ付与可能 | `03-domains/permission.md` |
| R-R001 | Relationship tupleは一意 | `03-domains/permission.md` |
| R-R002 | ownerリレーションは1リソースにつき1つ | `03-domains/permission.md` |
| R-R003 | parentリレーションは循環参照不可 | `03-domains/permission.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-PM001 | 権限収集順: owner check -> direct grants -> group grants -> hierarchy |
| FS-PM002 | 複数経路からの権限は合算（最も高いロールが有効） |
| FS-PM003 | ownerロールの取り消しは不可（所有権譲渡を使用） |
| FS-PM004 | 移動操作は移動元にmove_out、移動先にmove_in権限が必要 |

### Role Permissions Matrix
```
                     Viewer  Contributor  ContentMgr  Owner
file:read              Y        Y           Y         Y
folder:read            Y        Y           Y         Y
file:write             -        Y           Y         Y
file:rename            -        Y           Y         Y
file:delete            -        Y           Y         Y
file:restore           -        Y           Y         Y
file:move_in           -        Y           Y         Y
file:move_out          -        -           Y         Y
file:share             -        Y           Y         Y
folder:create          -        Y           Y         Y
folder:rename          -        Y           Y         Y
folder:delete          -        Y           Y         Y
folder:move_in         -        Y           Y         Y
folder:move_out        -        -           Y         Y
folder:share           -        Y           Y         Y
permission:read        -        Y           Y         Y
permission:grant       -        Y           Y         Y
permission:revoke      -        Y           Y         Y
file:permanent_delete  -        -           -         Y
root:delete            -        -           -         Y
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/files/:id/permissions` | Required | ファイル権限付与 |
| GET | `/api/v1/files/:id/permissions` | Required | ファイル権限一覧 |
| POST | `/api/v1/folders/:id/permissions` | Required | フォルダ権限付与 |
| GET | `/api/v1/folders/:id/permissions` | Required | フォルダ権限一覧 |
| DELETE | `/api/v1/permissions/:id` | Required | 権限取り消し |

### Request / Response Details

#### `POST /api/v1/{files|folders}/:id/permissions` - 権限付与

**Request Body:**
```json
{
  "grantee_type": "user",
  "grantee_id": "uuid",
  "role": "contributor"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| grantee_type | string | Yes | oneof: user, group | 付与先種別 |
| grantee_id | UUID | Yes | valid UUID | 付与先ID |
| role | string | Yes | oneof: viewer, contributor, content_manager | 付与ロール |

**Success Response (201):**
```json
{
  "id": "uuid", "grantee_type": "user", "grantee_id": "uuid",
  "role": "contributor", "granted_at": "timestamp"
}
```

#### `GET /api/v1/{files|folders}/:id/permissions` - 権限一覧

**Success Response (200):**
```json
{
  "grants": [
    {
      "id": "uuid", "grantee_type": "user", "grantee_id": "uuid",
      "grantee_name": "Alice", "role": "contributor", "granted_at": "timestamp"
    }
  ]
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | Invalid role / owner grant attempt | `VALIDATION_ERROR` |
| 401 | Not authenticated | `UNAUTHORIZED` |
| 403 | No permission:grant / higher role grant | `FORBIDDEN` |
| 404 | Resource or grant not found | `NOT_FOUND` |
| 409 | Duplicate grant | `CONFLICT` |

---

## 4. Frontend UI

### Layout / Wireframe
```
Permission Settings Panel (on file/folder detail):
+----------------------------------+
| Sharing & Permissions            |
+----------------------------------+
| Owner: Alice                     |
+----------------------------------+
| Shared with:                     |
| [avatar] Bob     contributor [v] |
| [avatar] Design  viewer     [v] |
|                   [+ Add user]   |
+----------------------------------+

Grant Role Dialog:
+----------------------------------+
| Share with                       |
+----------------------------------+
| User/Group: [search...        ] |
| Role:       [contributor v]      |
|                                  |
|          [Cancel]  [Share]       |
+----------------------------------+
```

### Components
| Component | Type | Description |
|-----------|------|-------------|
| PermissionSettingsPanel | Panel | ファイル/フォルダの権限設定 |
| GrantRoleDialog | Modal | ユーザー/グループにロール付与 |
| GranteeList | List | 権限付与先一覧 |
| RoleDropdown | Dropdown | ロール選択 (viewer/contributor/content_manager) |

### State Management
| State | Store | Type | Description |
|-------|-------|------|-------------|
| resourcePermissions | TanStack Query | Server | リソースの権限一覧 |
| effectiveRole | TanStack Query | Server | 自分の有効ロール |

---

## 5. Integration Flow

### Sequence Diagram - 権限付与
```
Client          Frontend        API             Resolver        DB
  |                |              |                |              |
  |-- grant ------>|              |                |              |
  |                |-- POST ----->|                |              |
  |                |              |-- hasPermission>|              |
  |                |              |                |-- check ---->|
  |                |              |<-- allowed -----|              |
  |                |              |-- getRole ----->|              |
  |                |              |<-- canGrant ----|              |
  |                |              |-- begin tx --->|              |
  |                |              |-- insert grant>|              |
  |                |              |-- insert rel ->|              |
  |                |              |-- commit ----->|              |
  |                |<-- 201 ------|                |              |
  |<-- update UI --|              |                |              |
```

### Permission Resolution Algorithm
```
CollectPermissions(userID, resourceType, resourceID):
  1. Check owner relationship -> if owner, return ALL permissions
  2. Collect direct grants (user -> resource)
  3. Find user's groups, collect group grants (group -> resource)
  4. Walk ancestor hierarchy (parent folders)
     - For each ancestor: collect direct + group grants
  5. Merge all permissions into PermissionSet
  6. Return PermissionSet
```

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: viewer/contributor/content_managerロールを付与できる
- [ ] AC-02: ユーザーまたはグループに権限を付与できる
- [ ] AC-03: リソースの権限一覧を取得できる
- [ ] AC-04: 付与した権限を取り消せる

### Validation Errors
- [ ] AC-10: ownerロールの直接付与は拒否される
- [ ] AC-11: 同じロールの重複付与は拒否される

### Authorization
- [ ] AC-20: permission:grant権限がないと付与できない
- [ ] AC-21: 自分のロールより高いロールは付与できない
- [ ] AC-22: permission:revoke権限がないと取り消せない
- [ ] AC-23: ownerロールの取り消しは拒否される
- [ ] AC-24: Viewerは共有不可

### Permission Inheritance
- [ ] AC-30: 親フォルダの権限が子フォルダ・ファイルに継承される
- [ ] AC-31: グループ経由の権限が解決される
- [ ] AC-32: 複数経路からの権限が合算される
- [ ] AC-33: ownerは全権限を持つ

### Role-Based Permissions
- [ ] AC-40: Viewerはfile:read, folder:readのみ
- [ ] AC-41: Contributorは作成/編集/削除/共有 + move_in可能
- [ ] AC-42: Content Managerはmove_outも可能
- [ ] AC-43: Ownerはルートフォルダ削除と完全削除が可能

### Move Permissions
- [ ] AC-50: 移動元フォルダにmove_out権限が必要
- [ ] AC-51: 移動先フォルダにmove_in権限が必要

---

## 7. Test Plan

### Backend Unit Tests
| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| Grant role (contributor) | GrantRoleCommand | Grant + relationship created |
| Grant owner role | GrantRoleCommand | FORBIDDEN error |
| Grant higher role | GrantRoleCommand | FORBIDDEN error |
| Duplicate grant | GrantRoleCommand | CONFLICT error |
| Revoke grant | RevokeGrantCommand | Grant + relationship deleted |
| Revoke owner grant | RevokeGrantCommand | BAD_REQUEST error |
| List grants | ListGrantsQuery | All grants returned |
| HasPermission (direct) | PermissionResolver | true |
| HasPermission (group) | PermissionResolver | true via group membership |
| HasPermission (hierarchy) | PermissionResolver | true via parent folder |
| HasPermission (owner) | PermissionResolver | true for all permissions |
| HasPermission (none) | PermissionResolver | false |

### Backend Integration Tests
| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| Grant role | POST /files/:id/permissions | Contributor auth | 201, grant in DB |
| Revoke grant | DELETE /permissions/:id | Permission holder | 204 |
| List grants | GET /folders/:id/permissions | Member | 200, grants listed |

### Frontend Tests
| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| Permission panel | PermissionSettingsPanel | Unit | Grants displayed |
| Grant dialog | GrantRoleDialog | Integration | API call with correct role |
| Role dropdown options | RoleDropdown | Unit | Only grantable roles shown |

---

## 8. Implementation Notes

### Changed Files (Backend)
| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/authz/permission.go` | Permission constants |
| Domain | `internal/domain/authz/role.go` | Role + hierarchy + CanGrant |
| Domain | `internal/domain/authz/permission_grant.go` | PermissionGrant entity |
| Domain | `internal/domain/authz/relationship.go` | Relationship entity |
| Domain | `internal/domain/authz/relation.go` | RelationType constants |
| Domain | `internal/domain/authz/permission_resolver.go` | Resolver interface |
| UseCase | `internal/usecase/authz/grant_role.go` | Grant command |
| UseCase | `internal/usecase/authz/revoke_grant.go` | Revoke command |
| UseCase | `internal/usecase/authz/list_grants.go` | List query |
| UseCase | `internal/usecase/authz/set_owner.go` | Owner setup |
| UseCase | `internal/usecase/authz/set_parent.go` | Parent setup |
| Infra | `internal/infrastructure/authz/permission_resolver.go` | Resolver impl |
| Interface | `internal/interface/handler/permission_handler.go` | HTTP handlers |
| Interface | `internal/interface/middleware/permission.go` | Permission middleware |
| Interface | `internal/interface/dto/permission.go` | DTOs |

### Changed Files (Frontend)
| Category | File | Change |
|----------|------|--------|
| Component | `src/components/permissions/settings-panel.tsx` | Permission panel |
| Component | `src/components/permissions/grant-dialog.tsx` | Grant dialog |
| Component | `src/components/permissions/grantee-list.tsx` | Grantee list |
| Feature | `src/features/permissions/api.ts` | API client functions |
| Feature | `src/features/permissions/hooks.ts` | TanStack Query hooks |

### Migration
```sql
CREATE TABLE permission_grants (
    id UUID PRIMARY KEY,
    resource_type VARCHAR(10) NOT NULL,
    resource_id UUID NOT NULL,
    grantee_type VARCHAR(10) NOT NULL,
    grantee_id UUID NOT NULL,
    role VARCHAR(20),
    permission VARCHAR(30),
    granted_by UUID NOT NULL REFERENCES users(id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(resource_type, resource_id, grantee_type, grantee_id, role, permission)
);

CREATE TABLE relationships (
    id UUID PRIMARY KEY,
    subject_type VARCHAR(10) NOT NULL,
    subject_id UUID NOT NULL,
    relation VARCHAR(20) NOT NULL,
    object_type VARCHAR(10) NOT NULL,
    object_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(subject_type, subject_id, relation, object_type, object_id)
);
```

### Considerations
- **Performance**: Permission resolution may require multiple queries; consider caching
- **Security**: RequirePermission middleware enforces permissions at route level
- **Hierarchy**: Ancestor traversal uses parent relationships (not closure table)

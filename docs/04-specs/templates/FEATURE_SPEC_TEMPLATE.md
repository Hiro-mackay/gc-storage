# {Feature Name}

## Meta

| Item | Value |
|------|-------|
| Status | Draft / Ready / In Progress / Done |
| Priority | High / Medium / Low |
| Tier | 1 (Auth) / 2 (Storage) / 3 (Secondary) / 4 (Collab) / 5 (Cross-cutting) |
| Domain Refs | `03-domains/{xxx}.md` |
| Depends On | `features/{xxx}.md` |

---

## 1. User Stories

**Primary:**
> As a {actor}, I want to {action} so that {benefit}.

**Secondary (if any):**
> As a {actor}, I want to {action} so that {benefit}.

### Context
{Why this feature matters. Current pain points or requirements.}

---

## 2. Domain Behaviors

### Referenced Domain Rules
{Link to relevant rules in 03-domains/*.md with brief summary}

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-XXX | ... | `03-domains/xxx.md` |

### Feature-Specific Rules
{Rules unique to this feature that aren't covered in domain docs}

| Rule ID | Description |
|---------|-------------|
| FS-001 | ... |

### State Transitions
```
{ASCII state diagram if applicable}
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/...` | Required | ... |
| GET | `/api/v1/...` | Required | ... |

### Request / Response Details

#### `POST /api/v1/...` - {Description}

**Request Body:**
```json
{
  "field": "value"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| field | string | Yes | max:255 | ... |

**Success Response (200/201):**
```json
{
  "data": {}
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | Validation error | `VALIDATION_ERROR` |
| 401 | Not authenticated | `UNAUTHORIZED` |
| 403 | Insufficient permission | `FORBIDDEN` |
| 404 | Resource not found | `NOT_FOUND` |
| 409 | Conflict (duplicate) | `CONFLICT` |

---

## 4. Frontend UI

### Layout / Wireframe
```
{ASCII wireframe}
```

### Components
| Component | Type | Description |
|-----------|------|-------------|
| ... | Page / Modal / Form | ... |

### State Management
| State | Store | Type | Description |
|-------|-------|------|-------------|
| ... | TanStack Query / Zustand / useState | Server / Global / Local | ... |

### User Interactions
1. User does {action}
2. UI shows {response}
3. ...

---

## 5. Integration Flow

### Sequence Diagram
```
Client          Frontend        API             DB/Service
  |                |              |                |
  |-- action ----->|              |                |
  |                |-- POST ----->|                |
  |                |              |-- query ------>|
  |                |              |<-- result -----|
  |                |<-- 200 ------|                |
  |<-- update UI --|              |                |
```

### Error Handling Flow
{How errors propagate from backend to frontend to user}

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: Given {precondition}, when {action}, then {expected result}
- [ ] AC-02: ...

### Validation Errors
- [ ] AC-10: Given {invalid input}, when {action}, then {error shown}

### Authorization
- [ ] AC-20: Given {unauthorized user}, when {action}, then {403 returned}

### Edge Cases
- [ ] AC-30: Given {boundary condition}, when {action}, then {expected behavior}

---

## 7. Test Plan

### Backend Unit Tests
| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| ... | ... | ... |

### Backend Integration Tests
| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| ... | POST /api/v1/... | ... | ... |

### Frontend Tests
| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| ... | ... | Unit / Integration | ... |

### E2E Tests (future)
| Test | Flow | Assertions |
|------|------|------------|
| ... | ... | ... |

---

## 8. Implementation Notes

### Changed Files (Backend)
| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/...` | ... |
| UseCase | `internal/usecase/...` | ... |
| Interface | `internal/interface/...` | ... |
| Infra | `internal/infrastructure/...` | ... |

### Changed Files (Frontend)
| Category | File | Change |
|----------|------|--------|
| Route | `src/app/routes/...` | ... |
| Component | `src/components/...` | ... |
| Feature | `src/features/...` | ... |
| Store | `src/stores/...` | ... |
| API | `src/lib/api/...` | ... |

### Migration
```sql
-- If DB changes needed
```

### Considerations
- **Performance**: ...
- **Security**: ...
- **Backward Compatibility**: ...

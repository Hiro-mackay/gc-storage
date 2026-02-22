# CLAUDE.md

## Commands

```bash
task dev              # full local environment
task test             # unit tests (backend + frontend)
task test:integration # integration tests
task check            # lint + test
task generate         # codegen (api types + sqlc + mocks)
```

## Architecture

- UseCase: CQRS — `{Action}Command` (writes) / `{Action}Query` (reads), method always `Execute`
- Auth: Session-based (HttpOnly Cookie `session_id`), NOT JWT Bearer
- DTO redundancy across layers is INTENTIONAL — do NOT consolidate Request DTO / UseCase Input / Entity
- API responses: always `{ "data": T, "meta": { ... } }` envelope — never return raw objects
- Auth info (userID) = explicit function parameter, NOT via context.Context
- Domain docs: read `docs/03-domains/*.md` before implementing related features

## Rules

- TDD required: RED -> GREEN -> REFACTOR (see `docs/01-policies/TDD_WORKFLOW.md`)
- NEVER write implementation code in `docs/` — only type defs, schemas, API specs
- Test naming: `TestFunctionName_Scenario_ExpectedBehavior`

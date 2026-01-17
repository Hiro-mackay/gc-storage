# GC Storage 開発者ガイド（Contributing Guide）

## 概要

GC Storageへのコントリビューションに関するガイドラインです。コードの品質と一貫性を保つため、以下の規約に従ってください。

---

## 1. 開発フロー

### 1.1 ブランチ戦略

**GitHub Flow** をベースとした戦略を採用しています。

```
main (本番環境)
 │
 ├── feature/xxx  (機能開発)
 ├── fix/xxx      (バグ修正)
 ├── hotfix/xxx   (緊急修正)
 ├── refactor/xxx (リファクタリング)
 ├── docs/xxx     (ドキュメント)
 └── chore/xxx    (設定・依存関係)
```

### 1.2 ブランチ命名規則

| プレフィックス | 用途 | 例 |
|--------------|------|-----|
| `feature/` | 新機能開発 | `feature/file-preview` |
| `fix/` | バグ修正 | `fix/upload-timeout` |
| `hotfix/` | 本番緊急修正 | `hotfix/auth-bypass` |
| `refactor/` | リファクタリング | `refactor/file-handler` |
| `docs/` | ドキュメント更新 | `docs/api-guide` |
| `chore/` | 設定・CI/CD | `chore/update-deps` |

**命名ルール:**
- 小文字のみ使用
- 単語の区切りはハイフン（`-`）
- 簡潔かつ説明的に
- Issue番号がある場合は含める: `feature/123-file-preview`

### 1.3 開発の流れ

```bash
# 1. mainブランチを最新化
git checkout main
git pull origin main

# 2. 作業ブランチを作成
git checkout -b feature/file-preview

# 3. 開発作業
# ...

# 4. コミット（後述のコミット規約に従う）
git add .
git commit -m "feat(file): add file preview functionality"

# 5. リモートにプッシュ
git push origin feature/file-preview

# 6. Pull Request を作成
```

---

## 2. コミット規約

### 2.1 Conventional Commits

[Conventional Commits](https://www.conventionalcommits.org/) に準拠します。

**フォーマット:**
```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

### 2.2 Type（必須）

| Type | 説明 |
|------|------|
| `feat` | 新機能 |
| `fix` | バグ修正 |
| `docs` | ドキュメントのみの変更 |
| `style` | コードの意味に影響しない変更（フォーマット等） |
| `refactor` | バグ修正でも新機能でもないコード変更 |
| `perf` | パフォーマンス改善 |
| `test` | テストの追加・修正 |
| `chore` | ビルドプロセスやツールの変更 |
| `ci` | CI設定の変更 |
| `revert` | コミットの取り消し |

### 2.3 Scope（任意）

影響範囲を示す:

| Scope | 対象 |
|-------|------|
| `file` | ファイル関連機能 |
| `folder` | フォルダ関連機能 |
| `auth` | 認証機能 |
| `group` | グループ機能 |
| `share` | 共有機能 |
| `api` | API全般 |
| `ui` | UIコンポーネント |
| `db` | データベース |
| `infra` | インフラストラクチャ |

### 2.4 Subject（必須）

- 命令形で記述（"Add feature" ではなく "add feature"）
- 英語で記述
- 50文字以内
- 末尾にピリオドを付けない
- 最初の文字は小文字

### 2.5 コミットメッセージ例

```bash
# Good
feat(file): add multipart upload support
fix(auth): resolve token refresh race condition
docs(api): update rate limit documentation
refactor(folder): extract path resolution logic
test(file): add unit tests for upload service
chore(deps): update go modules to latest versions

# Bad
Fixed bug                    # type がない、説明が不十分
feat: Add new feature.       # scope がない、末尾にピリオド
FEAT(FILE): ADD UPLOAD       # 大文字使用
```

### 2.6 Breaking Changes

破壊的変更がある場合は、フッターに `BREAKING CHANGE:` を記載:

```bash
feat(api): change file response format

BREAKING CHANGE: The `file_id` field is now `id` in all responses.
Migration guide: Update all client code to use the new field name.
```

---

## 3. Pull Request プロセス

### 3.1 PR作成前のチェックリスト

```
- [ ] ブランチ名が命名規則に従っている
- [ ] コミットメッセージが規約に従っている
- [ ] ローカルでテストが通っている
- [ ] Lintエラーがない
- [ ] 必要に応じてドキュメントを更新している
- [ ] 関連するIssueにリンクしている
```

### 3.2 PRテンプレート

```markdown
## Summary

<!-- 変更内容を簡潔に説明 -->

## Changes

-
-
-

## Related Issues

Closes #123

## Type of Change

- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change
- [ ] Documentation update
- [ ] Refactoring
- [ ] Other (please describe)

## Testing

<!-- テスト方法を説明 -->

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed

## Checklist

- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated (if needed)
- [ ] Tests pass locally
- [ ] No new warnings introduced
```

### 3.3 PR の粒度

- **小さく保つ**: 1つのPRで1つの論理的な変更
- **目安**: 変更行数は300行以下を推奨
- **大きな機能**: 複数のPRに分割（feature flagを使用）

### 3.4 レビュー依頼

```bash
# PR作成後、以下を確認
1. CIが全てパスしていること
2. 適切なレビュアーをアサイン
3. ラベルを付与（enhancement, bug, documentation等）
```

---

## 4. コードレビュー基準

### 4.1 レビュアーの観点

| 観点 | チェック内容 |
|------|------------|
| 機能性 | 要件を満たしているか |
| 設計 | アーキテクチャに沿っているか |
| 可読性 | コードが理解しやすいか |
| テスト | 十分なテストカバレッジか |
| セキュリティ | 脆弱性がないか |
| パフォーマンス | 効率的な実装か |

### 4.2 レビューコメントの種類

| プレフィックス | 意味 |
|--------------|------|
| `[Must]` | 必須の修正 |
| `[Should]` | 強く推奨される修正 |
| `[Nit]` | 些細な指摘（任意） |
| `[Question]` | 質問・確認 |
| `[Suggestion]` | 提案（代替案） |

### 4.3 レビューの進め方

1. **セルフレビュー**: PR作成者がまず自己レビュー
2. **自動チェック**: CIでLint、テスト、ビルドを確認
3. **コードレビュー**: 1名以上の承認が必要
4. **マージ**: Squash and merge を使用

### 4.4 マージ条件

- [ ] 1名以上のApprove
- [ ] CI全てパス
- [ ] コンフリクトなし
- [ ] 必須コメントが解決済み

---

## 5. コーディング規約

### 5.1 Go（バックエンド）

**基本ルール:**
- `gofmt` でフォーマット
- `golangci-lint` でLint
- [Effective Go](https://go.dev/doc/effective_go) に準拠

**命名規則:**

| 種類 | 規則 | 例 |
|------|------|-----|
| パッケージ | 小文字、単数形 | `entity`, `handler` |
| ファイル | スネークケース | `file_handler.go` |
| 構造体 | パスカルケース | `FileHandler` |
| インターフェース | パスカルケース | `FileRepository` |
| メソッド | パスカルケース | `GetByID` |
| 変数 | キャメルケース | `userID` |
| 定数 | パスカルケース or キャメルケース | `MaxFileSize` |

**ディレクトリ構成:**
```
backend/
├── cmd/api/          # エントリーポイント
├── internal/         # 内部パッケージ
│   ├── domain/       # ドメイン層
│   ├── usecase/      # ユースケース層
│   ├── interface/    # インターフェース層
│   └── infrastructure/ # インフラ層
├── pkg/              # 共有パッケージ
└── migrations/       # マイグレーション
```

### 5.2 TypeScript/React（フロントエンド）

**基本ルール:**
- ESLint + Prettier でフォーマット
- 厳格な TypeScript 設定を使用
- 関数コンポーネントを使用

**命名規則:**

| 種類 | 規則 | 例 |
|------|------|-----|
| コンポーネント | パスカルケース | `FileList.tsx` |
| フック | `use` プレフィックス | `useFileUpload.ts` |
| ユーティリティ | キャメルケース | `formatDate.ts` |
| 型 | パスカルケース | `FileResponse` |
| 定数 | UPPER_SNAKE_CASE | `MAX_FILE_SIZE` |

**ディレクトリ構成:**
```
frontend/
├── src/
│   ├── app/           # ルーティング
│   ├── components/    # UIコンポーネント
│   ├── features/      # 機能モジュール
│   ├── stores/        # Zustand ストア
│   ├── hooks/         # カスタムフック
│   ├── lib/           # ユーティリティ
│   └── types/         # 型定義
```

### 5.3 SQL

**命名規則:**

| 対象 | 規則 | 例 |
|------|------|-----|
| テーブル名 | 複数形、スネークケース | `users`, `file_versions` |
| カラム名 | スネークケース | `created_at`, `user_id` |
| インデックス | `idx_{table}_{column}` | `idx_files_folder_id` |

---

## 6. ドキュメント規約

### 6.1 コメント

**Go:**
```go
// GetByID retrieves a file by its ID.
// Returns ErrNotFound if the file does not exist.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*File, error) {
    // ...
}
```

**TypeScript:**
```typescript
/**
 * Uploads a file to the storage.
 * @param file - The file to upload
 * @param options - Upload options
 * @returns Promise that resolves when upload is complete
 */
async function uploadFile(file: File, options?: UploadOptions): Promise<void> {
    // ...
}
```

### 6.2 README更新

機能追加時は関連するREADMEを更新:
- 新しいAPI → API.md
- 新しい環境変数 → SETUP.md
- 新しい依存関係 → SETUP.md

---

## 7. Issue と Project 管理

### 7.1 Issue テンプレート

**Bug Report:**
```markdown
## Description
<!-- バグの説明 -->

## Steps to Reproduce
1.
2.
3.

## Expected Behavior
<!-- 期待される動作 -->

## Actual Behavior
<!-- 実際の動作 -->

## Environment
- OS:
- Browser:
- Version:

## Screenshots
<!-- あれば添付 -->
```

**Feature Request:**
```markdown
## Summary
<!-- 機能の概要 -->

## Problem
<!-- 解決したい課題 -->

## Proposed Solution
<!-- 提案する解決策 -->

## Alternatives Considered
<!-- 検討した代替案 -->

## Additional Context
<!-- 補足情報 -->
```

### 7.2 ラベル

| ラベル | 説明 |
|--------|------|
| `bug` | バグ報告 |
| `enhancement` | 機能追加 |
| `documentation` | ドキュメント |
| `good first issue` | 初心者向け |
| `help wanted` | 協力募集 |
| `priority: high` | 高優先度 |
| `priority: low` | 低優先度 |
| `wontfix` | 対応しない |
| `duplicate` | 重複 |

---

## 8. セキュリティ

### 8.1 脆弱性の報告

セキュリティ上の脆弱性を発見した場合:

1. **公開Issueを作成しない**
2. セキュリティチームに直接連絡
3. 詳細な再現手順を提供
4. 修正がリリースされるまで非公開を維持

### 8.2 機密情報の取り扱い

```
絶対にコミットしてはいけないもの:
- API キー（本番用）
- パスワード（本番用）
- 秘密鍵
- .env ファイル（本番用）
- credentials.json
```

**環境変数ファイルの管理:**

| ファイル | Git管理 | 説明 |
|---------|---------|------|
| `.env.local` | ✅ 対象 | ローカル開発用（固定値） |
| `.env.sample` | ✅ 対象 | 本番用テンプレート |
| `.env` | ❌ 対象外 | 本番/ステージング用 |
| `.env.production` | ❌ 対象外 | 本番用 |

`.gitignore` に含まれていることを確認:
```
.env
.env.production
*.pem
*.key
credentials.json

# .env.local と .env.sample は Git 管理対象
!.env.local
!.env.sample
```

---

## 9. リリースプロセス

### 9.1 バージョニング

[Semantic Versioning](https://semver.org/) に従う:

```
MAJOR.MINOR.PATCH

MAJOR: 破壊的変更
MINOR: 後方互換性のある機能追加
PATCH: 後方互換性のあるバグ修正
```

### 9.2 リリースフロー

```bash
# 1. main ブランチを最新化
git checkout main
git pull origin main

# 2. バージョンタグを作成
git tag -a v1.2.0 -m "Release v1.2.0"

# 3. タグをプッシュ
git push origin v1.2.0

# 4. GitHub Releases で公開
```

---

## 10. 困ったときは

- 質問がある場合は Issue を作成
- 開発環境の問題は [SETUP.md](./SETUP.md) を参照
- テストの書き方は [TESTING.md](./TESTING.md) を参照
- アーキテクチャの疑問は [ARCHITECTURE.md](./ARCHITECTURE.md) を参照

---

## 関連ドキュメント

- [SETUP.md](./SETUP.md) - 開発環境セットアップ
- [TESTING.md](./TESTING.md) - テスト戦略
- [BACKEND.md](./BACKEND.md) - バックエンド設計
- [FRONTEND.md](./FRONTEND.md) - フロントエンド設計
- [API.md](./API.md) - API設計

---

## 更新履歴

| 日付 | バージョン | 内容 |
|------|-----------|------|
| 2026-01-17 | 1.0.0 | 初版作成 |

# コーディング規約

> プロジェクト全体に適用される命名規則、フォーマット、エラーハンドリング方針を定義します。

---

## 1. 共通規則

### 1.1 ファイル・ディレクトリ命名

| 種類 | 規則 | 例 |
|------|------|-----|
| ディレクトリ | ケバブケース | `user-profile/`, `file-upload/` |
| ファイル（Go） | スネークケース | `file_handler.go`, `user_repository.go` |
| ファイル（TypeScript） | ケバブケース | `file-list.tsx`, `use-debounce.ts` |
| テストファイル | `*_test.go`, `*.test.ts` | `file_handler_test.go`, `file-list.test.tsx` |

### 1.2 コードフォーマット

| 言語 | ツール | 設定 |
|------|--------|------|
| Go | gofmt / goimports | デフォルト |
| TypeScript | Prettier | プロジェクト設定 |
| SQL | pg_format | デフォルト |

### 1.3 インポート順序

**Go:**
```go
import (
    // 1. 標準ライブラリ
    "context"
    "fmt"

    // 2. サードパーティ
    "github.com/labstack/echo/v4"

    // 3. 内部パッケージ
    "github.com/Hiro-mackay/gc-storage/internal/domain"
)
```

**TypeScript:**
```typescript
// 1. 外部ライブラリ
import { useQuery } from '@tanstack/react-query';

// 2. 内部モジュール（絶対パス）
import { Button } from '@/components/ui/button';

// 3. 相対インポート
import { FileItem } from './file-item';

// 4. 型インポート
import type { File } from '@/types';
```

---

## 2. Go コーディング規約

### 2.1 命名規則

| 種類 | 規則 | 例 |
|------|------|-----|
| パッケージ | 小文字、単数形 | `entity`, `handler`, `repository` |
| 構造体 | パスカルケース | `FileHandler`, `UserRepository` |
| インターフェース | パスカルケース | `FileRepository`, `UserService` |
| メソッド | パスカルケース | `GetByID`, `CreateFile` |
| 変数 | キャメルケース | `fileID`, `userEmail` |
| 定数 | パスカルケース or ALL_CAPS | `MaxFileSize`, `DEFAULT_TIMEOUT` |
| プライベート | 小文字始まり | `parseRequest`, `validateInput` |

### 2.2 エラーハンドリング

```go
// Good: エラーを適切にラップして返す
func (r *fileRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.File, error) {
    file, err := r.queries.GetFile(ctx, id)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, apperror.NewNotFoundError("file", id.String())
        }
        return nil, fmt.Errorf("failed to get file: %w", err)
    }
    return mapToEntity(file), nil
}

// Bad: エラーを握りつぶす
func (r *fileRepository) GetByID(ctx context.Context, id uuid.UUID) *entity.File {
    file, _ := r.queries.GetFile(ctx, id) // エラー無視は禁止
    return mapToEntity(file)
}
```

### 2.3 エラー種別

| エラー種別 | 対応 |
|-----------|------|
| ビジネスエラー | `apperror`パッケージで構造化エラーを返す |
| システムエラー | ログ出力後、汎用エラーに変換 |
| バリデーションエラー | フィールド単位でエラー詳細を返す |

### 2.4 コンテキスト使用方針

```go
// Good: 第一引数にcontext.Context
func (uc *FileUploadUseCase) Execute(ctx context.Context, input UploadInput) (*UploadOutput, error)

// Bad: contextを省略
func (uc *FileUploadUseCase) Execute(input UploadInput) (*UploadOutput, error)
```

- すべてのRepository/UseCaseメソッドは第一引数に`context.Context`を受け取る
- タイムアウト、キャンセル、トレーシング情報の伝播に使用
- 認証情報はContextではなく、明示的な引数として渡す

### 2.5 テスト

```go
// テスト関数の命名: Test<対象>_<シナリオ>
func TestFileRepository_GetByID_Success(t *testing.T) { ... }
func TestFileRepository_GetByID_NotFound(t *testing.T) { ... }

// テーブル駆動テストを推奨
func TestValidateFileName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid name", "document.pdf", false},
        {"empty name", "", true},
        {"too long", strings.Repeat("a", 256), true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateFileName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateFileName() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

## 3. TypeScript コーディング規約

### 3.1 命名規則

| 種類 | 規則 | 例 |
|------|------|-----|
| コンポーネント | パスカルケース | `FileList`, `UserProfile` |
| 関数 | キャメルケース | `handleClick`, `formatDate` |
| 変数 | キャメルケース | `fileName`, `isLoading` |
| 定数 | UPPER_SNAKE_CASE | `MAX_FILE_SIZE`, `API_BASE_URL` |
| 型/インターフェース | パスカルケース | `FileResponse`, `UserState` |
| フック | `use`プレフィックス | `useFileUpload`, `useDebounce` |

### 3.2 コンポーネント命名

| 種別 | 命名パターン | 例 |
|------|-------------|-----|
| ページ | `{Name}Page` | `FilesPage`, `SettingsPage` |
| レイアウト | `{Name}Layout` | `MainLayout`, `AuthLayout` |
| フィーチャー | `{Feature}{Type}` | `FileList`, `FileUploader` |
| 共通UI | `{Name}` | `Button`, `Dialog`, `Input` |

### 3.3 型定義

```typescript
// Good: 明示的な型定義
interface FileListProps {
  files: File[];
  onSelect: (file: File) => void;
  isLoading?: boolean;
}

function FileList({ files, onSelect, isLoading = false }: FileListProps) {
  // ...
}

// Bad: any型の使用
function FileList(props: any) { ... }
```

### 3.4 状態管理

| 状態の種類 | 管理方法 | 例 |
|-----------|---------|-----|
| サーバー状態 | TanStack Query | ファイル一覧、ユーザー情報 |
| URL状態 | TanStack Router | 現在のフォルダID、検索クエリ |
| ローカルUI状態 | useState | ダイアログ開閉、入力値 |
| グローバルUI状態 | Zustand | テーマ、サイドバー開閉、選択状態 |

### 3.5 Zustand使用ルール

```typescript
// Good: 必要な状態のみ購読（セレクター使用）
const viewMode = useUIStore((state) => state.viewMode);
const setViewMode = useUIStore((state) => state.setViewMode);

// Bad: Store全体を購読（不要な再レンダリング）
const store = useUIStore();
```

### 3.6 イベントハンドラー命名

```typescript
// Good: handle + 動詞 + 名詞
const handleFileSelect = (file: File) => { ... };
const handleFormSubmit = (e: FormEvent) => { ... };

// Bad: 曖昧な命名
const click = () => { ... };
const doSomething = () => { ... };
```

---

## 4. SQL コーディング規約

### 4.1 命名規則

| 種類 | 規則 | 例 |
|------|------|-----|
| テーブル | スネークケース、複数形 | `users`, `file_versions` |
| カラム | スネークケース | `created_at`, `file_name` |
| インデックス | `idx_{table}_{columns}` | `idx_files_folder_id` |
| 外部キー | `fk_{table}_{ref_table}` | `fk_files_users` |

### 4.2 クエリスタイル

```sql
-- Good: 読みやすいフォーマット
SELECT
    f.id,
    f.name,
    f.size,
    u.email AS owner_email
FROM files f
INNER JOIN users u ON u.id = f.owner_id
WHERE f.folder_id = $1
    AND f.deleted_at IS NULL
ORDER BY f.created_at DESC
LIMIT $2 OFFSET $3;

-- Bad: 1行に詰め込む
SELECT f.id, f.name, f.size, u.email AS owner_email FROM files f INNER JOIN users u ON u.id = f.owner_id WHERE f.folder_id = $1 AND f.deleted_at IS NULL ORDER BY f.created_at DESC LIMIT $2 OFFSET $3;
```

---

## 5. コメント規約

### 5.1 コメントの原則

```go
// Good: 「なぜ」を説明
// 24時間以上経過したpendingファイルは孤立データとみなし削除対象とする
func (s *cleanupService) findOrphanedFiles() { ... }

// Bad: 「何を」を説明（コードを読めばわかる）
// ファイルを取得する
func (r *repository) GetFile() { ... }
```

### 5.2 TODO/FIXMEコメント

```go
// TODO(username): 説明文 - issue#123
// FIXME(username): 説明文 - issue#456
```

---

## 6. Git コミットメッセージ

### 6.1 Conventional Commits

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### 6.2 Type一覧

| Type | 用途 |
|------|------|
| `feat` | 新機能 |
| `fix` | バグ修正 |
| `docs` | ドキュメント |
| `style` | フォーマット（機能変更なし） |
| `refactor` | リファクタリング |
| `test` | テスト追加・修正 |
| `chore` | ビルド、CI、依存関係更新 |

### 6.3 例

```
feat(files): add multipart upload support

- Implement chunked upload for files > 100MB
- Add progress tracking via WebSocket
- Update API documentation

Closes #123
```

---

## 関連ドキュメント

- [TECH_STACK.md](./TECH_STACK.md) - 技術スタック
- [TESTING.md](./TESTING.md) - テスト戦略
- [CONTRIBUTING.md](./CONTRIBUTING.md) - 開発プロセス

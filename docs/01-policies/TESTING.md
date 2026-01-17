# GC Storage テスト戦略

## 概要

本ドキュメントでは、GC Storageのテスト戦略、各種テストの実装方法、およびカバレッジ目標について説明します。

---

## 1. テストピラミッド

```
              ┌─────────────┐
              │    E2E      │  ← 少数、高コスト、遅い
              │   Tests     │     ユーザーフロー全体を検証
              ├─────────────┤
              │ Integration │  ← 中程度、複数コンポーネント連携
              │   Tests     │     API、DB、外部サービス連携
              ├─────────────┤
              │    Unit     │  ← 多数、低コスト、速い
              │   Tests     │     単一コンポーネントの動作検証
              └─────────────┘
```

| テスト種別 | 割合目安 | 実行速度 | 対象 |
|-----------|---------|---------|------|
| Unit | 70% | 高速 | 関数、メソッド単体 |
| Integration | 20% | 中速 | API、DB連携 |
| E2E | 10% | 低速 | ユーザーシナリオ |

---

## 2. カバレッジ目標

### 2.1 全体目標

| 対象 | Line Coverage | Branch Coverage |
|------|---------------|-----------------|
| バックエンド | 80%以上 | 70%以上 |
| フロントエンド | 70%以上 | 60%以上 |
| クリティカルパス | 90%以上 | 85%以上 |

### 2.2 クリティカルパス

以下の機能は特に高いカバレッジを維持:

- 認証・認可（auth, permission）
- ファイルアップロード・ダウンロード
- 権限チェック
- データ整合性に関わる処理

---

## 3. バックエンドテスト（Go）

### 3.1 テストツール

| ツール | 用途 |
|--------|------|
| `testing` | 標準テストパッケージ |
| `testify` | アサーション、モック |
| `gomock` | インターフェースモック生成 |
| `testcontainers-go` | 統合テスト用コンテナ |
| `httptest` | HTTPハンドラテスト |

### 3.2 ディレクトリ構成

```
backend/
├── internal/
│   ├── domain/
│   │   └── entity/
│   │       ├── file.go
│   │       └── file_test.go      # 同一ディレクトリ
│   ├── usecase/
│   │   └── file/
│   │       ├── upload.go
│   │       └── upload_test.go
│   └── interface/
│       └── handler/
│           ├── file_handler.go
│           └── file_handler_test.go
├── tests/
│   ├── integration/              # 統合テスト
│   │   ├── file_api_test.go
│   │   └── auth_api_test.go
│   └── testutil/                 # テストユーティリティ
│       ├── fixtures.go
│       └── helpers.go
```

### 3.3 Unit テスト

**ドメイン層のテスト例:**

```go
// internal/domain/entity/file_test.go
package entity_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "gc-storage/internal/domain/entity"
)

func TestNewFile(t *testing.T) {
    tests := []struct {
        name     string
        input    entity.FileInput
        wantErr  bool
        errMsg   string
    }{
        {
            name: "valid file",
            input: entity.FileInput{
                Name:     "document.pdf",
                Size:     1024,
                MimeType: "application/pdf",
            },
            wantErr: false,
        },
        {
            name: "empty name",
            input: entity.FileInput{
                Name:     "",
                Size:     1024,
                MimeType: "application/pdf",
            },
            wantErr: true,
            errMsg:  "file name is required",
        },
        {
            name: "invalid characters in name",
            input: entity.FileInput{
                Name:     "doc/test.pdf",
                Size:     1024,
                MimeType: "application/pdf",
            },
            wantErr: true,
            errMsg:  "invalid characters in file name",
        },
        {
            name: "exceeds max size",
            input: entity.FileInput{
                Name:     "large.bin",
                Size:     6 * 1024 * 1024 * 1024, // 6GB
                MimeType: "application/octet-stream",
            },
            wantErr: true,
            errMsg:  "file size exceeds maximum",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            file, err := entity.NewFile(tt.input)

            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
                assert.Nil(t, file)
            } else {
                require.NoError(t, err)
                assert.NotNil(t, file)
                assert.Equal(t, tt.input.Name, file.Name)
            }
        })
    }
}
```

**ユースケース層のテスト例（モック使用）:**

```go
// internal/usecase/file/upload_test.go
package file_test

import (
    "context"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "gc-storage/internal/domain/entity"
    "gc-storage/internal/usecase/file"
    "gc-storage/tests/testutil/mocks"
)

func TestUploadUseCase_Execute(t *testing.T) {
    ctx := context.Background()
    userID := uuid.New()
    folderID := uuid.New()

    t.Run("successful upload initiation", func(t *testing.T) {
        // モックの準備
        mockFileRepo := mocks.NewMockFileRepository(t)
        mockStorageService := mocks.NewMockStorageService(t)
        mockPermissionService := mocks.NewMockPermissionService(t)

        // 期待する呼び出しを設定
        mockPermissionService.On(
            "HasPermission",
            ctx,
            userID,
            "folder",
            folderID,
            entity.PermFileWrite,
        ).Return(true, nil)

        mockFileRepo.On(
            "Create",
            ctx,
            mock.AnythingOfType("*entity.File"),
        ).Return(nil)

        mockStorageService.On(
            "GenerateUploadURL",
            ctx,
            mock.AnythingOfType("string"),
            15*time.Minute,
        ).Return("https://minio.example.com/upload?signature=xxx", nil)

        // ユースケースの実行
        uc := file.NewUploadUseCase(
            mockFileRepo,
            mockStorageService,
            mockPermissionService,
        )

        output, err := uc.Execute(ctx, file.UploadInput{
            UserID:   userID,
            FolderID: &folderID,
            Name:     "document.pdf",
            Size:     1024,
            MimeType: "application/pdf",
        })

        // アサーション
        require.NoError(t, err)
        assert.NotNil(t, output)
        assert.NotEmpty(t, output.FileID)
        assert.NotEmpty(t, output.UploadURL)

        // モックの検証
        mockPermissionService.AssertExpectations(t)
        mockFileRepo.AssertExpectations(t)
        mockStorageService.AssertExpectations(t)
    })

    t.Run("permission denied", func(t *testing.T) {
        mockFileRepo := mocks.NewMockFileRepository(t)
        mockStorageService := mocks.NewMockStorageService(t)
        mockPermissionService := mocks.NewMockPermissionService(t)

        mockPermissionService.On(
            "HasPermission",
            ctx,
            userID,
            "folder",
            folderID,
            entity.PermFileWrite,
        ).Return(false, nil)

        uc := file.NewUploadUseCase(
            mockFileRepo,
            mockStorageService,
            mockPermissionService,
        )

        output, err := uc.Execute(ctx, file.UploadInput{
            UserID:   userID,
            FolderID: &folderID,
            Name:     "document.pdf",
            Size:     1024,
            MimeType: "application/pdf",
        })

        require.Error(t, err)
        assert.Nil(t, output)
        assert.Contains(t, err.Error(), "forbidden")
    })
}
```

### 3.4 Integration テスト

**testcontainers-go を使用したDBテスト:**

```go
// tests/integration/file_api_test.go
package integration_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"

    "gc-storage/internal/infrastructure/database"
    "gc-storage/tests/testutil"
)

type FileAPITestSuite struct {
    suite.Suite
    ctx             context.Context
    pgContainer     *postgres.PostgresContainer
    redisContainer  testcontainers.Container
    server          *httptest.Server
    client          *http.Client
    authToken       string
}

func (s *FileAPITestSuite) SetupSuite() {
    s.ctx = context.Background()

    // PostgreSQL コンテナ起動
    pgContainer, err := postgres.Run(s.ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("gc_storage_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    require.NoError(s.T(), err)
    s.pgContainer = pgContainer

    // Redis コンテナ起動
    s.redisContainer, err = testutil.StartRedisContainer(s.ctx)
    require.NoError(s.T(), err)

    // マイグレーション実行
    connStr, _ := pgContainer.ConnectionString(s.ctx)
    err = database.RunMigrations(connStr, "../../migrations")
    require.NoError(s.T(), err)

    // テストサーバー起動
    s.server = testutil.NewTestServer(s.ctx, connStr, s.redisContainer)
    s.client = s.server.Client()

    // テストユーザーでログイン
    s.authToken, _ = testutil.CreateTestUserAndLogin(s.ctx, s.client, s.server.URL)
}

func (s *FileAPITestSuite) TearDownSuite() {
    s.server.Close()
    s.pgContainer.Terminate(s.ctx)
    s.redisContainer.Terminate(s.ctx)
}

func (s *FileAPITestSuite) TestCreateFile() {
    // ファイルアップロード開始
    resp, err := testutil.PostJSON(s.client, s.server.URL+"/api/v1/files/upload", map[string]interface{}{
        "name":      "test.pdf",
        "size":      1024,
        "mime_type": "application/pdf",
    }, s.authToken)

    require.NoError(s.T(), err)
    assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

    var result map[string]interface{}
    testutil.ParseJSON(resp, &result)

    assert.NotEmpty(s.T(), result["data"].(map[string]interface{})["file_id"])
    assert.NotEmpty(s.T(), result["data"].(map[string]interface{})["upload_url"])
}

func (s *FileAPITestSuite) TestGetFile_NotFound() {
    resp, err := testutil.Get(s.client, s.server.URL+"/api/v1/files/00000000-0000-0000-0000-000000000000", s.authToken)

    require.NoError(s.T(), err)
    assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

func (s *FileAPITestSuite) TestGetFile_Forbidden() {
    // 別ユーザーのファイルにアクセス
    otherUserToken, _ := testutil.CreateTestUserAndLogin(s.ctx, s.client, s.server.URL)
    fileID := testutil.CreateTestFile(s.ctx, s.client, s.server.URL, s.authToken)

    resp, err := testutil.Get(s.client, s.server.URL+"/api/v1/files/"+fileID, otherUserToken)

    require.NoError(s.T(), err)
    assert.Equal(s.T(), http.StatusForbidden, resp.StatusCode)
}

func TestFileAPITestSuite(t *testing.T) {
    suite.Run(t, new(FileAPITestSuite))
}
```

### 3.5 テスト実行コマンド

```bash
# 全テスト実行
task backend:test

# 詳細出力付き
go test -v ./...

# カバレッジ付き
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 特定パッケージのみ
go test -v ./internal/usecase/file/...

# 統合テストのみ
go test -v -tags=integration ./tests/integration/...

# 特定テストのみ
go test -v -run TestUploadUseCase ./internal/usecase/file/
```

### 3.6 モック生成

```bash
# gomock によるモック生成
mockgen -source=internal/domain/repository/file_repository.go \
        -destination=tests/testutil/mocks/file_repository_mock.go \
        -package=mocks

# Task コマンド
task backend:generate-mocks
```

---

## 4. フロントエンドテスト（React）

### 4.1 テストツール

| ツール | 用途 |
|--------|------|
| Vitest | テストランナー |
| Testing Library | コンポーネントテスト |
| MSW | APIモック |
| Playwright | E2Eテスト |

### 4.2 ディレクトリ構成

```
frontend/
├── src/
│   ├── components/
│   │   └── files/
│   │       ├── FileItem.tsx
│   │       └── FileItem.test.tsx   # コロケーション
│   ├── hooks/
│   │   ├── useDebounce.ts
│   │   └── useDebounce.test.ts
│   └── lib/
│       └── utils/
│           ├── format.ts
│           └── format.test.ts
├── tests/
│   ├── e2e/                        # E2Eテスト
│   │   ├── auth.spec.ts
│   │   └── file-upload.spec.ts
│   └── mocks/                      # MSWハンドラー
│       ├── handlers.ts
│       └── server.ts
```

### 4.3 Unit テスト（Vitest）

**ユーティリティ関数のテスト:**

```typescript
// src/lib/utils/format.test.ts
import { describe, it, expect } from 'vitest';
import { formatFileSize, formatDate } from './format';

describe('formatFileSize', () => {
  it('formats bytes correctly', () => {
    expect(formatFileSize(0)).toBe('0 B');
    expect(formatFileSize(500)).toBe('500 B');
  });

  it('formats kilobytes correctly', () => {
    expect(formatFileSize(1024)).toBe('1.00 KB');
    expect(formatFileSize(1536)).toBe('1.50 KB');
  });

  it('formats megabytes correctly', () => {
    expect(formatFileSize(1048576)).toBe('1.00 MB');
    expect(formatFileSize(5242880)).toBe('5.00 MB');
  });

  it('formats gigabytes correctly', () => {
    expect(formatFileSize(1073741824)).toBe('1.00 GB');
  });
});

describe('formatDate', () => {
  it('formats ISO date string', () => {
    const date = '2024-01-15T10:30:00Z';
    expect(formatDate(date)).toBe('2024/01/15 10:30');
  });

  it('handles invalid date', () => {
    expect(formatDate('invalid')).toBe('Invalid Date');
  });
});
```

**カスタムフックのテスト:**

```typescript
// src/hooks/useDebounce.test.ts
import { describe, it, expect, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useDebounce } from './useDebounce';

describe('useDebounce', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns initial value immediately', () => {
    const { result } = renderHook(() => useDebounce('test', 500));
    expect(result.current).toBe('test');
  });

  it('updates value after delay', () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebounce(value, delay),
      { initialProps: { value: 'initial', delay: 500 } }
    );

    expect(result.current).toBe('initial');

    rerender({ value: 'updated', delay: 500 });
    expect(result.current).toBe('initial');

    act(() => {
      vi.advanceTimersByTime(500);
    });

    expect(result.current).toBe('updated');
  });

  it('resets timer on rapid changes', () => {
    const { result, rerender } = renderHook(
      ({ value }) => useDebounce(value, 500),
      { initialProps: { value: 'a' } }
    );

    rerender({ value: 'b' });
    act(() => vi.advanceTimersByTime(200));

    rerender({ value: 'c' });
    act(() => vi.advanceTimersByTime(200));

    // まだ更新されていない
    expect(result.current).toBe('a');

    act(() => vi.advanceTimersByTime(300));
    // 最後の値に更新
    expect(result.current).toBe('c');
  });
});
```

### 4.4 コンポーネントテスト

```typescript
// src/components/files/FileItem.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { FileItem } from './FileItem';

const mockFile = {
  id: '123',
  name: 'document.pdf',
  size: 1048576,
  mimeType: 'application/pdf',
  createdAt: '2024-01-15T10:30:00Z',
};

describe('FileItem', () => {
  it('renders file information correctly', () => {
    render(<FileItem file={mockFile} />);

    expect(screen.getByText('document.pdf')).toBeInTheDocument();
    expect(screen.getByText('1.00 MB')).toBeInTheDocument();
    expect(screen.getByText('PDF')).toBeInTheDocument();
  });

  it('calls onSelect when clicked', () => {
    const handleSelect = vi.fn();
    render(<FileItem file={mockFile} onSelect={handleSelect} />);

    fireEvent.click(screen.getByRole('listitem'));

    expect(handleSelect).toHaveBeenCalledWith(mockFile.id);
  });

  it('calls onDelete when delete button clicked', () => {
    const handleDelete = vi.fn();
    render(<FileItem file={mockFile} onDelete={handleDelete} />);

    fireEvent.click(screen.getByRole('button', { name: /delete/i }));

    expect(handleDelete).toHaveBeenCalledWith(mockFile.id);
  });

  it('shows selected state when isSelected is true', () => {
    render(<FileItem file={mockFile} isSelected={true} />);

    expect(screen.getByRole('listitem')).toHaveClass('selected');
  });

  it('renders file icon based on mime type', () => {
    render(<FileItem file={mockFile} />);

    expect(screen.getByTestId('file-icon-pdf')).toBeInTheDocument();
  });
});
```

### 4.5 MSW によるAPIモック

```typescript
// tests/mocks/handlers.ts
import { http, HttpResponse } from 'msw';

export const handlers = [
  // ファイル一覧取得
  http.get('/api/v1/files', () => {
    return HttpResponse.json({
      data: [
        {
          id: '1',
          name: 'file1.pdf',
          size: 1024,
          mimeType: 'application/pdf',
          createdAt: '2024-01-15T10:00:00Z',
        },
        {
          id: '2',
          name: 'file2.png',
          size: 2048,
          mimeType: 'image/png',
          createdAt: '2024-01-15T11:00:00Z',
        },
      ],
      meta: {
        pagination: {
          page: 1,
          perPage: 50,
          totalItems: 2,
          totalPages: 1,
        },
      },
    });
  }),

  // ファイルアップロード開始
  http.post('/api/v1/files/upload', async ({ request }) => {
    const body = await request.json();
    return HttpResponse.json({
      data: {
        fileId: 'new-file-id',
        uploadUrl: 'https://minio.example.com/upload',
        expiresAt: new Date(Date.now() + 15 * 60 * 1000).toISOString(),
      },
      meta: null,
    });
  }),

  // ファイル削除
  http.delete('/api/v1/files/:id', ({ params }) => {
    return HttpResponse.json({
      data: null,
      meta: { message: 'ファイルを削除しました' },
    });
  }),
];
```

```typescript
// tests/mocks/server.ts
import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);
```

```typescript
// vitest.setup.ts
import { beforeAll, afterEach, afterAll } from 'vitest';
import { server } from './tests/mocks/server';

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
```

### 4.6 TanStack Query のテスト

```typescript
// src/features/files/hooks/useFiles.test.tsx
import { describe, it, expect } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useFiles } from './useFiles';

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
};

describe('useFiles', () => {
  it('fetches files successfully', async () => {
    const { result } = renderHook(() => useFiles({ folderId: null }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data).toHaveLength(2);
    expect(result.current.data[0].name).toBe('file1.pdf');
  });

  it('handles loading state', () => {
    const { result } = renderHook(() => useFiles({ folderId: null }), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
  });
});
```

### 4.7 テスト実行コマンド

```bash
# 全テスト実行
pnpm test

# ウォッチモード
pnpm test:watch

# カバレッジ
pnpm test:coverage

# UIモード
pnpm test:ui

# 特定ファイル
pnpm test FileItem
```

---

## 5. E2E テスト（Playwright）

### 5.1 セットアップ

```bash
# Playwright のインストール
pnpm add -D @playwright/test

# ブラウザのインストール
npx playwright install
```

### 5.2 E2E テスト例

```typescript
// tests/e2e/file-upload.spec.ts
import { test, expect } from '@playwright/test';

test.describe('File Upload', () => {
  test.beforeEach(async ({ page }) => {
    // ログイン
    await page.goto('/login');
    await page.fill('[name="email"]', 'test@example.com');
    await page.fill('[name="password"]', 'password123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/files');
  });

  test('should upload a file successfully', async ({ page }) => {
    // ファイル選択
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('tests/fixtures/sample.pdf');

    // アップロード進捗を確認
    await expect(page.locator('[data-testid="upload-progress"]')).toBeVisible();

    // 完了を待機
    await expect(page.locator('[data-testid="upload-success"]')).toBeVisible({
      timeout: 30000,
    });

    // ファイル一覧に表示されていることを確認
    await expect(page.locator('text=sample.pdf')).toBeVisible();
  });

  test('should show error for oversized file', async ({ page }) => {
    // 大きすぎるファイルを選択（モック）
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('tests/fixtures/large-file.bin');

    // エラーメッセージを確認
    await expect(
      page.locator('text=ファイルサイズが上限を超えています')
    ).toBeVisible();
  });

  test('should delete a file', async ({ page }) => {
    // ファイルを選択
    await page.click('[data-testid="file-item"]:first-child');

    // 削除ボタンをクリック
    await page.click('[data-testid="delete-button"]');

    // 確認ダイアログで削除を確定
    await page.click('[data-testid="confirm-delete"]');

    // ファイルが消えたことを確認
    await expect(page.locator('[data-testid="file-item"]')).toHaveCount(0);
  });
});
```

```typescript
// tests/e2e/auth.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test('should login successfully', async ({ page }) => {
    await page.goto('/login');

    await page.fill('[name="email"]', 'test@example.com');
    await page.fill('[name="password"]', 'password123');
    await page.click('button[type="submit"]');

    await page.waitForURL('/files');
    await expect(page).toHaveURL('/files');
  });

  test('should show error for invalid credentials', async ({ page }) => {
    await page.goto('/login');

    await page.fill('[name="email"]', 'wrong@example.com');
    await page.fill('[name="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');

    await expect(page.locator('text=認証に失敗しました')).toBeVisible();
  });

  test('should logout successfully', async ({ page }) => {
    // ログイン済み状態から開始
    await page.goto('/login');
    await page.fill('[name="email"]', 'test@example.com');
    await page.fill('[name="password"]', 'password123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/files');

    // ログアウト
    await page.click('[data-testid="user-menu"]');
    await page.click('[data-testid="logout-button"]');

    await page.waitForURL('/login');
    await expect(page).toHaveURL('/login');
  });
});
```

### 5.3 Playwright 設定

```typescript
// playwright.config.ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
  webServer: {
    command: 'pnpm dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
});
```

### 5.4 E2E テスト実行

```bash
# 全ブラウザで実行
npx playwright test

# 特定ブラウザのみ
npx playwright test --project=chromium

# UIモード
npx playwright test --ui

# デバッグモード
npx playwright test --debug

# レポート表示
npx playwright show-report
```

---

## 6. CI/CD でのテスト

### 6.1 GitHub Actions 設定

```yaml
# .github/workflows/test.yml
name: Test

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  backend-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: gc_storage_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install dependencies
        working-directory: ./backend
        run: go mod download

      - name: Run tests
        working-directory: ./backend
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./backend/coverage.out

  frontend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: pnpm/action-setup@v2
        with:
          version: 8

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'pnpm'
          cache-dependency-path: frontend/pnpm-lock.yaml

      - name: Install dependencies
        working-directory: ./frontend
        run: pnpm install

      - name: Run tests
        working-directory: ./frontend
        run: pnpm test:coverage

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./frontend/coverage/lcov.info

  e2e-test:
    runs-on: ubuntu-latest
    needs: [backend-test, frontend-test]
    steps:
      - uses: actions/checkout@v4

      - uses: pnpm/action-setup@v2
        with:
          version: 8

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'pnpm'
          cache-dependency-path: frontend/pnpm-lock.yaml

      - name: Install dependencies
        working-directory: ./frontend
        run: pnpm install

      - name: Install Playwright
        working-directory: ./frontend
        run: npx playwright install --with-deps

      - name: Run E2E tests
        working-directory: ./frontend
        run: npx playwright test

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: playwright-report
          path: frontend/playwright-report/
```

---

## 7. テストのベストプラクティス

### 7.1 AAA パターン

```go
func TestExample(t *testing.T) {
    // Arrange - 準備
    input := "test"
    expected := "TEST"

    // Act - 実行
    result := strings.ToUpper(input)

    // Assert - 検証
    assert.Equal(t, expected, result)
}
```

### 7.2 テスト命名規則

```go
// Go
func TestFunctionName_Scenario_ExpectedBehavior(t *testing.T)
func TestUploadFile_ValidInput_ReturnsFileID(t *testing.T)
func TestUploadFile_InvalidSize_ReturnsError(t *testing.T)
```

```typescript
// TypeScript
describe('functionName', () => {
  it('should return expected result when condition', () => {});
});
```

### 7.3 テストデータ管理

- フィクスチャは `tests/fixtures/` に配置
- ファクトリ関数でテストデータを生成
- 機密データはモック化

---

## 関連ドキュメント

- [SETUP.md](./SETUP.md) - 開発環境セットアップ
- [CONTRIBUTING.md](./CONTRIBUTING.md) - 開発者ガイド
- [BACKEND.md](./BACKEND.md) - バックエンド設計
- [FRONTEND.md](./FRONTEND.md) - フロントエンド設計

---

## 更新履歴

| 日付 | バージョン | 内容 |
|------|-----------|------|
| 2026-01-17 | 1.0.0 | 初版作成 |

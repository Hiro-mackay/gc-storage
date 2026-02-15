# GC Storage フロントエンド設計書

## 概要

本ドキュメントでは、GC StorageのReactフロントエンドに関する**設計原則、ライブラリルール、状態管理戦略**について説明します。コンポーネント実装の詳細はソースコードを参照してください。

---

## 1. 技術スタック

| カテゴリ | 技術 | バージョン |
|---------|------|-----------|
| Framework | React | 18.x |
| Build Tool | Vite | 5.x |
| Routing | TanStack Router | 1.x |
| Data Fetching | TanStack Query | 5.x |
| State Management | Zustand | 4.x |
| UI Components | shadcn/ui | latest |
| Styling | Tailwind CSS | 3.x |
| Form Validation | Zod | latest |
| HTTP Client | openapi-fetch | 0.x |
| Type Generation | openapi-typescript | 7.x |
| Testing | Vitest + Testing Library | latest |

**補足:**
- openapi-fetch: OpenAPIスキーマから型推論するfetch wrapper
- openapi-typescript: OpenAPIスキーマからTypeScript型定義を自動生成

**使用しないライブラリ:**
- React Hook Form（shadcn/ui form依存として使用）
- Redux（TanStack Query + Zustandで十分なため）
- React Context（グローバル状態管理にはZustandを使用）

---

## 2. ディレクトリ構成

```
frontend/
├── src/
│   ├── app/
│   │   ├── routes/               # TanStack Router ルート定義
│   │   ├── router.tsx            # ルーター設定
│   │   └── App.tsx               # アプリケーションルート
│   ├── components/
│   │   ├── ui/                   # shadcn/ui コンポーネント
│   │   ├── layout/               # レイアウトコンポーネント
│   │   └── common/               # 共通コンポーネント
│   ├── features/                 # 機能別モジュール
│   │   ├── auth/
│   │   ├── files/
│   │   ├── groups/
│   │   └── search/
│   ├── stores/                   # Zustand Stores
│   │   ├── uiStore.ts
│   │   ├── selectionStore.ts
│   │   └── uploadStore.ts
│   ├── hooks/                    # 共通カスタムフック
│   ├── lib/
│   │   ├── api/                  # APIクライアント
│   │   │   ├── schema.d.ts       # OpenAPI型定義（自動生成）
│   │   │   └── client.ts         # openapi-fetch クライアント
│   │   └── utils/                # ユーティリティ
│   ├── types/                    # 型定義
│   └── styles/                   # グローバルスタイル
```

---

## 3. 状態管理戦略

### 3.1 状態の分類と管理方法

| 状態の種類 | 管理方法 | 例 |
|-----------|---------|-----|
| サーバー状態 | TanStack Query | ファイル一覧、ユーザー情報 |
| URL状態 | TanStack Router | 現在のフォルダID、検索クエリ |
| ローカルUI状態 | useState | ダイアログ開閉、入力値 |
| グローバルUI状態 | Zustand | テーマ、サイドバー開閉、選択状態 |

### 3.2 TanStack Query + Zustand の役割分担

**原則:**
- サーバーから取得するデータは**100% TanStack Query**で管理
- グローバルなクライアント状態は**Zustand**で管理
- ローカル状態は**useState**を最小限に使用

**役割分担:**

| 状態 | 管理方法 | 理由 |
|------|---------|------|
| APIデータ | TanStack Query | キャッシュ、再取得、無効化を自動管理 |
| UI設定 | Zustand (persist) | テーマ、表示設定など永続化が必要 |
| 一時的なUI状態 | Zustand | 選択状態、モーダル状態など複数コンポーネントで共有 |
| コンポーネントローカル状態 | useState | 他で使わない入力値など |

### 3.3 Query Key設計

```typescript
// features/files/api/queries.ts

export const fileKeys = {
  all: ['files'] as const,
  lists: () => [...fileKeys.all, 'list'] as const,
  list: (folderId: string | null, options: ListOptions) =>
    [...fileKeys.lists(), folderId, options] as const,
  details: () => [...fileKeys.all, 'detail'] as const,
  detail: (id: string) => [...fileKeys.details(), id] as const,
};

export const folderKeys = {
  all: ['folders'] as const,
  lists: () => [...folderKeys.all, 'list'] as const,
  list: (parentId: string | null) => [...folderKeys.lists(), parentId] as const,
  path: (id: string) => [...folderKeys.all, 'path', id] as const,
};
```

**命名規則:**
- `{entity}Keys.all`: すべてのキャッシュを無効化する際に使用
- `{entity}Keys.lists()`: リスト系キャッシュの親キー
- `{entity}Keys.list(params)`: 特定パラメータのリストキャッシュ
- `{entity}Keys.detail(id)`: 詳細キャッシュ

### 3.4 Zustand Store設計

**ディレクトリ構成:**
```
src/stores/
├── uiStore.ts          # UI状態（テーマ、サイドバー、表示モード）
├── selectionStore.ts   # ファイル選択状態
└── uploadStore.ts      # アップロード状態
```

**UI Store例:**

```typescript
// stores/uiStore.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

type ViewMode = 'list' | 'grid';
type Theme = 'light' | 'dark' | 'system';

interface UIState {
  sidebarOpen: boolean;
  viewMode: ViewMode;
  theme: Theme;
}

interface UIActions {
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setViewMode: (mode: ViewMode) => void;
  setTheme: (theme: Theme) => void;
}

export const useUIStore = create<UIState & UIActions>()(
  persist(
    (set) => ({
      sidebarOpen: true,
      viewMode: 'list',
      theme: 'system',

      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setViewMode: (mode) => set({ viewMode: mode }),
      setTheme: (theme) => set({ theme }),
    }),
    {
      name: 'ui-storage',
      partialize: (state) => ({
        viewMode: state.viewMode,
        theme: state.theme,
      }),
    }
  )
);
```

**Selection Store例:**

```typescript
// stores/selectionStore.ts
import { create } from 'zustand';

interface SelectionState {
  selectedIds: string[];
}

interface SelectionActions {
  select: (id: string, multi?: boolean) => void;
  deselect: (id: string) => void;
  clear: () => void;
  selectAll: (ids: string[]) => void;
}

export const useSelectionStore = create<SelectionState & SelectionActions>((set) => ({
  selectedIds: [],

  select: (id, multi = false) =>
    set((state) => ({
      selectedIds: multi
        ? state.selectedIds.includes(id)
          ? state.selectedIds.filter((i) => i !== id)
          : [...state.selectedIds, id]
        : [id],
    })),

  deselect: (id) =>
    set((state) => ({
      selectedIds: state.selectedIds.filter((i) => i !== id),
    })),

  clear: () => set({ selectedIds: [] }),

  selectAll: (ids) => set({ selectedIds: ids }),
}));
```

### 3.5 Zustand使用ルール

| ルール | 説明 |
|--------|------|
| Store分割 | 機能ごとにStoreを分割（1ファイル1Store） |
| 永続化 | 必要な状態のみ`persist`ミドルウェアで永続化 |
| セレクター使用 | 必要な状態のみ購読してパフォーマンス最適化 |
| アクションのコロケーション | 状態と操作を同じStoreに定義 |

**セレクター使用例:**

```typescript
// Good: 必要な状態のみ購読
const viewMode = useUIStore((state) => state.viewMode);
const setViewMode = useUIStore((state) => state.setViewMode);

// Bad: Store全体を購読（不要な再レンダリングが発生）
const store = useUIStore();
```

---

## 4. データフェッチ設計

### 4.1 API呼び出しパターン（openapi-fetch）

APIクライアントは `openapi-fetch` を使用し、OpenAPIスキーマから自動生成された型定義による型安全なAPI呼び出しを行います。

```typescript
import { api } from "@/lib/api/client";

// 型安全なAPI呼び出し
const { data, error } = await api.GET("/folders/{id}/contents", {
  params: { path: { id: folderId } },
});
```

パス・パラメータ・リクエストボディ・レスポンスがすべてOpenAPIスキーマから型推論されるため、コンパイル時にAPIの不整合を検出できます。

### 4.2 TanStack Router + Query連携

```
Route loader (prefetchQuery)
       ↓
Page Component (useSuspenseQuery)
       ↓
Child Component (useQuery - cached)
```

### 4.3 ルートローダーでのプリフェッチ

```typescript
// app/routes/_authenticated.files.$folderId.tsx

export const Route = createFileRoute('/_authenticated/files/$folderId')({
  loader: async ({ context, params }) => {
    const { queryClient } = context;

    // データをプリフェッチ（キャッシュに格納）
    await queryClient.ensureQueryData({
      queryKey: fileKeys.list(params.folderId, defaultOptions),
      queryFn: async () => {
        const { data } = await api.GET("/folders/{id}/contents", {
          params: { path: { id: params.folderId } },
        });
        return data;
      },
    });

    // パンくず用のパスもプリフェッチ
    await queryClient.ensureQueryData({
      queryKey: folderKeys.path(params.folderId),
      queryFn: async () => {
        const { data } = await api.GET("/folders/{id}/ancestors", {
          params: { path: { id: params.folderId } },
        });
        return data;
      },
    });
  },
});
```

### 4.4 ページコンポーネントでの取得

```typescript
// ページコンポーネント: useSuspenseQueryでSuspense対応
function FolderPage() {
  const { folderId } = Route.useParams();

  const { data } = useSuspenseQuery({
    queryKey: fileKeys.list(folderId, defaultOptions),
    queryFn: async () => {
      const { data } = await api.GET("/folders/{id}/contents", {
        params: { path: { id: folderId } },
      });
      return data;
    },
  });

  return <FileList files={data.files} folders={data.folders} />;
}
```

### 4.5 子コンポーネントでの取得

```typescript
// 子コンポーネント: useQueryでキャッシュ済みデータを取得
function FileItem({ fileId }: { fileId: string }) {
  // 既にキャッシュされていれば即座に返る
  const { data } = useQuery({
    queryKey: fileKeys.detail(fileId),
    queryFn: async () => {
      const { data } = await api.GET("/files/{id}", {
        params: { path: { id: fileId } },
      });
      return data;
    },
    staleTime: 30 * 1000,
  });

  // ...
}
```

### 4.6 データ更新時のキャッシュ無効化

```typescript
// ファイルアップロード完了時
const uploadMutation = useMutation({
  mutationFn: filesApi.completeUpload,
  onSuccess: (_, { folderId }) => {
    // 該当フォルダのリストキャッシュを無効化
    queryClient.invalidateQueries({
      queryKey: fileKeys.list(folderId, {}),
    });
  },
});

// ファイル削除時
const deleteMutation = useMutation({
  mutationFn: filesApi.deleteFile,
  onSuccess: (_, fileId) => {
    // リストキャッシュを無効化
    queryClient.invalidateQueries({ queryKey: fileKeys.lists() });
    // 詳細キャッシュを削除
    queryClient.removeQueries({ queryKey: fileKeys.detail(fileId) });
  },
});
```

---

## 5. フォーム管理

### 5.1 基本方針

- **React Hook Formは使用しない**
- **shadcn/ui Fieldコンポーネント**に準拠
- バリデーションは**Zod**で定義
- フォーム状態は**useState**で管理

### 5.2 フォーム実装パターン

```typescript
// features/auth/components/LoginForm.tsx

import { useState } from 'react';
import { z } from 'zod';
import { Field, FieldLabel, FieldControl, FieldError } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

const loginSchema = z.object({
  email: z.string().email('有効なメールアドレスを入力してください'),
  password: z.string().min(8, 'パスワードは8文字以上必要です'),
});

type LoginFormValues = z.infer<typeof loginSchema>;

export function LoginForm({ onSubmit }: { onSubmit: (values: LoginFormValues) => void }) {
  const [values, setValues] = useState<LoginFormValues>({ email: '', password: '' });
  const [errors, setErrors] = useState<Partial<Record<keyof LoginFormValues, string>>>({});

  const handleChange = (field: keyof LoginFormValues) => (
    e: React.ChangeEvent<HTMLInputElement>
  ) => {
    setValues((prev) => ({ ...prev, [field]: e.target.value }));
    // フィールド変更時にエラーをクリア
    setErrors((prev) => ({ ...prev, [field]: undefined }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const result = loginSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: typeof errors = {};
      result.error.issues.forEach((issue) => {
        const field = issue.path[0] as keyof LoginFormValues;
        fieldErrors[field] = issue.message;
      });
      setErrors(fieldErrors);
      return;
    }

    onSubmit(result.data);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <Field>
        <FieldLabel>メールアドレス</FieldLabel>
        <FieldControl>
          <Input
            type="email"
            value={values.email}
            onChange={handleChange('email')}
            placeholder="email@example.com"
          />
        </FieldControl>
        {errors.email && <FieldError>{errors.email}</FieldError>}
      </Field>

      <Field>
        <FieldLabel>パスワード</FieldLabel>
        <FieldControl>
          <Input
            type="password"
            value={values.password}
            onChange={handleChange('password')}
          />
        </FieldControl>
        {errors.password && <FieldError>{errors.password}</FieldError>}
      </Field>

      <Button type="submit" className="w-full">ログイン</Button>
    </form>
  );
}
```

### 5.3 バリデーションルール

- スキーマは**Zod**で定義
- エラーメッセージは日本語で記述
- 必須チェック、形式チェック、長さチェックを適切に設定

---

## 6. コンポーネント設計原則

### 6.1 Presentational / Container パターン

| 種別 | 責務 | 例 |
|------|------|-----|
| Presentational | UI表示のみ、propsで受け取る | FileItem, Button |
| Container | データ取得・ロジック、子に渡す | FileListContainer |

### 6.2 Composition パターン

Props Drillingを避けるために、Compositionパターンを活用:

```typescript
// Bad: Props Drilling
<FileManager
  files={files}
  onSelect={handleSelect}
  onDelete={handleDelete}
  onRename={handleRename}
  selectedFiles={selectedFiles}
  viewMode={viewMode}
/>

// Good: Composition
<FileManager>
  <FileManager.Toolbar>
    <ViewToggle />
    <SortSelect />
  </FileManager.Toolbar>
  <FileManager.List>
    {files.map((file) => (
      <FileManager.Item key={file.id} file={file} />
    ))}
  </FileManager.List>
</FileManager>
```

### 6.3 カスタムフックによるロジック分離

グローバル状態はZustandで管理し、カスタムフックはローカルなロジック再利用に使用:

```typescript
// hooks/useDebounce.ts
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => clearTimeout(handler);
  }, [value, delay]);

  return debouncedValue;
}
```

```typescript
// hooks/useFileDownload.ts
export function useFileDownload() {
  const [isDownloading, setIsDownloading] = useState(false);

  const download = async (fileId: string, fileName: string) => {
    setIsDownloading(true);
    try {
      const { url } = await filesApi.getDownloadUrl(fileId);
      const link = document.createElement('a');
      link.href = url;
      link.download = fileName;
      link.click();
    } finally {
      setIsDownloading(false);
    }
  };

  return { download, isDownloading };
}
```

### 6.4 コンポーネント命名規則

| 種別 | 命名 | 例 |
|------|------|-----|
| ページ | `{Name}Page` | `FilesPage`, `SettingsPage` |
| レイアウト | `{Name}Layout` | `MainLayout`, `AuthLayout` |
| フィーチャー | `{Feature}{Type}` | `FileList`, `FileUploader` |
| 共通UI | `{Name}` | `Button`, `Dialog`, `Input` |

---

## 7. スタイリング規則

### 7.1 Tailwind CSS使用方針

- **shadcn/ui**のコンポーネントをベースに使用
- カスタムスタイルはTailwindクラスで適用
- グローバルCSSは最小限に

### 7.2 カラー設計

```css
/* globals.css */
:root {
  --background: 0 0% 100%;
  --foreground: 222.2 84% 4.9%;
  --primary: 222.2 47.4% 11.2%;
  --primary-foreground: 210 40% 98%;
  /* ... */
}

.dark {
  --background: 222.2 84% 4.9%;
  --foreground: 210 40% 98%;
  /* ... */
}
```

### 7.3 レスポンシブ設計

Tailwindのブレークポイントを使用:

| ブレークポイント | 幅 | 用途 |
|----------------|-----|------|
| sm | 640px | モバイル横向き |
| md | 768px | タブレット |
| lg | 1024px | デスクトップ小 |
| xl | 1280px | デスクトップ |

---

## 8. エラーハンドリング

### 8.1 Error Boundary

```typescript
// components/common/ErrorBoundary.tsx
export function ErrorBoundary({ children }: { children: React.ReactNode }) {
  return (
    <ReactErrorBoundary
      fallbackRender={({ error, resetErrorBoundary }) => (
        <ErrorFallback error={error} onRetry={resetErrorBoundary} />
      )}
    >
      {children}
    </ReactErrorBoundary>
  );
}
```

### 8.2 APIエラー処理

```typescript
// TanStack Queryのグローバル設定
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30 * 1000,
    },
    mutations: {
      onError: (error) => {
        // トースト通知等
        toast.error(getErrorMessage(error));
      },
    },
  },
});
```

---

## 9. テスト方針

### 9.1 テストの種類

| 種類 | ツール | 対象 |
|------|--------|------|
| Unit | Vitest | ユーティリティ、カスタムフック |
| Component | Testing Library | UIコンポーネント |
| Integration | Playwright | E2Eフロー |

### 9.2 テストファイル配置

```
src/
├── components/
│   └── files/
│       ├── FileItem.tsx
│       └── FileItem.test.tsx    # コロケーション
├── hooks/
│   ├── useDebounce.ts
│   └── useDebounce.test.ts
└── lib/
    └── utils/
        ├── format.ts
        └── format.test.ts
```

---

## 関連ドキュメント

- [バックエンド設計](./BACKEND.md)
- [データベース設計](./DATABASE.md)
- [API設計](./API.md)

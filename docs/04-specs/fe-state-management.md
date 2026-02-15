# フロントエンド状態管理仕様書

## メタ情報

| 項目 | 値 |
|------|-----|
| ステータス | Ready |
| 優先度 | High |
| 関連仕様 | fe-foundation, fe-file-browser |

---

## 1. 概要

本ドキュメントでは、フロントエンドの状態管理要件を定義します。状態の種類に応じて適切な管理方法を使い分けます。

---

## 2. 状態管理の方針

### 2.1 状態の分類

| 状態の種類 | 管理方法 | 永続化 | 例 |
|-----------|----------|--------|-----|
| サーバー状態 | TanStack Query | キャッシュ | フォルダ内容、ユーザー情報 |
| URL状態 | TanStack Router | URL | 現在のフォルダ、表示設定 |
| グローバルUI状態 | Zustand | localStorage | サイドバー開閉、テーマ |
| ローカルUI状態 | useState | なし | ダイアログ開閉、入力値 |
| 一時的状態 | Zustand | なし | 認証状態、選択アイテム |

### 2.2 使い分けの原則

- **サーバーから取得するデータ** → TanStack Query
- **URLで共有したいデータ** → TanStack Router
- **複数コンポーネントで共有するUI設定** → Zustand
- **単一コンポーネント内のUI状態** → useState

---

## 3. 認証ストア

### 3.1 状態定義

| フィールド | 型 | 初期値 | 説明 |
|-----------|-----|--------|------|
| status | 'initializing' \| 'authenticated' \| 'unauthenticated' | 'initializing' | 認証状態 |
| user | User \| null | null | ユーザー情報 |

### 3.2 アクション

| アクション | 引数 | 動作 |
|-----------|------|------|
| setUser | user: User | ユーザー設定、status を authenticated に |
| clearAuth | なし | ユーザークリア、status を unauthenticated に |
| setInitializing | なし | status を initializing に |

### 3.3 永続化

- **永続化しない**
- Cookieが認証の唯一の真の情報源
- ページリロード時はAPIから再取得

### 3.4 状態遷移

```
アプリ起動
    │
    ▼
initializing ──GET /api/v1/me──┬── 200 ──▶ authenticated
                               │
                               └── 401 ──▶ unauthenticated

ログイン成功
    │
    ▼
authenticated

ログアウト / 401エラー
    │
    ▼
unauthenticated
```

---

## 4. UIストア

### 4.1 状態定義

| フィールド | 型 | 初期値 | 永続化 | 説明 |
|-----------|-----|--------|--------|------|
| sidebarOpen | boolean | true | Yes | サイドバー開閉状態 |
| viewMode | 'list' \| 'grid' | 'list' | Yes | ファイル表示モード |
| sortBy | 'name' \| 'updatedAt' \| 'size' \| 'type' | 'name' | Yes | ソート項目 |
| sortOrder | 'asc' \| 'desc' | 'asc' | Yes | ソート順序 |
| theme | 'light' \| 'dark' \| 'system' | 'system' | Yes | テーマ設定 |

### 4.2 アクション

| アクション | 引数 | 動作 |
|-----------|------|------|
| toggleSidebar | なし | sidebarOpen を反転 |
| setSidebarOpen | open: boolean | sidebarOpen を設定 |
| setViewMode | mode: 'list' \| 'grid' | viewMode を設定 |
| setSortBy | sortBy: string | sortBy を設定 |
| setSortOrder | order: 'asc' \| 'desc' | sortOrder を設定 |
| setTheme | theme: string | theme を設定 |

### 4.3 永続化

- localStorage に保存
- キー: `gc-storage-ui-settings`
- ページリロード時に復元

---

## 5. 選択ストア

### 5.1 状態定義

| フィールド | 型 | 初期値 | 説明 |
|-----------|-----|--------|------|
| selectedIds | Set<string> | 空のSet | 選択中のアイテムID |
| lastSelectedId | string \| null | null | 最後に選択したアイテムID |

### 5.2 アクション

| アクション | 引数 | 動作 |
|-----------|------|------|
| select | id: string | 単一選択（既存の選択をクリア） |
| toggle | id: string | 選択状態を反転（Ctrl+クリック用） |
| selectRange | id: string, items: Item[] | 範囲選択（Shift+クリック用） |
| selectAll | items: Item[] | 全選択 |
| clear | なし | 選択をクリア |
| isSelected | id: string | 選択状態を取得 |

### 5.3 永続化

- **永続化しない**
- フォルダ遷移時にクリア

### 5.4 選択動作

| 操作 | 動作 |
|------|------|
| クリック | 単一選択 |
| Ctrl/Cmd + クリック | 選択を追加/解除 |
| Shift + クリック | 範囲選択 |
| Ctrl/Cmd + A | 全選択 |
| Escape | 選択解除 |
| 空白部分クリック | 選択解除 |

---

## 6. アップロードストア

### 6.1 状態定義

| フィールド | 型 | 初期値 | 説明 |
|-----------|-----|--------|------|
| uploads | Map<string, UploadItem> | 空のMap | アップロード中のアイテム |
| isUploading | boolean | false | アップロード中フラグ |

### 6.2 UploadItem型

| フィールド | 型 | 説明 |
|-----------|-----|------|
| id | string | アップロードID |
| fileName | string | ファイル名 |
| fileSize | number | ファイルサイズ |
| progress | number | 進捗率（0-100） |
| status | 'pending' \| 'uploading' \| 'completed' \| 'failed' | ステータス |
| error | string \| null | エラーメッセージ |

### 6.3 アクション

| アクション | 引数 | 動作 |
|-----------|------|------|
| addUpload | file: File, folderId: string | アップロードをキューに追加 |
| updateProgress | id: string, progress: number | 進捗を更新 |
| completeUpload | id: string | 完了状態に設定 |
| failUpload | id: string, error: string | 失敗状態に設定 |
| retryUpload | id: string | 失敗したアップロードを再試行 |
| cancelUpload | id: string | アップロードをキャンセル |
| clearCompleted | なし | 完了したアップロードをクリア |

### 6.4 永続化

- **永続化しない**
- ページリロード時はアップロード中断

### 6.5 アップロードフロー

```
ファイル選択/ドロップ
    │
    ▼
pending（キューに追加）
    │
    ▼
uploading（Presigned URL取得 → S3アップロード）
    │
    ├── 成功 ──▶ completed
    │
    └── 失敗 ──▶ failed
                    │
                    └── 再試行 ──▶ uploading
```

---

## 7. サーバー状態（TanStack Query）

### 7.1 クエリキー設計

| データ | クエリキー |
|--------|----------|
| 現在のユーザー | `['me']` |
| フォルダ内容 | `['folders', folderId, 'contents']` |
| パンくずリスト | `['folders', folderId, 'ancestors']` |
| ゴミ箱一覧 | `['trash']` |
| ファイルバージョン | `['files', fileId, 'versions']` |

### 7.2 キャッシュ設定

| データ | staleTime | gcTime | 説明 |
|--------|-----------|--------|------|
| ユーザー情報 | 5分 | 30分 | 頻繁に変更されない |
| フォルダ内容 | 1分 | 10分 | 適度に新鮮さを保つ |
| パンくずリスト | 5分 | 30分 | フォルダ構造は安定 |
| ゴミ箱一覧 | 1分 | 10分 | 操作後は即時無効化 |

### 7.3 キャッシュ無効化

| 操作 | 無効化するキー |
|------|--------------|
| フォルダ作成 | 親フォルダの `contents` |
| フォルダ削除 | 親フォルダの `contents`, `ancestors` |
| ファイルアップロード | 対象フォルダの `contents` |
| ファイル削除 | 対象フォルダの `contents`, `trash` |
| ゴミ箱復元 | 復元先フォルダの `contents`, `trash` |

---

## 8. URL状態（TanStack Router）

### 8.1 パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|-----|------|
| folderId | string (UUID) | 表示中のフォルダID |
| groupId | string (UUID) | 表示中のグループID |
| provider | 'google' \| 'github' | OAuthプロバイダー |

### 8.2 クエリパラメータ

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|----------|------|
| view | 'list' \| 'grid' | 'list' | 表示モード |
| sort | string | 'name' | ソート項目 |
| order | 'asc' \| 'desc' | 'asc' | ソート順序 |
| redirect | string | なし | ログイン後のリダイレクト先 |
| token | string | なし | 認証トークン |

### 8.3 URLとストアの同期

- URL変更 → ストア更新
- ストア変更 → URL更新（ナビゲーション）
- 初回ロード時はURLの値をストアに反映

---

## 9. 状態の初期化フロー

### 9.1 アプリ起動時

```
1. UIストアをlocalStorageから復元
2. 認証状態を initializing に設定
3. GET /api/v1/me を呼び出し
4. 成功: ユーザー情報を設定、authenticated に
   失敗: unauthenticated に
5. ルーティング処理を開始
```

### 9.2 ページ遷移時

```
1. 選択状態をクリア
2. ルートローダーでデータをプリフェッチ
3. コンポーネントをレンダリング
```

---

## 10. 受け入れ基準

### 10.1 認証ストア

- [ ] ログイン後、ユーザー情報が保持される
- [ ] ログアウト後、ユーザー情報がクリアされる
- [ ] 401エラー時、認証状態がクリアされる
- [ ] ページリロード時、APIからユーザー情報を再取得する

### 10.2 UIストア

- [ ] サイドバー開閉状態がリロード後も維持される
- [ ] 表示モードがリロード後も維持される
- [ ] ソート設定がリロード後も維持される

### 10.3 選択ストア

- [ ] クリックで単一選択できる
- [ ] Ctrl+クリックで複数選択できる
- [ ] Shift+クリックで範囲選択できる
- [ ] フォルダ遷移時に選択がクリアされる

### 10.4 アップロードストア

- [ ] アップロード進捗が表示される
- [ ] 失敗したアップロードを再試行できる
- [ ] アップロードをキャンセルできる

### 10.5 サーバー状態

- [ ] データがキャッシュされ、重複リクエストが発生しない
- [ ] 操作後、関連データが自動的に再取得される
- [ ] エラー時にリトライできる

---

## 関連ドキュメント

- [fe-foundation.md](./fe-foundation.md) - フロントエンド基盤仕様
- [fe-file-browser.md](./fe-file-browser.md) - ファイルブラウザ仕様
- [FRONTEND.md](../02-architecture/FRONTEND.md) - フロントエンド設計

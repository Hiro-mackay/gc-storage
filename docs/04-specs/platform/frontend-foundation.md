# フロントエンド基盤仕様書

> fe-foundation.md, fe-routing.md, fe-state-management.md を統合

## メタ情報

| 項目 | 値 |
|------|-----|
| ステータス | Ready |
| 優先度 | High |
| 関連ドメイン | user, session |
| 依存する仕様 | auth-identity, infra-api, fe-openapi-typegen |

---

## 1. 概要

本ドキュメントでは、GC Storageフロントエンドの基盤要件を統合的に定義します。

- **API通信基盤**: openapi-fetch + openapi-typescript による型安全なAPI通信
- **認証状態管理**: Cookie(HttpOnly)ベースの認証とUI用ストア
- **ルーティング**: TanStack Router による認証ガード付きSPA routing
- **レイアウト**: ヘッダー + サイドバー + メインコンテンツ
- **状態管理**: 4種類の状態を適切なツールで管理

**関連アーキテクチャ:**
- [FRONTEND.md](../../02-architecture/FRONTEND.md) - フロントエンド設計方針
- [API.md](../../02-architecture/API.md) - API設計
- [fe-openapi-typegen.md](./openapi-typegen.md) - OpenAPI型生成パイプライン

---

## 2. API通信基盤

### 2.1 技術選定

| ライブラリ | 用途 |
|-----------|------|
| openapi-fetch | OpenAPIスキーマから型推論するfetch wrapper |
| openapi-typescript | OpenAPIスキーマからTypeScript型定義を自動生成 |

### 2.2 機能要件

| ID | 要件 | 優先度 |
|----|------|--------|
| API-01 | APIリクエストにcredentials: 'include'を設定し、Cookieを自動送信 | High |
| API-02 | APIエラーレスポンスを統一された形式でハンドリング | High |
| API-03 | 401エラー時にログアウト状態に遷移 | High |
| API-04 | ネットワークエラーを適切にハンドリング | Medium |
| API-05 | リクエストタイムアウトを設定（30秒） | Medium |
| API-06 | OpenAPIスキーマから生成された型定義でE2Eの型安全性を確保 | High |

### 2.3 エラーハンドリング

| HTTPステータス | 動作 |
|---------------|------|
| 400 Bad Request | バリデーションエラーとして処理 |
| 401 Unauthorized | storeをクリアし、ログインページへリダイレクト |
| 403 Forbidden | 権限エラーとして処理 |
| 404 Not Found | リソース未発見エラーとして処理 |
| 500+ Server Error | サーバーエラーとして処理、リトライ可能 |
| Network Error | ネットワークエラーとして処理 |

### 2.4 エラーメッセージ対応表

| エラーコード | 表示メッセージ |
|-------------|--------------|
| NETWORK_ERROR | インターネット接続を確認してください |
| UNAUTHORIZED | ログインが必要です |
| FORBIDDEN | この操作を行う権限がありません |
| NOT_FOUND | リソースが見つかりませんでした |
| VALIDATION_ERROR | 入力内容を確認してください |
| SERVER_ERROR | サーバーでエラーが発生しました。しばらく待ってから再試行してください |

---

## 3. 認証状態管理

### 3.1 認証情報の管理方針

| データ | 管理方法 | 説明 |
|--------|----------|------|
| Session ID | HttpOnly Cookie | バックエンドが設定、フロントエンドからはアクセス不可 |
| ユーザー情報（ID, name, email） | Store（一時的） | UI表示用、ページリロード時はAPIで再取得 |
| 認証状態フラグ | Store（一時的） | UI制御用、永続化しない |

**Cookieが認証の唯一の真の情報源（Single Source of Truth）。**

### 3.2 認証状態

| 状態 | 説明 |
|------|------|
| initializing | アプリ起動時、認証状態を確認中 |
| unauthenticated | ログインしていない状態 |
| authenticated | ログイン済みの状態 |

### 3.3 認証確認フロー

```
アプリ起動
    |
    v
initializing --GET /api/v1/me--+-- 200 --> authenticated
                                |
                                +-- 401 --> unauthenticated

ログアウト / 401エラー --> unauthenticated
```

### 3.4 セッション自動延長

- 各APIリクエスト時にサーバーがセッション有効期限を自動延長（スライディングウィンドウ）
- フロントエンドはセッション管理を意識する必要がない
- 7日間アクセスがない場合にセッション期限切れ

---

## 4. ルーティング

### 4.1 公開ルート（認証不要）

| パス | 説明 | 認証済み時 |
|------|------|-----------|
| `/login` | ログインページ | `/files` へリダイレクト |
| `/register` | 新規登録ページ | `/files` へリダイレクト |
| `/forgot-password` | パスワードリセット要求 | `/files` へリダイレクト |
| `/reset-password` | パスワードリセット | そのまま表示 |
| `/verify-email` | メール確認 | そのまま表示 |
| `/oauth/callback/:provider` | OAuthコールバック | 処理後 `/files` へ |

### 4.2 認証必須ルート

| パス | 説明 | 未認証時 |
|------|------|---------|
| `/` | ホーム | `/files` へリダイレクト |
| `/files` | マイファイル（ルートフォルダ） | `/login` へリダイレクト |
| `/files/:folderId` | フォルダ詳細 | `/login` へリダイレクト |
| `/trash` | ゴミ箱 | `/login` へリダイレクト |
| `/starred` | スター付き | `/login` へリダイレクト |
| `/recent` | 最近のファイル | `/login` へリダイレクト |
| `/shared` | 共有されたファイル | `/login` へリダイレクト |
| `/groups` | グループ一覧 | `/login` へリダイレクト |
| `/groups/:groupId` | グループ詳細 | `/login` へリダイレクト |
| `/settings` | 設定 | `/login` へリダイレクト |

### 4.3 認証ガード

**未認証ユーザーが認証必須ページにアクセス**:
1. 元のURLを保存
2. `/login` にリダイレクト
3. ログイン成功後、元のURLにリダイレクト

**認証済みユーザーがログインページにアクセス**:
1. `/files` にリダイレクト

### 4.4 レイアウト階層

```
__root (ルートレイアウト)
+-- _auth (認証不要レイアウト)
|   +-- login, register, forgot-password, reset-password
|   +-- verify-email, oauth/callback/:provider
+-- _authenticated (認証必須レイアウト)
    +-- files, files/:folderId, trash, starred, recent
    +-- shared, groups, groups/:groupId, settings
```

### 4.5 データプリフェッチ（ルートローダー）

| ルート | プリフェッチするデータ |
|--------|---------------------|
| `/files` | ルートフォルダ内容 |
| `/files/:folderId` | フォルダ内容 + パンくずリスト |
| `/trash` | ゴミ箱一覧 |

### 4.6 URL状態管理

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|----------|------|
| view | 'list' \| 'grid' | 'list' | 表示モード |
| sort | string | 'name' | ソート項目 |
| order | 'asc' \| 'desc' | 'asc' | ソート順序 |
| redirect | string | なし | ログイン後リダイレクト先 |
| token | string | なし | 認証トークン |

---

## 5. レイアウト

### 5.1 認証済みレイアウト

```
+-------------------------------------------------------------+
| ヘッダー                                                      |
| [メニュー] ロゴ           [検索]        [通知] [ユーザー]      |
+----------+--------------------------------------------------+
|          |                                                   |
| サイド   |                                                   |
| バー     |                 メインコンテンツ                    |
|          |                                                   |
+----------+--------------------------------------------------+
```

### 5.2 公開レイアウト（認証ページ用）

```
+-------------------------------------------------------------+
|                                                              |
|                         ロゴ                                 |
|                                                              |
|                  +-------------------+                       |
|                  |   認証フォーム     |                       |
|                  +-------------------+                       |
|                                                              |
+-------------------------------------------------------------+
```

### 5.3 ナビゲーション項目

| 項目 | アイコン | リンク先 | アクティブ判定 |
|------|---------|---------|--------------|
| マイファイル | フォルダ | `/files` | `/files` or `/files/*` |
| スター付き | 星 | `/starred` | `/starred` |
| 最近 | 時計 | `/recent` | `/recent` |
| 共有されたファイル | 共有 | `/shared` | `/shared` |
| グループ | ユーザーグループ | `/groups` | `/groups` or `/groups/*` |
| ゴミ箱 | ゴミ箱 | `/trash` | `/trash` |

---

## 6. 状態管理

### 6.1 状態の分類

| 状態の種類 | 管理方法 | 永続化 | 例 |
|-----------|----------|--------|-----|
| サーバー状態 | TanStack Query | キャッシュ | フォルダ内容、ユーザー情報 |
| URL状態 | TanStack Router | URL | 現在のフォルダ、表示設定 |
| グローバルUI状態 | Zustand | localStorage | サイドバー開閉、テーマ |
| ローカルUI状態 | useState | なし | ダイアログ開閉、入力値 |
| 一時的状態 | Zustand | なし | 認証状態、選択アイテム |

### 6.2 認証ストア（Zustand, 永続化なし）

| フィールド | 型 | 初期値 |
|-----------|-----|--------|
| status | 'initializing' \| 'authenticated' \| 'unauthenticated' | 'initializing' |
| user | User \| null | null |

### 6.3 UIストア（Zustand, localStorage永続化）

| フィールド | 型 | 初期値 |
|-----------|-----|--------|
| sidebarOpen | boolean | true |
| viewMode | 'list' \| 'grid' | 'list' |
| sortBy | 'name' \| 'updatedAt' \| 'size' \| 'type' | 'name' |
| sortOrder | 'asc' \| 'desc' | 'asc' |
| theme | 'light' \| 'dark' \| 'system' | 'system' |

### 6.4 選択ストア（Zustand, 永続化なし）

| フィールド | 型 | 初期値 |
|-----------|-----|--------|
| selectedIds | Set<string> | 空のSet |
| lastSelectedId | string \| null | null |

**選択動作:**
| 操作 | 動作 |
|------|------|
| クリック | 単一選択 |
| Ctrl/Cmd + クリック | 選択を追加/解除 |
| Shift + クリック | 範囲選択 |
| Ctrl/Cmd + A | 全選択 |
| Escape | 選択解除 |

### 6.5 アップロードストア（Zustand, 永続化なし）

| フィールド | 型 | 初期値 |
|-----------|-----|--------|
| uploads | Map<string, UploadItem> | 空のMap |
| isUploading | boolean | false |

### 6.6 サーバー状態（TanStack Query）

**クエリキー設計:**
| データ | クエリキー | staleTime | gcTime |
|--------|----------|-----------|--------|
| 現在のユーザー | `['me']` | 5min | 30min |
| フォルダ内容 | `['folders', folderId, 'contents']` | 1min | 10min |
| パンくずリスト | `['folders', folderId, 'ancestors']` | 5min | 30min |
| ゴミ箱一覧 | `['trash']` | 1min | 10min |

**キャッシュ無効化:**
| 操作 | 無効化するキー |
|------|--------------|
| フォルダ作成 | 親フォルダの `contents` |
| フォルダ削除 | 親フォルダの `contents`, `ancestors` |
| ファイルアップロード | 対象フォルダの `contents` |
| ファイル削除 | 対象フォルダの `contents`, `trash` |
| ゴミ箱復元 | 復元先フォルダの `contents`, `trash` |

---

## 7. エラー・ローディング表示

### 7.1 エラーバウンダリ

- 予期しないエラーをキャッチしてフォールバックUIを表示
- 「再試行」「ホームに戻る」オプションを提供
- 開発環境ではエラー詳細を表示

### 7.2 ローディング

| ID | 要件 | 優先度 |
|----|------|--------|
| LOAD-01 | ページ読み込み中にスピナーを表示 | High |
| LOAD-02 | ボタンクリック後、処理中は操作を無効化 | High |
| LOAD-03 | 長時間の処理には進捗表示を提供 | Medium |
| LOAD-04 | スケルトンスクリーンでプレースホルダ表示 | Medium |

---

## 8. レスポンシブ対応

### ブレークポイント

| 名前 | 幅 | 対象 |
|------|-----|------|
| sm | 640px | モバイル横向き |
| md | 768px | タブレット |
| lg | 1024px | デスクトップ小 |
| xl | 1280px | デスクトップ |

### 要件

- モバイルでサイドバーがオーバーレイ/ハンバーガーメニュー
- モバイルでファイル一覧がシングルカラム
- 320px幅でもコンテンツが切れずに表示

---

## 9. 非機能要件

### パフォーマンス

| 項目 | 目標値 |
|------|--------|
| 初回ロード（LCP） | 2.5秒以内 |
| インタラクション応答（FID） | 100ms以内 |
| レイアウトシフト（CLS） | 0.1以下 |

### アクセシビリティ

- キーボードのみでナビゲーション可能
- フォーカス状態が視覚的に明確
- スクリーンリーダー対応

---

## 関連ドキュメント

- [FRONTEND.md](../../02-architecture/FRONTEND.md) - フロントエンド設計
- [API.md](../../02-architecture/API.md) - API設計
- [openapi-typegen.md](./openapi-typegen.md) - OpenAPI型生成

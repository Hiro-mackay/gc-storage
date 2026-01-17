# GC Storage バックエンド設計書

## 概要

本ドキュメントでは、GC StorageのGoバックエンドアーキテクチャの**設計原則とルール**について説明します。クリーンアーキテクチャをベースとした4層構造を採用し、テスタビリティと保守性を重視した設計となっています。

---

## 1. ディレクトリ構成

```
backend/
├── cmd/
│   └── api/
│       └── main.go                 # アプリケーションエントリーポイント
├── internal/
│   ├── config/                     # 設定管理
│   ├── domain/                     # ドメイン層
│   │   ├── entity/                 # エンティティ
│   │   ├── repository/             # リポジトリインターフェース
│   │   ├── service/                # ドメインサービス
│   │   └── valueobject/            # 値オブジェクト
│   ├── usecase/                    # ユースケース層
│   │   ├── auth/
│   │   ├── file/
│   │   ├── folder/
│   │   ├── group/
│   │   ├── permission/
│   │   └── search/
│   ├── interface/                  # インターフェース層
│   │   ├── handler/                # HTTPハンドラー
│   │   ├── middleware/             # ミドルウェア
│   │   ├── presenter/              # レスポンス変換
│   │   └── dto/                    # データ転送オブジェクト
│   │       ├── request/
│   │       └── response/
│   └── infrastructure/             # インフラストラクチャ層
│       ├── persistence/
│       │   ├── postgres/
│       │   └── redis/
│       ├── storage/
│       │   └── minio/
│       ├── external/
│       │   ├── oauth/
│       │   └── email/
│       └── database/
├── pkg/                            # 共有パッケージ
│   ├── apperror/
│   ├── validator/
│   ├── logger/
│   └── jwt/
├── migrations/                     # DBマイグレーション
├── api/
│   └── openapi.yaml               # OpenAPI仕様
└── scripts/
```

---

## 2. レイヤー設計

### 2.1 4層アーキテクチャ

```
┌────────────────────────────────────────────────────────────────┐
│                    Interface Layer (interface/)                │
│  HTTP Request/Response, ルーティング, 入力検証                    │
├────────────────────────────────────────────────────────────────┤
│                    UseCase Layer (usecase/)                    │
│  ビジネスロジックのオーケストレーション, トランザクション管理            │
├────────────────────────────────────────────────────────────────┤
│                    Domain Layer (domain/)                      │
│  ビジネスルール, ドメインロジック, エンティティ                       │
├────────────────────────────────────────────────────────────────┤
│                Infrastructure Layer (infrastructure/)          │
│  データ永続化, 外部サービス連携, 技術的実装詳細                       │
└────────────────────────────────────────────────────────────────┘
```

### 2.2 依存性の方向

```
Interface → UseCase → Domain ← Infrastructure
                         ↑
                    依存性逆転
```

**原則:**
- 上位レイヤーは下位レイヤーに依存可能
- 下位レイヤーは上位レイヤーに依存しない
- Domain層は他のどのレイヤーにも依存しない
- Infrastructure層はDomain層のインターフェースを実装（依存性逆転）

---

## 3. レイヤー責務

### 3.1 Domain Layer

| コンポーネント | 責務 |
|--------------|------|
| Entity | ビジネスオブジェクトの定義、ビジネスルールのカプセル化 |
| Value Object | 不変の値、自己検証機能を持つ |
| Repository Interface | データ永続化の抽象化（インターフェースのみ定義） |
| Domain Service | 複数エンティティにまたがるビジネスロジック |

**ルール:**
- Entityはビジネスルールに関するメソッドを持つ（例: `user.CanUpload(fileSize)`）
- Value Objectは生成時にバリデーションを行う
- Repository Interfaceは技術的詳細を含まない

### 3.2 UseCase Layer

| 責務 | 説明 |
|------|------|
| ビジネスロジックの調整 | 複数のRepository/Serviceを組み合わせる |
| トランザクション管理 | データの整合性を保証 |
| 認可チェック | ユーザーの操作権限を検証 |
| Input/Outputの定義 | UseCaseの入出力を明確に定義 |

**ルール:**
- 1ユースケース = 1ファイル
- Input/Output構造体を必ず定義
- HTTPやDBの詳細を知らない

### 3.3 Infrastructure Layer

| 責務 | 説明 |
|------|------|
| Repository実装 | Domain層のインターフェースを実装 |
| 外部サービス連携 | OAuth, メール送信など |
| DB接続管理 | コネクションプール、マイグレーション |
| ストレージ操作 | MinIOへのファイル操作 |

---

## 4. Interface Layer 詳細設計

Interface層はHTTPリクエスト/レスポンスを処理する層です。以下の3つのコンポーネントで構成されます。

### 4.1 処理フロー

```
HTTP Request
     │
     ▼
┌─────────────┐
│   Handler   │  リクエスト受付、バリデーション
└─────────────┘
     │ Request DTO
     ▼
┌─────────────┐
│   UseCase   │  ビジネスロジック実行
└─────────────┘
     │ UseCase Output
     ▼
┌─────────────┐
│  Presenter  │  レスポンス変換
└─────────────┘
     │ Response DTO
     ▼
HTTP Response
```

### 4.2 Handler

**役割:**
- HTTPリクエストの受付
- リクエストボディのパース・バリデーション
- 認証情報（UserID等）の取得
- UseCaseの呼び出し
- エラーハンドリング

**サンプル実装:**

```go
// internal/interface/handler/file_handler.go

type FileHandler struct {
    uploadUC  *file.UploadUseCase
    presenter *presenter.FilePresenter
}

// POST /api/v1/files/upload
func (h *FileHandler) InitUpload(c echo.Context) error {
    // 1. リクエストのパースとバリデーション
    var req request.InitUploadRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError(err)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    // 2. 認証情報の取得
    userID := getUserIDFromContext(c)

    // 3. UseCaseの呼び出し
    output, err := h.uploadUC.Execute(c.Request().Context(), file.UploadInput{
        UserID:   userID,
        FolderID: req.FolderID,
        Name:     req.Name,
        Size:     req.Size,
        MimeType: req.MimeType,
    })
    if err != nil {
        return err
    }

    // 4. レスポンス変換と返却
    return c.JSON(http.StatusOK, h.presenter.ToInitUploadResponse(output))
}
```

### 4.3 Presenter

**役割:**
- UseCaseの出力をHTTPレスポンス形式に変換
- ドメインオブジェクトをDTO形式に変換
- 必要に応じてデータの整形（日付フォーマット等）

**サンプル実装:**

```go
// internal/interface/presenter/file_presenter.go

type FilePresenter struct{}

func (p *FilePresenter) ToInitUploadResponse(output *file.UploadOutput) *response.InitUploadResponse {
    return &response.InitUploadResponse{
        FileID:    output.FileID.String(),
        UploadURL: output.UploadURL,
        ExpiresAt: output.ExpiresAt.Format(time.RFC3339),
    }
}

func (p *FilePresenter) ToFileResponse(file *entity.File) *response.FileResponse {
    return &response.FileResponse{
        ID:        file.ID.String(),
        Name:      file.Name,
        Size:      file.Size,
        MimeType:  file.MimeType,
        CreatedAt: file.CreatedAt.Format(time.RFC3339),
        UpdatedAt: file.UpdatedAt.Format(time.RFC3339),
    }
}
```

### 4.4 DTO (Data Transfer Object)

**役割:**
- リクエスト/レスポンスのデータ構造を定義
- バリデーションタグの定義
- JSONシリアライズ設定

**ルール:**
- Request DTOとResponse DTOは分離する
- DTOはドメインオブジェクトを直接参照しない
- バリデーションはRequest DTOで定義

**サンプル実装:**

```go
// internal/interface/dto/request/file.go

type InitUploadRequest struct {
    Name     string  `json:"name" validate:"required,max=255"`
    FolderID *string `json:"folder_id" validate:"omitempty,uuid"`
    Size     int64   `json:"size" validate:"required,min=1,max=5368709120"`
    MimeType string  `json:"mime_type" validate:"required"`
}

type MoveFileRequest struct {
    DestinationFolderID string `json:"destination_folder_id" validate:"required,uuid"`
}
```

```go
// internal/interface/dto/response/file.go

type InitUploadResponse struct {
    FileID    string `json:"file_id"`
    UploadURL string `json:"upload_url"`
    ExpiresAt string `json:"expires_at"`
}

type FileResponse struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    FolderID  string `json:"folder_id,omitempty"`
    Size      int64  `json:"size"`
    MimeType  string `json:"mime_type"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
}
```

### 4.5 疎結合と冗長性のバランス

Interface層の設計において、以下のバランスを考慮します:

| 方針 | 説明 |
|------|------|
| DTOの分離 | Request/Responseを分離することで、それぞれの変更が独立する |
| Presenterの導入 | UseCase出力とHTTPレスポンスを分離し、表示形式の変更を容易にする |
| 適度な冗長性 | 多少の重複を許容し、各コンポーネントの独立性を保つ |

**冗長性を許容するケース:**
- Request DTOとUseCase Inputが似ている場合でも、両方定義する
- Response DTOとEntity構造が似ている場合でも、Presenterで変換する

**理由:**
- レイヤー間の結合を防ぐ
- 将来の変更に対する柔軟性を確保
- テスタビリティの向上

---

## 5. コーディング規約

### 5.1 命名規則

| 種類 | 規則 | 例 |
|------|------|-----|
| パッケージ | 小文字、単数形 | `entity`, `handler` |
| ファイル | スネークケース | `file_handler.go` |
| 構造体 | パスカルケース | `FileHandler` |
| インターフェース | パスカルケース（`er`接尾辞推奨） | `FileRepository` |
| メソッド | パスカルケース | `InitUpload` |
| 変数 | キャメルケース | `fileID` |

### 5.2 エラーハンドリング方針

| 種類 | 対応 |
|------|------|
| ビジネスエラー | `apperror`パッケージで定義した構造化エラーを返す |
| システムエラー | ログ出力後、汎用エラーに変換して返す |
| バリデーションエラー | フィールド単位でエラー詳細を返す |

### 5.3 コンテキスト使用方針

- すべてのRepository/UseCaseメソッドは第一引数に`context.Context`を受け取る
- タイムアウト、キャンセル、トレーシング情報の伝播に使用
- 認証情報はContextではなく、明示的な引数として渡す

---

## 関連ドキュメント

- [データベース設計](./DATABASE.md)
- [API設計](./API.md)
- [フロントエンド設計](./FRONTEND.md)

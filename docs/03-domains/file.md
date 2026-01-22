# File ドメイン

## 概要

Fileドメインは、ファイルのアップロード、ダウンロード、バージョン管理を担当します。
Storage Contextの中核として、ファイルの実体（MinIO）とメタデータ（PostgreSQL）の整合性を保証します。

### 設計方針

- **MinIOバージョニング使用**: 同一StorageKeyに対してMinIOがバージョンIDを管理
- **アーカイブテーブル方式**: 論理削除（trashed_at）ではなく、別テーブルへの移動で実現
- **所有者情報の分離**: StorageKeyにビジネス情報を含めず、メタデータはDB管理
- **Webhook駆動のアップロード完了**: クライアントからの完了通知ではなく、MinIO Bucket Notificationでサーバーが検知

---

## エンティティ

### File（集約ルート）

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | ファイルの一意識別子 |
| folder_id | UUID | Yes | 所属フォルダID（必須 - ファイルは必ずフォルダに所属） |
| owner_id | UUID | Yes | 現在の所有者ID（所有権譲渡で変更可能） |
| created_by | UUID | Yes | 最初の作成者ID（不変、履歴追跡用） |
| name | FileName | Yes | ファイル名（値オブジェクト） |
| mime_type | MimeType | Yes | MIMEタイプ（値オブジェクト） |
| size | int64 | Yes | 現在のバージョンのファイルサイズ（バイト） |
| storage_key | StorageKey | Yes | MinIO内のオブジェクトキー（値オブジェクト） |
| current_version | int | Yes | 現在のバージョン番号 |
| status | FileStatus | Yes | ファイル状態 |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**ステータス定義:**

| ステータス | 説明 |
|-----------|------|
| uploading | アップロード中 |
| active | アクティブ（通常利用可能） |
| upload_failed | アップロード失敗 |

**ステータス遷移:**

```
┌────────────┐
│ uploading  │
└─────┬──────┘
      │
  ┌───┴───┐
  │       │
  ▼       ▼
┌──────┐ ┌───────────────┐
│active│ │ upload_failed │
└──┬───┘ └───────────────┘
   │
   │ Trash操作
   ▼
┌──────────────────┐
│  ArchivedFile    │ ← 別テーブルへ移動
└────────┬─────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌──────┐  ┌────────────┐
│ 復元 │  │ 完全削除    │
│→File │  │（物理削除） │
└──────┘  └────────────┘
```

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-FL001 | storage_keyはfile_idから生成される |
| R-FL002 | 同一フォルダ内でnameは一意 |
| R-FL003 | sizeは0以上（0バイトファイルは許容） |
| R-FL004 | current_versionは1以上の連番 |
| R-FL005 | statusがuploadingからactiveへの遷移はアップロード完了時のみ |
| R-FL006 | statusがupload_failedのファイルは自動クリーンアップ対象 |
| R-FL007 | folder_idは必須（ファイルは必ずフォルダに所属） |
| R-FL008 | 新規作成時は`owner_id = created_by = 作成者` |
| R-FL009 | `created_by`は不変（所有権譲渡後も変更されない） |

---

### FileVersion

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | バージョンの一意識別子 |
| file_id | UUID | Yes | ファイルID |
| version_number | int | Yes | ユーザー向けバージョン番号（1, 2, 3...） |
| minio_version_id | string | Yes | MinIOが生成したバージョンID |
| size | int64 | Yes | このバージョンのサイズ |
| checksum | string | Yes | SHA-256チェックサム |
| uploaded_by | UUID | Yes | アップロードしたユーザーID |
| created_at | timestamp | Yes | 作成日時 |

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-FV001 | version_numberは同一file_id内で一意かつ連番 |
| R-FV002 | minio_version_idはMinIOから取得した値をそのまま保存 |
| R-FV003 | checksumはアップロード完了時に必ず計算・検証 |
| R-FV004 | 特定バージョンの取得はstorage_key + minio_version_idで行う |

---

### ArchivedFile

ゴミ箱に移動されたファイル。復元に必要な情報を完全に保持する。

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | アーカイブの一意識別子 |
| original_file_id | UUID | Yes | 元のファイルID |
| original_folder_id | UUID | Yes | 復元先フォルダID |
| original_path | string | Yes | 復元時の参考パス（例: "/documents/report.pdf"） |
| name | FileName | Yes | ファイル名 |
| mime_type | MimeType | Yes | MIMEタイプ |
| size | int64 | Yes | 最新バージョンのサイズ |
| owner_id | UUID | Yes | 所有者ID |
| created_by | UUID | Yes | 最初の作成者ID |
| storage_key | StorageKey | Yes | MinIOキー（復元・削除用） |
| archived_at | timestamp | Yes | アーカイブ日時 |
| archived_by | UUID | Yes | ゴミ箱に入れたユーザーID |
| expires_at | timestamp | Yes | 自動削除日時 |

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-AF001 | expires_atはarchived_atから30日後に設定 |
| R-AF002 | expires_atを過ぎたファイルはバッチ処理で自動削除 |
| R-AF003 | 復元時、original_folder_idが存在しない場合はルートに復元 |
| R-AF004 | 完全削除時はMinIOの全バージョンも削除 |

---

### ArchivedFileVersion

ゴミ箱内ファイルのバージョン情報。

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 一意識別子 |
| archived_file_id | UUID | Yes | ArchivedFileへの参照 |
| original_version_id | UUID | Yes | 元のFileVersion ID |
| version_number | int | Yes | バージョン番号 |
| minio_version_id | string | Yes | MinIOバージョンID |
| size | int64 | Yes | サイズ |
| checksum | string | Yes | SHA-256チェックサム |
| uploaded_by | UUID | Yes | アップロードしたユーザーID |
| created_at | timestamp | Yes | 元の作成日時 |

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-AFV001 | ArchivedFile削除時、関連するArchivedFileVersionも削除 |
| R-AFV002 | 復元時、全バージョンをFileVersionとして復元 |

---

### UploadSession

アップロードセッションの管理。シングルパート・マルチパート両方に対応。

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | セッションID |
| file_id | UUID | Yes | 作成予定のファイルID（事前生成） |
| owner_id | UUID | Yes | 所有者ID |
| created_by | UUID | Yes | 作成者ID（アップロード者） |
| folder_id | UUID | Yes | アップロード先フォルダID（必須） |
| file_name | FileName | Yes | ファイル名 |
| mime_type | MimeType | Yes | MIMEタイプ |
| total_size | int64 | Yes | 予定サイズ |
| storage_key | StorageKey | Yes | MinIOキー |
| minio_upload_id | string | No | MinIOマルチパートアップロードID |
| is_multipart | boolean | Yes | マルチパートかどうか |
| total_parts | int | Yes | 予定パーツ数（シングルパートは1） |
| uploaded_parts | int | Yes | アップロード済みパーツ数 |
| status | UploadStatus | Yes | セッション状態 |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |
| expires_at | timestamp | Yes | セッション有効期限 |

**ステータス定義:**

| ステータス | 説明 |
|-----------|------|
| pending | 初期化済み、未開始 |
| in_progress | パーツアップロード中 |
| completed | 完了 |
| aborted | 中断 |
| expired | 期限切れ |

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-US001 | expires_atのデフォルトは作成から24時間後 |
| R-US002 | expires_atを過ぎたセッションは自動キャンセル |
| R-US003 | is_multipart=trueの場合、minio_upload_idは必須 |
| R-US004 | completedまたはaborted後は状態変更不可 |
| R-US005 | マルチパートセッションのキャンセル時はMinIO側もAbort |

---

### UploadPart

マルチパートアップロードの各パーツ情報。

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 一意識別子 |
| session_id | UUID | Yes | UploadSessionへの参照 |
| part_number | int | Yes | パート番号（1から開始） |
| size | int64 | Yes | パートサイズ |
| etag | string | Yes | MinIOから返却されたETag |
| uploaded_at | timestamp | Yes | アップロード日時 |

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-UP001 | part_numberは同一session_id内で一意 |
| R-UP002 | etagはCompleteMultipartUpload時に必要 |

---

## 値オブジェクト

### StorageKey

MinIO内のオブジェクトキー。所有者情報を含まず、file_idのみで構成。

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | MinIO内のオブジェクトキー |

**形式:** `{file_id}`

**要件:**

| ID | 要件 |
|----|------|
| R-SK001 | 最大長は1024バイト（UTF-8） |
| R-SK002 | file_idのUUID文字列形式で生成 |
| R-SK003 | MinIO非対応文字（`\` `:`）を含まない |
| R-SK004 | 所有者情報を含めない（所有者変更時にオブジェクト移動不要） |
| R-SK005 | 推測困難な形式（UUID）でセキュリティを確保 |

---

### MimeType

MIMEタイプを表す値オブジェクト。カテゴリ情報を持つ。

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | MIMEタイプ文字列（例: "image/png"） |
| category | MimeCategory | カテゴリ |

**カテゴリ定義:**

| カテゴリ | 説明 | 例 |
|---------|------|-----|
| image | 画像 | image/png, image/jpeg, image/gif |
| video | 動画 | video/mp4, video/webm |
| audio | 音声 | audio/mpeg, audio/wav |
| document | ドキュメント | application/pdf, text/plain, application/msword |
| archive | アーカイブ | application/zip, application/x-tar |
| other | その他 | application/octet-stream |

**要件:**

| ID | 要件 |
|----|------|
| R-MT001 | 形式は `type/subtype` パターンに従う |
| R-MT002 | カテゴリはvalueのtype部分から自動判定 |
| R-MT003 | 不明なtypeの場合はカテゴリをotherに設定 |
| R-MT004 | カテゴリ判定ヘルパーメソッドを提供（IsImage, IsVideo等） |

---

### FileName

ファイル名を表す値オブジェクト。

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | ファイル名文字列 |
| extension | string | 拡張子（".pdf"など） |

**要件:**

| ID | 要件 |
|----|------|
| R-FN001 | 1-255バイト（UTF-8） |
| R-FN002 | 禁止文字（`/ \ : * ? " < > |`）を含まない |
| R-FN003 | 先頭・末尾の空白はトリム |
| R-FN004 | 空文字は不可 |
| R-FN005 | 拡張子は最後の`.`以降から取得 |

---

### FileStatus

ファイル状態を表すenum。

| 値 | 説明 |
|-----|------|
| uploading | アップロード中 |
| active | アクティブ |
| upload_failed | アップロード失敗 |

---

### UploadStatus

アップロードセッション状態を表すenum。

| 値 | 説明 |
|-----|------|
| pending | 初期化済み、未開始 |
| in_progress | アップロード中 |
| completed | 完了 |
| aborted | 中断 |
| expired | 期限切れ |

---

## 定数

| 定数名 | 値 | 説明 |
|--------|-----|------|
| TrashRetentionDays | 30 | ゴミ箱保持期間（日） |
| UploadSessionTTL | 24時間 | アップロードセッション有効期限 |
| MultipartThreshold | 5MB | マルチパートアップロード切替閾値 |
| MinPartSize | 5MB | 最小パートサイズ |
| MaxPartSize | 5GB | 最大パートサイズ |
| StorageKeyMaxBytes | 1024 | StorageKey最大長 |
| FileNameMaxBytes | 255 | FileName最大長 |
| UploadStatusPollInterval | 1秒 | クライアントのステータス確認間隔 |
| UploadStatusPollTimeout | 30秒 | クライアントのステータス確認タイムアウト |

---

## 操作フロー

### アップロードフロー

アップロード完了の検知はMinIO Bucket Notification（Webhook）を使用する。
クライアントからの完了通知は不要で、サーバーが一意に完了を判断する。

#### シングルパートアップロード（5MB未満）

```
1. クライアント → API: InitiateUpload
2. API:
   - UploadSession作成（is_multipart=false, status=pending）
   - File作成（status=uploading）
   - Presigned PUT URL生成
3. API → クライアント: session_id, presigned_url返却
4. クライアント → MinIO: 直接PUT
5. MinIO → API: Webhook通知（s3:ObjectCreated:Put）
6. API（内部処理 HandleUploadCompleted）:
   - storage_keyからUploadSession特定
   - File.status = active
   - FileVersion作成（minio_version_id保存）
   - UploadSession.status = completed
7. クライアント → API: GetUploadStatus（ポーリング、1秒間隔）
8. API → クライアント: status=completed, file_id返却
```

#### マルチパートアップロード（5MB以上）

```
1. クライアント → API: InitiateUpload（size >= 5MB）
2. API:
   - MinIO InitiateMultipartUpload
   - UploadSession作成（is_multipart=true, minio_upload_id保存, status=pending）
   - File作成（status=uploading）
3. API → クライアント: session_id, 各パートのpresigned_url返却
4. クライアント → MinIO: パートPUT（並列可）
5. MinIO → API: Webhook通知（s3:ObjectCreated:Put per part）
6. API（内部処理 HandlePartUploaded）:
   - UploadPart作成（part_number, etag保存）
   - uploaded_parts++
   - UploadSession.status = in_progress
7. （全パーツ完了まで4-6を繰り返し）
8. API: uploaded_parts == total_partsを検知
9. API（内部処理）:
   - MinIO CompleteMultipartUpload（全パーツのETag送信）
10. MinIO → API: Webhook通知（最終オブジェクト作成）
11. API（内部処理 HandleUploadCompleted）:
    - File.status = active
    - FileVersion作成
    - UploadSession.status = completed
12. クライアント → API: GetUploadStatus（ポーリング）
13. API → クライアント: status=completed, file_id返却
```

#### クライアント側ポーリング仕様

| 項目 | 値 |
|------|-----|
| ポーリング間隔 | 1秒 |
| タイムアウト | 30秒 |
| リトライ | タイムアウト後、エラーとして処理 |

**GetUploadStatusレスポンス:**

| フィールド | 型 | 説明 |
|-----------|-----|------|
| session_id | UUID | セッションID |
| status | UploadStatus | pending, in_progress, completed, failed, expired |
| file_id | UUID | completed時のみ、作成されたファイルID |
| progress | object | マルチパート時の進捗（uploaded_parts, total_parts） |
| error | string | failed時のみ、エラー理由 |

### ゴミ箱操作フロー

#### Trash（ゴミ箱へ移動）

```
1. files → archived_files へデータコピー
2. file_versions → archived_file_versions へ全バージョンコピー
3. file_versions 削除（CASCADE）
4. files から削除
5. MinIOオブジェクトは削除しない
```

#### Restore（復元）

```
1. archived_files → files へデータコピー
2. archived_file_versions → file_versions へ全バージョンコピー
3. archived_file_versions 削除（CASCADE）
4. archived_files から削除
5. MinIOオブジェクトはそのまま
```

#### PermanentDelete（完全削除）

```
1. MinIOから全バージョン削除
2. archived_file_versions 削除（CASCADE）
3. archived_files から削除
```

### 自動クリーンアップ

#### 期限切れゴミ箱の削除（日次バッチ）

```
1. archived_files から expires_at < NOW() を検索
2. 各ファイルに対してPermanentDelete実行
```

#### 期限切れアップロードセッションの処理（定期バッチ）

```
1. upload_sessions から expires_at < NOW() かつ status IN (pending, in_progress) を検索
2. マルチパートの場合: MinIO AbortMultipartUpload
3. 関連するupload_parts削除
4. UploadSession.status = expired
5. 関連するFile削除（status=uploadingの場合）
```

---

## リポジトリ

### FileRepository

| 操作 | 説明 |
|-----|------|
| Create | ファイル作成 |
| FindByID | ID検索 |
| Update | 更新 |
| Delete | 物理削除 |
| FindByFolderID | フォルダ内ファイル一覧 |
| FindByNameAndFolder | フォルダ内の同名ファイル検索 |
| FindByOwner | 所有者のファイル一覧 |
| Search | 全文検索 |

### FileVersionRepository

| 操作 | 説明 |
|-----|------|
| Create | バージョン作成 |
| FindByID | ID検索 |
| FindByFileID | ファイルの全バージョン取得 |
| FindByFileAndVersion | 特定バージョン取得 |
| Delete | 物理削除 |
| BulkCreate | 一括作成（復元用） |

### ArchivedFileRepository

| 操作 | 説明 |
|-----|------|
| Create | アーカイブ作成 |
| FindByID | ID検索 |
| FindByOwner | 所有者のゴミ箱一覧 |
| FindExpired | 期限切れファイル検索 |
| Delete | 物理削除 |

### ArchivedFileVersionRepository

| 操作 | 説明 |
|-----|------|
| BulkCreate | 一括作成 |
| FindByArchivedFileID | アーカイブファイルの全バージョン取得 |
| DeleteByArchivedFileID | アーカイブファイルのバージョン一括削除 |

### UploadSessionRepository

| 操作 | 説明 |
|-----|------|
| Create | セッション作成 |
| FindByID | ID検索 |
| FindByStorageKey | StorageKeyで検索（Webhook処理用） |
| Update | 更新 |
| FindExpired | 期限切れセッション検索 |
| Delete | 物理削除 |

### UploadPartRepository

| 操作 | 説明 |
|-----|------|
| Create | パーツ作成 |
| FindBySessionID | セッションの全パーツ取得 |
| DeleteBySessionID | セッションのパーツ一括削除 |

---

## 不変条件

### ストレージ整合性

| ID | 不変条件 |
|----|---------|
| I-ST001 | storage_keyはfile_idと1:1対応 |
| I-ST002 | 完全削除時、MinIOの全バージョンも削除 |
| I-ST003 | checksumはアップロード完了時に必ず計算・保存 |
| I-ST004 | MinIOバージョニングを使用し、バージョンごとにminio_version_idを保持 |

### バージョン整合性

| ID | 不変条件 |
|----|---------|
| I-VR001 | version_numberは連番（ギャップなし） |
| I-VR002 | current_versionは存在するバージョンを指す |
| I-VR003 | 全バージョンはminio_version_idで個別にアクセス可能 |

### ステータス整合性

| ID | 不変条件 |
|----|---------|
| I-SS001 | uploading状態のファイルはダウンロード不可 |
| I-SS002 | upload_failed状態のファイルは自動クリーンアップ対象 |
| I-SS003 | ゴミ箱のファイルは30日後に自動削除 |

### 命名制約

| ID | 不変条件 |
|----|---------|
| I-NM001 | 同一フォルダ内でファイル名は一意 |
| I-NM002 | ファイル名に禁止文字を含まない |

### 所有権制約

| ID | 不変条件 |
|----|---------|
| I-OW001 | ファイルは必ず所有者（owner_id）を持つ |
| I-OW002 | ファイルは必ず作成者（created_by）を持つ |
| I-OW003 | created_byは不変（所有権譲渡後も変更されない） |
| I-OW004 | 新規作成時は owner_id = created_by = 作成者 |

### フォルダ制約

| ID | 不変条件 |
|----|---------|
| I-FO001 | ファイルは必ずフォルダに所属する（folder_id NOT NULL） |

### アップロードセッション

| ID | 不変条件 |
|----|---------|
| I-UP001 | 期限切れセッションは自動キャンセル |
| I-UP002 | マルチパートアップロード中断時はMinIO側もAbort |
| I-UP003 | アップロード完了はMinIO Webhookでのみ判定（クライアント通知は使用しない） |
| I-UP004 | マルチパートの全パーツ完了時、サーバーがCompleteMultipartUploadを実行 |

---

## ユースケース

### クライアント向けAPI

| ユースケース | アクター | 概要 |
|------------|--------|------|
| InitiateUpload | User | アップロード開始（session_id, Presigned URL取得） |
| GetUploadStatus | User | アップロード状態確認（ポーリング用） |
| AbortUpload | User | アップロードキャンセル |
| DownloadFile | User | ファイルダウンロード |
| RenameFile | User | ファイル名変更 |
| MoveFile | User | ファイル移動 |
| TrashFile | User | ゴミ箱へ移動 |
| RestoreFile | User | ゴミ箱から復元 |
| PermanentlyDeleteFile | User | 完全削除 |
| ListVersions | User | バージョン一覧 |
| DownloadVersion | User | 特定バージョンダウンロード |
| SearchFiles | User | ファイル検索 |
| ListTrash | User | ゴミ箱一覧 |
| EmptyTrash | User | ゴミ箱を空にする |

### 内部ハンドラー（MinIO Webhook受信）

| ハンドラー | トリガー | 概要 |
|-----------|---------|------|
| HandleUploadCompleted | s3:ObjectCreated:Put | アップロード完了処理（File/FileVersion作成） |
| HandlePartUploaded | s3:ObjectCreated:Put (part) | マルチパートのパーツ完了処理 |

---

## ドメインイベント

| イベント | トリガー | ペイロード |
|---------|---------|-----------|
| FileUploadStarted | アップロード開始 | fileId, sessionId, name, size |
| FileUploaded | アップロード完了 | fileId, name, size, versionNumber |
| FileUploadFailed | アップロード失敗 | fileId, sessionId, reason |
| FileRenamed | 名前変更 | fileId, oldName, newName |
| FileMoved | 移動 | fileId, oldFolderId, newFolderId |
| FileArchived | ゴミ箱移動 | fileId, archivedFileId |
| FileRestored | 復元 | archivedFileId, fileId |
| FilePermanentlyDeleted | 完全削除 | archivedFileId |
| FileVersionCreated | バージョン作成 | fileId, versionNumber |

---

## 他コンテキストとの連携

### Folder Domain（同一コンテキスト）

- ファイルは必ずフォルダに所属（folder_id NOT NULL）
- フォルダ削除時、配下のファイルもアーカイブ

### Identity Context（上流）

- UserIDの参照（owner_id, created_by, uploaded_by）

### Authorization Context（下流）

- ファイルに対する権限付与（PermissionGrant）
- 親フォルダからの権限継承
- ロール: Viewer / Contributor / Content Manager / Owner

### Collaboration Context（上流）

- グループにファイルへのロールを付与（PermissionGrant経由）
- ※ グループはファイルを直接所有しない

### Sharing Context（下流）

- ファイルの共有リンク作成

---

## インフラ要件

### MinIO Bucket Notification設定

アップロード完了検知のため、MinIOにWebhook通知を設定する必要がある。

| 設定項目 | 値 |
|---------|-----|
| イベントタイプ | s3:ObjectCreated:* |
| 通知先 | API Webhookエンドポイント |
| 対象バケット | ファイルストレージ用バケット |

**設定コマンド例:**
```
mc event add myminio/files arn:minio:sqs::primary:webhook --event put
```

**Webhookエンドポイント:**
```
POST /internal/webhooks/minio
```

**要件:**

| ID | 要件 |
|----|------|
| R-MN001 | MinIOからのWebhook通知を受信するエンドポイントを実装 |
| R-MN002 | Webhook通知の認証・検証を行う |
| R-MN003 | 通知の冪等性を保証（同一イベントの重複処理を防止） |
| R-MN004 | 同期モード（MINIO_API_SYNC_EVENTS=on）を推奨 |

---

## 関連ドキュメント

- [イベントストーミング](./EVENT_STORMING.md) - ドメインイベント定義
- [フォルダドメイン](./folder.md) - フォルダ管理
- [共有ドメイン](./sharing.md) - 共有機能

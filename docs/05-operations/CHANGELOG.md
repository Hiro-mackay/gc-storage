# Changelog

このプロジェクトのすべての重要な変更はこのファイルに記録されます。

フォーマットは [Keep a Changelog](https://keepachangelog.com/ja/1.0.0/) に基づいており、
このプロジェクトは [Semantic Versioning](https://semver.org/lang/ja/) に準拠しています。

---

## [Unreleased]

### 追加予定
- ファイル内容の全文検索（Elasticsearch連携）
- リアルタイム同期（WebSocket）
- ストレージ使用量のクォータ管理
- 2要素認証（2FA）

---

## [1.0.0] - 2026-XX-XX

### Added（追加）

#### 認証・認可
- JWT ベースの認証システム
- Google / GitHub OAuth 2.0 連携
- PBAC + ReBAC ハイブリッド認可モデル
- ロールベースの権限管理（viewer, contributor, content_manager, owner）

#### ファイル管理
- ファイルアップロード（Presigned URL方式）
- マルチパートアップロード（大容量ファイル対応、最大5GB）
- ファイルダウンロード
- ファイルプレビュー（画像、PDF）
- ファイルのバージョン管理
- ファイルメタデータ（サイズ、MIMEタイプ、ハッシュ値）

#### フォルダ管理
- フォルダの作成・削除・名前変更
- フォルダの移動
- フォルダ階層のナビゲーション（パンくずリスト）

#### 共有機能
- 共有リンクの生成
- 期限付き共有リンク
- パスワード保護付き共有リンク

#### グループ機能
- グループの作成・管理
- グループメンバーの招待・削除
- グループロール管理（viewer, contributor, owner）
- グループフォルダの共有

#### 検索・フィルタリング
- ファイル名検索（部分一致）
- 拡張子フィルタリング
- 更新日時フィルタリング

#### インフラストラクチャ
- Kubernetes ベースのデプロイメント
- PostgreSQL データベース
- Redis キャッシュ・セッションストア
- MinIO オブジェクトストレージ（S3互換）
- Prometheus + Grafana 監視

#### API
- RESTful API（OpenAPI 3.0仕様）
- レート制限
- ページネーション
- 統一エラーレスポンス

---

## リリース計画

### [1.1.0] - 予定

#### 追加予定
- ゴミ箱機能（論理削除 + 復元）
- ファイルのお気に入り機能
- 最近使ったファイル一覧
- ファイルのタグ付け

#### 改善予定
- 検索パフォーマンスの向上
- アップロード進捗表示の改善
- モバイルレスポンシブ対応の強化

### [1.2.0] - 予定

#### 追加予定
- ファイルコメント機能
- アクティビティログ（ユーザー向け）
- 通知機能（メール、アプリ内）
- ファイルリクエスト機能

### [2.0.0] - 予定

#### 追加予定
- 全文検索（Elasticsearch）
- リアルタイム同期（WebSocket）
- デスクトップクライアント
- モバイルアプリ

#### Breaking Changes（予定）
- API v2 への移行
- 認証フローの変更

---

## バージョニングポリシー

### Semantic Versioning

```
MAJOR.MINOR.PATCH

MAJOR: 後方互換性のない変更
MINOR: 後方互換性のある機能追加
PATCH: 後方互換性のあるバグ修正
```

### プレリリース

```
1.0.0-alpha.1  : アルファ版（内部テスト）
1.0.0-beta.1   : ベータ版（限定公開テスト）
1.0.0-rc.1     : リリース候補
```

---

## 変更の種類

| タイプ | 説明 |
|--------|------|
| Added | 新機能の追加 |
| Changed | 既存機能の変更 |
| Deprecated | 将来削除予定の機能 |
| Removed | 削除された機能 |
| Fixed | バグ修正 |
| Security | セキュリティ関連の修正 |

---

## 関連リンク

- [GitHub Releases](https://github.com/Hiro-mackay/gc-storage/releases)
- [Migration Guide](./MIGRATION_GUIDE.md) - バージョンアップガイド
- [API Documentation](./API.md) - API仕様

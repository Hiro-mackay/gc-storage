# 運用ドキュメント

> このディレクトリには、本番環境の運用・保守に必要な情報を記載します。

---

## 概要

GC Storageの運用に必要なデプロイ手順、監視設定、障害対応、コンプライアンス対応を管理します。

---

## ドキュメント一覧

| ファイル | 内容 | 参照タイミング |
|---------|------|--------------|
| [DEPLOYMENT.md](./DEPLOYMENT.md) | デプロイ手順・CI/CD・ロールバック | リリース時 |
| [OPERATIONS.md](./OPERATIONS.md) | 日常運用タスク・監視・バックアップ | 定期運用時 |
| [INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md) | 障害対応フロー・エスカレーション | 障害発生時 |
| [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) | よくある問題と解決策 | 問題発生時 |
| [COMPLIANCE.md](./COMPLIANCE.md) | GDPR対応・データ保持ポリシー | コンプライアンス確認時 |
| [CHANGELOG.md](./CHANGELOG.md) | リリース履歴・変更ログ | バージョン確認時 |
| [MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md) | バージョンアップ手順 | アップグレード時 |
| [ROADMAP.md](./ROADMAP.md) | プロダクトロードマップ | 計画確認時 |

---

## クイックリファレンス

### 緊急時対応

```
1. INCIDENT_RESPONSE.md で障害レベルを判定
2. エスカレーションフローに従って連絡
3. TROUBLESHOOTING.md で解決策を確認
```

### 定期リリース

```
1. CHANGELOG.md でリリース内容を確認
2. DEPLOYMENT.md の手順に従ってデプロイ
3. OPERATIONS.md の監視ダッシュボードで確認
```

### バージョンアップ

```
1. MIGRATION_GUIDE.md で変更点を確認
2. ステージング環境でテスト
3. DEPLOYMENT.md に従って本番適用
```

---

## 環境一覧

| 環境 | 用途 | URL |
|------|------|-----|
| Development | ローカル開発 | localhost:3000 |
| Staging | 検証環境 | staging.example.com |
| Production | 本番環境 | app.example.com |

---

## 連絡先

| 担当 | 連絡先 |
|------|--------|
| インフラチーム | infra@example.com |
| セキュリティチーム | security@example.com |
| オンコール | oncall@example.com |

---

## 関連ドキュメント

- [アーキテクチャ/INFRASTRUCTURE.md](../02-architecture/INFRASTRUCTURE.md) - インフラ設計
- [アーキテクチャ/SECURITY.md](../02-architecture/SECURITY.md) - セキュリティ設計
- [ポリシー/SETUP.md](../01-policies/SETUP.md) - 開発環境セットアップ

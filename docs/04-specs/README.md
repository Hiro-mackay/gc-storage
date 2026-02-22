# 機能仕様レイヤー (04-specs)

> Feature-based の仕様書構造。1つの機能をフルサイクル（Domain -> API -> UI -> Test）で開発できる。

---

## 概要

機能ごとのフルスタック仕様書を管理します。各 Feature Spec には以下が含まれます:

1. **User Stories** - ユーザー視点の要求
2. **Domain Behaviors** - `03-domains/` への参照 + feature固有ルール
3. **API Contract** - エンドポイント、DTO、エラー
4. **Frontend UI** - レイアウト、コンポーネント、状態管理
5. **Integration Flow** - シーケンス図、状態フロー
6. **Acceptance Criteria** - フルスタックでテスト可能な基準
7. **Test Plan** - backend unit/integration + frontend + E2E
8. **Implementation Notes** - 変更対象ファイル一覧

**粒度**: 1つの Feature Spec = 複数PRにまたがる可能性あり（API + UI）

---

## ディレクトリ構造

| ディレクトリ | 内容 |
|------------|------|
| `infra/` | インフラ基盤仕様（PostgreSQL, Redis, MinIO, SMTP, API base） |
| `platform/` | プラットフォーム共通仕様（OpenAPI型生成, FE基盤） |
| `features/` | Feature specs（フルスタック仕様書） |
| `templates/` | テンプレート（Feature spec, Test spec） |
| `archive/` | 旧specファイル（参照用） |

詳細は [SPEC_MAP.md](./SPEC_MAP.md) を参照。

---

## Feature Spec の作成方法

1. [FEATURE_SPEC_TEMPLATE.md](./templates/FEATURE_SPEC_TEMPLATE.md) をコピー
2. 対応する `03-domains/*.md` を参照してドメインルールを確認
3. 8セクション全てを埋める
4. [SPEC_MAP.md](./SPEC_MAP.md) にリンクを追加

---

## ステータス定義

| ステータス | 説明 |
|-----------|------|
| Draft | 作成中 |
| Ready | レビュー済み、実装可能 |
| In Progress | 実装中 |
| Done | 実装完了 |

---

## 関連ドキュメント

- [SPEC_MAP.md](./SPEC_MAP.md) - 仕様マップ（全Feature Specの一覧と依存関係）
- [03-domains/](../03-domains/) - ドメイン定義
- [02-architecture/](../02-architecture/) - アーキテクチャ設計
- [01-policies/](../01-policies/) - 開発ポリシー

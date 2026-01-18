package storage

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config はMinIO接続設定を定義します
type Config struct {
	Endpoint        string // MinIOエンドポイント (例: localhost:9000)
	AccessKeyID     string // アクセスキーID
	SecretAccessKey string // シークレットアクセスキー
	BucketName      string // バケット名
	UseSSL          bool   // SSL使用有無
	Region          string // リージョン (default: us-east-1)
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() Config {
	return Config{
		UseSSL: false,
		Region: "us-east-1",
	}
}

// MinIOClient はMinIO操作を提供します
type MinIOClient struct {
	client *minio.Client
	config Config
}

// NewMinIOClient は新しいMinIOClientを作成します
func NewMinIOClient(cfg Config) (*MinIOClient, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinIOClient{
		client: client,
		config: cfg,
	}, nil
}

// Client は内部のminio.Clientを返します
func (m *MinIOClient) Client() *minio.Client {
	return m.client
}

// BucketName はバケット名を返します
func (m *MinIOClient) BucketName() string {
	return m.config.BucketName
}

// Config は設定を返します
func (m *MinIOClient) Config() Config {
	return m.config
}

// Health はMinIOの接続状態を確認します
func (m *MinIOClient) Health(ctx context.Context) error {
	_, err := m.client.BucketExists(ctx, m.config.BucketName)
	return err
}

// EnsureBucket はバケットが存在しない場合は作成します
func (m *MinIOClient) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.config.BucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, m.config.BucketName, minio.MakeBucketOptions{
			Region: m.config.Region,
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

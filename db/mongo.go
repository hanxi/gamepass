package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	client *mongo.Client
	db     *mongo.Database
}

func NewMongoDB(uri, dbName string) (*MongoDB, error) {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err = client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	mongodb := &MongoDB{
		client: client,
		db:     db,
	}

	// 初始化索引
	if err := mongodb.setupIndexes(ctx); err != nil {
		return nil, err
	}

	return mongodb, nil
}

func (m *MongoDB) setupIndexes(ctx context.Context) error {
	// 用户集合索引
	if err := m.setupUserIndexes(ctx); err != nil {
		return err
	}

	// Token 黑名单索引
	if err := m.setupTokenBlacklistIndexes(ctx); err != nil {
		return err
	}

	return nil
}

func (m *MongoDB) setupUserIndexes(ctx context.Context) error {
	_, err := m.db.Collection("users").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "provider", Value: 1},
				{Key: "provider_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "email", Value: 1}},
		},
	})
	return err
}

func (m *MongoDB) setupTokenBlacklistIndexes(ctx context.Context) error {
	_, err := m.db.Collection("token_blacklist").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	})
	return err
}

func (m *MongoDB) GetDB() *mongo.Database {
	return m.db
}

func (m *MongoDB) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

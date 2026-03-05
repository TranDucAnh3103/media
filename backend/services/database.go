package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB - Global MongoDB client instance
var MongoDB *mongo.Database

// Collections
var (
	UsersCollection  *mongo.Collection
	ComicsCollection *mongo.Collection
	VideosCollection *mongo.Collection
)

// ConnectDB - Kết nối tới MongoDB Atlas
func ConnectDB() error {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return fmt.Errorf("MONGO_URI environment variable not set")
	}

	// Cấu hình client
	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(50).
		SetMinPoolSize(5).
		SetMaxConnecting(10)

	// Context với timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Kết nối
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping để kiểm tra kết nối
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Lấy database
	dbName := os.Getenv("MONGO_DB_NAME")
	if dbName == "" {
		dbName = "media_db"
	}
	MongoDB = client.Database(dbName)

	// Khởi tạo collections
	UsersCollection = MongoDB.Collection("users")
	ComicsCollection = MongoDB.Collection("comics")
	VideosCollection = MongoDB.Collection("videos")

	// Tạo indexes
	if err := createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	fmt.Println("✅ Connected to MongoDB Atlas")
	return nil
}

// createIndexes - Tạo indexes cho collections
func createIndexes(ctx context.Context) error {
	// Users indexes
	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}
	if _, err := UsersCollection.Indexes().CreateMany(ctx, userIndexes); err != nil {
		return err
	}

	// Comics indexes - Text search + filter
	comicIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "title", Value: "text"},
				{Key: "description", Value: "text"},
				{Key: "author", Value: "text"},
			},
			Options: options.Index().SetName("comics_text_search"),
		},
		{Keys: bson.D{{Key: "genres", Value: 1}}},
		{Keys: bson.D{{Key: "tags", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "views", Value: -1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	}
	if _, err := ComicsCollection.Indexes().CreateMany(ctx, comicIndexes); err != nil {
		return err
	}

	// Videos indexes - Text search + filter
	videoIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "title", Value: "text"},
				{Key: "description", Value: "text"},
			},
			Options: options.Index().SetName("videos_text_search"),
		},
		{Keys: bson.D{{Key: "genres", Value: 1}}},
		{Keys: bson.D{{Key: "tags", Value: 1}}},
		{Keys: bson.D{{Key: "duration_type", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "views", Value: -1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	}
	if _, err := VideosCollection.Indexes().CreateMany(ctx, videoIndexes); err != nil {
		return err
	}

	return nil
}

// DisconnectDB - Ngắt kết nối MongoDB
func DisconnectDB() {
	if MongoDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		MongoDB.Client().Disconnect(ctx)
	}
}

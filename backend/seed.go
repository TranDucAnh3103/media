//go:build seed
// +build seed

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// User model
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Username  string             `bson:"username"`
	Email     string             `bson:"email"`
	Password  string             `bson:"password"`
	Role      string             `bson:"role"`
	Avatar    string             `bson:"avatar"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

// Video model
type Video struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Title        string             `bson:"title"`
	Description  string             `bson:"description"`
	Thumbnail    string             `bson:"thumbnail"`
	VideoURL     string             `bson:"video_url"`
	StorageType  string             `bson:"storage_type"`
	Duration     int                `bson:"duration"`
	DurationType string             `bson:"duration_type"`
	Quality      string             `bson:"quality"`
	FileSize     int64              `bson:"file_size"`
	Tags         []string           `bson:"tags"`
	Genres       []string           `bson:"genres"`
	Views        int64              `bson:"views"`
	Likes        int64              `bson:"likes"`
	Dislikes     int64              `bson:"dislikes"`
	Status       string             `bson:"status"`
	UploadedBy   primitive.ObjectID `bson:"uploaded_by"`
	CreatedAt    time.Time          `bson:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at"`
}

// Comic model
type Comic struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Title       string             `bson:"title"`
	Description string             `bson:"description"`
	Author      string             `bson:"author"`
	CoverImage  string             `bson:"cover_image"`
	Tags        []string           `bson:"tags"`
	Genres      []string           `bson:"genres"`
	Status      string             `bson:"status"`
	Chapters    []Chapter          `bson:"chapters"`
	Views       int64              `bson:"views"`
	Likes       int64              `bson:"likes"`
	Rating      float64            `bson:"rating"`
	RatingCount int64              `bson:"rating_count"`
	UploadedBy  primitive.ObjectID `bson:"uploaded_by"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

// Chapter model
type Chapter struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Number     int                `bson:"number"`
	Title      string             `bson:"title"`
	Images     []ComicImage       `bson:"images"`
	Views      int64              `bson:"views"`
	UploadedAt time.Time          `bson:"uploaded_at"`
}

// ComicImage model
type ComicImage struct {
	Page     int    `bson:"page"`
	URL      string `bson:"url"`
	PublicID string `bson:"public_id"`
	Width    int    `bson:"width"`
	Height   int    `bson:"height"`
}

func main() {
	// Load env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Support both MONGO_URI and MONGODB_URI
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = os.Getenv("MONGODB_URI")
	}
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	dbName := os.Getenv("MONGO_DB_NAME")
	if dbName == "" {
		dbName = "mediahub"
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer client.Disconnect(ctx)

	// Ping database
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	db := client.Database(dbName)
	fmt.Printf("✓ Connected to MongoDB (database: %s)\n", dbName)

	// Clear ALL existing data
	fmt.Println("→ Clearing ALL existing data...")
	db.Collection("users").DeleteMany(ctx, bson.M{})
	db.Collection("videos").DeleteMany(ctx, bson.M{})
	db.Collection("comics").DeleteMany(ctx, bson.M{})
	fmt.Println("  ✓ All data cleared")

	// ============================================
	// 1. Create demo user and admin user
	// ============================================
	fmt.Println("→ Creating users...")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("demo123456"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	hashedAdminPassword, err := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash admin password:", err)
	}

	// Admin user
	adminUser := User{
		ID:        primitive.NewObjectID(),
		Username:  "Admin",
		Email:     "admin@mediahub.com",
		Password:  string(hashedAdminPassword),
		Role:      "admin",
		Avatar:    "https://api.dicebear.com/7.x/avataaars/svg?seed=Admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = db.Collection("users").InsertOne(ctx, adminUser)
	if err != nil {
		log.Fatal("Failed to create admin user:", err)
	}
	fmt.Printf("  ✓ Admin created: %s (%s)\n", adminUser.Username, adminUser.Email)
	fmt.Println("    Password: admin123456")

	// Demo user
	demoUser := User{
		ID:        primitive.NewObjectID(),
		Username:  "DemoUser",
		Email:     "demo@mediahub.com",
		Password:  string(hashedPassword),
		Role:      "user",
		Avatar:    "https://api.dicebear.com/7.x/avataaars/svg?seed=DemoUser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = db.Collection("users").InsertOne(ctx, demoUser)
	if err != nil {
		log.Fatal("Failed to create user:", err)
	}
	fmt.Printf("  ✓ User created: %s (%s)\n", demoUser.Username, demoUser.Email)
	fmt.Println("    Password: demo123456")


	// ============================================
	// Summary 
	// ============================================
	fmt.Println("\n" + "═══════════════════════════════════════════")
	fmt.Println("  🎉 Seed completed successfully!")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("\n📋 Demo Account:")
	fmt.Println("   Email:    demo@mediahub.com")
	fmt.Println("   Password: demo123456")
	fmt.Println("\n📺 Demo Video:")
	fmt.Printf("   Title: %s\n", demoVideo.Title)
	fmt.Printf("   Duration: %d seconds\n", demoVideo.Duration)
	fmt.Println("\n📚 Demo Comic:")
	fmt.Printf("   Title: %s\n", demoComic.Title)
	fmt.Printf("   Chapters: %d (with %d pages)\n", len(demoComic.Chapters), len(sampleImages))
	fmt.Println()
}

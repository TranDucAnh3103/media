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
	// 1. Create demo user
	// ============================================
	fmt.Println("→ Creating demo user...")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("demo123456"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

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
	// 2. Create demo video (YouTube - 2 minutes)
	// ============================================
	fmt.Println("→ Creating demo video...")

	demoVideo := Video{
		ID:           primitive.NewObjectID(),
		Title:        "Relaxing Nature - Beautiful Scenery",
		Description:  "Enjoy 2 minutes of relaxing nature scenery with calming music. Perfect for meditation and relaxation.",
		Thumbnail:    "https://img.youtube.com/vi/lM02vNMRRB0/maxresdefault.jpg",
		VideoURL:     "https://www.youtube.com/watch?v=lM02vNMRRB0",
		StorageType:  "youtube",
		Duration:     120,
		DurationType: "short",
		Quality:      "1080p",
		FileSize:     0,
		Tags:         []string{"nature", "relaxing", "scenery", "meditation", "calm"},
		Genres:       []string{"Nature", "Relaxation"},
		Views:        156,
		Likes:        12,
		Dislikes:     0,
		Status:       "ready",
		UploadedBy:   demoUser.ID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = db.Collection("videos").InsertOne(ctx, demoVideo)
	if err != nil {
		log.Fatal("Failed to create video:", err)
	}
	fmt.Printf("  ✓ Video created: %s\n", demoVideo.Title)
	fmt.Printf("    URL: %s\n", demoVideo.VideoURL)

	// ============================================
	// 3. Create demo comic with 5 sample images
	// ============================================
	fmt.Println("→ Creating demo comic...")

	// Sample manga/comic pages from Lorem Picsum (placeholder images)
	sampleImages := []ComicImage{
		{Page: 1, URL: "https://picsum.photos/seed/comic1/800/1200", PublicID: "comic1", Width: 800, Height: 1200},
		{Page: 2, URL: "https://picsum.photos/seed/comic2/800/1200", PublicID: "comic2", Width: 800, Height: 1200},
		{Page: 3, URL: "https://picsum.photos/seed/comic3/800/1200", PublicID: "comic3", Width: 800, Height: 1200},
		{Page: 4, URL: "https://picsum.photos/seed/comic4/800/1200", PublicID: "comic4", Width: 800, Height: 1200},
		{Page: 5, URL: "https://picsum.photos/seed/comic5/800/1200", PublicID: "comic5", Width: 800, Height: 1200},
	}

	demoComic := Comic{
		ID:          primitive.NewObjectID(),
		Title:       "Hành Trình Phiêu Lưu",
		Description: "Câu chuyện kể về một chàng trai trẻ bắt đầu cuộc hành trình khám phá thế giới bí ẩn. Với sự dũng cảm và trái tim nhân hậu, anh ta sẽ vượt qua mọi thử thách để tìm ra sự thật về quá khứ của mình.",
		Author:      "Nguyễn Văn A",
		CoverImage:  "https://picsum.photos/seed/comiccover/400/600",
		Tags:        []string{"phiêu lưu", "hành động", "fantasy", "shounen"},
		Genres:      []string{"Adventure", "Fantasy", "Action"},
		Status:      "ongoing",
		Chapters: []Chapter{
			{
				ID:         primitive.NewObjectID(),
				Number:     1,
				Title:      "Khởi Đầu Mới",
				Images:     sampleImages,
				Views:      45,
				UploadedAt: time.Now(),
			},
		},
		Views:       89,
		Likes:       7,
		Rating:      4.5,
		RatingCount: 10,
		UploadedBy:  demoUser.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = db.Collection("comics").InsertOne(ctx, demoComic)
	if err != nil {
		log.Fatal("Failed to create comic:", err)
	}
	fmt.Printf("  ✓ Comic created: %s\n", demoComic.Title)
	fmt.Printf("    Chapter 1 with %d pages\n", len(sampleImages))

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

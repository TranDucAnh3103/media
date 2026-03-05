package controllers

import (
	"context"
	"os"
	"time"

	"media-backend/models"
	"media-backend/services"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// UserController - Controller xử lý user operations
type UserController struct{}

// NewUserController - Tạo instance mới
func NewUserController() *UserController {
	return &UserController{}
}

// Register - Đăng ký tài khoản mới
// POST /api/auth/register
func (c *UserController) Register(ctx *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Username, email, and password are required",
		})
	}

	if len(req.Password) < 6 {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Password must be at least 6 characters",
		})
	}

	// Kiểm tra email đã tồn tại
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var existingUser models.User
	err := services.UsersCollection.FindOne(dbCtx, bson.M{"email": req.Email}).Decode(&existingUser)
	if err == nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Email already registered",
		})
	}

	// Kiểm tra username đã tồn tại
	err = services.UsersCollection.FindOne(dbCtx, bson.M{"username": req.Username}).Decode(&existingUser)
	if err == nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Username already taken",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	// Tạo user mới
	now := time.Now()
	user := models.User{
		ID:        primitive.NewObjectID(),
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		Role:      "user",
		Bookmarks: []models.Bookmark{},
		Playlists: []models.Playlist{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Insert vào database
	_, err = services.UsersCollection.InsertOne(dbCtx, user)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	// Tạo JWT token
	token, err := generateJWT(user.ID.Hex(), user.Role)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return ctx.Status(201).JSON(fiber.Map{
		"message": "Registration successful",
		"token":   token,
		"user":    user.ToResponse(),
	})
}

// Login - Đăng nhập
// POST /api/auth/login
func (c *UserController) Login(ctx *fiber.Ctx) error {
	var req models.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Tìm user theo email
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err := services.UsersCollection.FindOne(dbCtx, bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		return ctx.Status(401).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	// Kiểm tra password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return ctx.Status(401).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	// Tạo JWT token
	token, err := generateJWT(user.ID.Hex(), user.Role)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Login successful",
		"token":   token,
		"user":    user.ToResponse(),
	})
}

// GetProfile - Lấy thông tin profile
// GET /api/user/profile
func (c *UserController) GetProfile(ctx *fiber.Ctx) error {
	userID := ctx.Locals("userID").(string)

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err = services.UsersCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return ctx.JSON(user.ToResponse())
}

// UpdateProfile - Cập nhật profile
// PUT /api/user/profile
func (c *UserController) UpdateProfile(ctx *fiber.Ctx) error {
	userID := ctx.Locals("userID").(string)

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var updateData struct {
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	}
	if err := ctx.BodyParser(&updateData); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	if updateData.Username != "" {
		update["$set"].(bson.M)["username"] = updateData.Username
	}
	if updateData.Avatar != "" {
		update["$set"].(bson.M)["avatar"] = updateData.Avatar
	}

	_, err = services.UsersCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, update)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Profile updated successfully",
	})
}

// AddBookmark - Thêm bookmark
// POST /api/user/bookmarks
func (c *UserController) AddBookmark(ctx *fiber.Ctx) error {
	userID := ctx.Locals("userID").(string)

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var bookmark models.Bookmark
	if err := ctx.BodyParser(&bookmark); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	bookmark.CreatedAt = time.Now()

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Xóa bookmark cũ nếu đã tồn tại, sau đó thêm mới
	_, _ = services.UsersCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
		"$pull": bson.M{
			"bookmarks": bson.M{"content_id": bookmark.ContentID},
		},
	})

	_, err = services.UsersCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
		"$push": bson.M{"bookmarks": bookmark},
		"$set":  bson.M{"updated_at": time.Now()},
	})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to add bookmark",
		})
	}

	return ctx.JSON(fiber.Map{
		"message":  "Bookmark added successfully",
		"bookmark": bookmark,
	})
}

// RemoveBookmark - Xóa bookmark
// DELETE /api/user/bookmarks/:contentId
func (c *UserController) RemoveBookmark(ctx *fiber.Ctx) error {
	userID := ctx.Locals("userID").(string)
	contentID := ctx.Params("contentId")

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	contentObjID, err := primitive.ObjectIDFromHex(contentID)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid content ID",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = services.UsersCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
		"$pull": bson.M{
			"bookmarks": bson.M{"content_id": contentObjID},
		},
		"$set": bson.M{"updated_at": time.Now()},
	})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to remove bookmark",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Bookmark removed successfully",
	})
}

// CreatePlaylist - Tạo playlist mới
// POST /api/user/playlists
func (c *UserController) CreatePlaylist(ctx *fiber.Ctx) error {
	userID := ctx.Locals("userID").(string)

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := ctx.BodyParser(&req); err != nil || req.Name == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Playlist name is required",
		})
	}

	playlist := models.Playlist{
		ID:        primitive.NewObjectID(),
		Name:      req.Name,
		VideoIDs:  []primitive.ObjectID{},
		CreatedAt: time.Now(),
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = services.UsersCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
		"$push": bson.M{"playlists": playlist},
		"$set":  bson.M{"updated_at": time.Now()},
	})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to create playlist",
		})
	}

	return ctx.Status(201).JSON(fiber.Map{
		"message":  "Playlist created successfully",
		"playlist": playlist,
	})
}

// AddToPlaylist - Thêm video vào playlist
// POST /api/user/playlists/:playlistId/videos
func (c *UserController) AddToPlaylist(ctx *fiber.Ctx) error {
	userID := ctx.Locals("userID").(string)
	playlistID := ctx.Params("playlistId")

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	playlistObjID, err := primitive.ObjectIDFromHex(playlistID)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid playlist ID",
		})
	}

	var req struct {
		VideoID string `json:"video_id"`
	}
	if err := ctx.BodyParser(&req); err != nil || req.VideoID == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Video ID is required",
		})
	}

	videoObjID, err := primitive.ObjectIDFromHex(req.VideoID)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video ID",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = services.UsersCollection.UpdateOne(dbCtx,
		bson.M{"_id": objID, "playlists._id": playlistObjID},
		bson.M{
			"$addToSet": bson.M{"playlists.$.video_ids": videoObjID},
			"$set":      bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to add video to playlist",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Video added to playlist successfully",
	})
}

// generateJWT - Tạo JWT token
func generateJWT(userID string, role string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-change-me"
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 ngày
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

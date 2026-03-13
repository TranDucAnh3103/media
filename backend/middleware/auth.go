package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware - Middleware xác thực JWT token
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Lấy token từ header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Parse Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		tokenString := parts[1]

		// Verify token
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "default-secret-change-me"
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(401, "Invalid signing method")
			}
			return []byte(secret), nil
		})

		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, _ := claims["user_id"].(string)
			role, _ := claims["role"].(string)

			// Set user info in context
			c.Locals("userID", userID)
			c.Locals("role", role)

			return c.Next()
		}

		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid token claims",
		})
	}
}

// AdminMiddleware - Middleware kiểm tra quyền admin
func AdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("role")
		if role != "admin" {
			return c.Status(403).JSON(fiber.Map{
				"error": "Admin access required",
			})
		}
		return c.Next()
	}
}

// OptionalAuthMiddleware - Middleware auth tùy chọn (không bắt buộc)
// Dùng cho routes có thể truy cập cả public và authenticated
func OptionalAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Next()
		}

		tokenString := parts[1]
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "default-secret-change-me"
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(401, "Invalid signing method")
			}
			return []byte(secret), nil
		})

		if err == nil {
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				userID, _ := claims["user_id"].(string)
				role, _ := claims["role"].(string)
				c.Locals("userID", userID)
				c.Locals("role", role)
			}
		}

		return c.Next()
	}
}

// GetUserID - Lấy userID từ Locals an toàn (tránh panic khi type assertion)
// Trả về (userID, true) nếu có và hợp lệ; ("", false) nếu không
func GetUserID(c *fiber.Ctx) (string, bool) {
	v := c.Locals("userID")
	if v == nil {
		return "", false
	}
	id, ok := v.(string)
	return id, ok && id != ""
}

// RequireUserID - Lấy userID, trả 401 nếu không có
func RequireUserID(c *fiber.Ctx) (string, error) {
	id, ok := GetUserID(c)
	if !ok {
		return "", c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return id, nil
}

// RateLimitMiddleware - Middleware giới hạn request rate
// TODO: Implement proper rate limiting with Redis
func RateLimitMiddleware(maxRequests int, windowSeconds int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Simple implementation - use Redis in production
		return c.Next()
	}
}

package main

import (
	"log"
	"os"

	"media-backend/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

// Global WebSocket clients cho realtime notification
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan string)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Khởi tạo Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: 500 * 1024 * 1024, // 500MB limit cho upload video
	})

	// Middleware
	app.Use(logger.New())

	// CORS - Allow all origins for API access
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
		AllowCredentials: false, // Must be false when AllowOrigins is "*"
	}))

	// WebSocket endpoint cho realtime notifications
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		clients[c] = true
		defer func() {
			delete(clients, c)
			c.Close()
		}()

		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				break
			}
		}
	}))

	// Goroutine broadcast messages to all clients
	go func() {
		for {
			msg := <-broadcast
			for client := range clients {
				if err := client.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
					client.Close()
					delete(clients, client)
				}
			}
		}
	}()

	// Setup routes
	routes.SetupRoutes(app)

	// Get port from env or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

// BroadcastNotification gửi notification tới tất cả WebSocket clients
func BroadcastNotification(message string) {
	broadcast <- message
}

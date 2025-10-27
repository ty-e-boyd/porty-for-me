package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ty-e-boyd/porty-for-me/template"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

// User represents a subscriber to the newsletter
type User struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Email            string    `gorm:"uniqueIndex;not null" json:"email"`
	Name             string    `json:"name"`
	Subscribed       bool      `gorm:"default:true" json:"subscribed"`
	UnsubscribeToken string    `gorm:"uniqueIndex;not null" json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

type SubscribeRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type UnsubscribeRequest struct {
	Token string `json:"token"`
}

type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to database
	if err := connectDatabase(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	e := echo.New()

	// Little bit of middlewares for housekeeping
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())

	// This will initiate our template renderer
	template.NewTemplateRenderer(e, "public/*.html")

	// Routes
	e.GET("/", handleIndex)
	e.GET("/subscribe", handleSubscribePage)
	e.GET("/unsubscribe", handleUnsubscribe)

	// API Routes
	e.POST("/api/subscribe", handleSubscribeAPI)
	e.POST("/api/unsubscribe", handleUnsubscribeAPI)

	e.Logger.Fatal(e.Start(":4040"))
}

func connectDatabase() error {
	dbConnectString := os.Getenv("DB_CONNECT_STRING")
	if dbConnectString == "" {
		return fmt.Errorf("DB_CONNECT_STRING environment variable is required")
	}

	var err error
	db, err = gorm.Open(postgres.Open(dbConnectString), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the User model
	if err := db.AutoMigrate(&User{}); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✓ Database connection established")
	return nil
}

func handleIndex(c echo.Context) error {
	return c.Render(http.StatusOK, "index", nil)
}

func handleSubscribePage(c echo.Context) error {
	return c.Render(http.StatusOK, "subscribe", nil)
}

func handleUnsubscribe(c echo.Context) error {
	token := c.QueryParam("token")

	if token == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Unsubscribe token is required",
		})
	}

	// Find user by token
	var user User
	result := db.Where("unsubscribe_token = ?", token).First(&user)
	if result.Error != nil {
		log.Printf("Error finding user with token: %v", result.Error)
		return c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Invalid unsubscribe token",
		})
	}

	// Update subscription status
	user.Subscribed = false
	if err := db.Save(&user).Error; err != nil {
		log.Printf("Error unsubscribing user: %v", err)
		return c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to unsubscribe",
		})
	}

	log.Printf("User %s unsubscribed successfully", user.Email)

	// Render unsubscribe confirmation page
	return c.Render(http.StatusOK, "unsubscribe", nil)
}

func handleSubscribeAPI(c echo.Context) error {
	var req SubscribeRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Email is required",
		})
	}

	// Check if user already exists
	var existingUser User
	result := db.Where("email = ?", req.Email).First(&existingUser)

	if result.Error == nil {
		// User exists, check if they're already subscribed
		if existingUser.Subscribed {
			return c.JSON(http.StatusOK, APIResponse{
				Success: true,
				Message: "You're already subscribed to The Paper!",
			})
		}

		// User exists but was unsubscribed, resubscribe them
		existingUser.Subscribed = true
		if err := db.Save(&existingUser).Error; err != nil {
			log.Printf("Error resubscribing user: %v", err)
			return c.JSON(http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "Failed to resubscribe",
			})
		}

		log.Printf("User %s resubscribed successfully", existingUser.Email)
		return c.JSON(http.StatusOK, APIResponse{
			Success: true,
			Message: "Welcome back! You've been resubscribed to The Paper.",
		})
	}

	// Create new user
	token, err := generateUnsubscribeToken()
	if err != nil {
		log.Printf("Error generating token: %v", err)
		return c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to create subscription",
		})
	}

	newUser := User{
		Email:            req.Email,
		Subscribed:       true,
		UnsubscribeToken: token,
	}

	if err := db.Create(&newUser).Error; err != nil {
		log.Printf("Error creating user: %v", err)
		return c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to subscribe",
		})
	}

	log.Printf("New user subscribed: %s", newUser.Email)
	return c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: "Successfully subscribed! You'll receive The Paper daily digest in your inbox.",
	})
}

func handleUnsubscribeAPI(c echo.Context) error {
	var req UnsubscribeRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.Token == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Unsubscribe token is required",
		})
	}

	// Find user by token
	var user User
	result := db.Where("unsubscribe_token = ?", req.Token).First(&user)
	if result.Error != nil {
		return c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Invalid unsubscribe token",
		})
	}

	// Update subscription status
	user.Subscribed = false
	if err := db.Save(&user).Error; err != nil {
		log.Printf("Error unsubscribing user: %v", err)
		return c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to unsubscribe",
		})
	}

	log.Printf("User %s unsubscribed successfully", user.Email)
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "You have been successfully unsubscribed",
	})
}

func generateUnsubscribeToken() (string, error) {
	// Generate a random token using crypto/rand
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

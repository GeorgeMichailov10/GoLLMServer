package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// User Model(s)

type User struct {
	ID       primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Username string
	Password string
	Chats    map[primitive.ObjectID]string // Id : title for now where title will simply be the chat id as a string for now
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CRUD functions

func CreateUser(c echo.Context) error {
	var user User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

	user.ID = primitive.NewObjectID()

	passwordHash, passwordErr := HashPassword(user.Password)
	if passwordErr != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create user"})
	}

	user.Password = passwordHash

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := UserCollection.InsertOne(ctx, user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create user"})
	}
	return c.JSON(http.StatusCreated, user)
}

func GetUser(username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := UserCollection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNilDocument {
			return nil, nil
		}
		log.Println("Error fetching user:", err)
		return nil, err
	}

	return &user, nil
}

func DeleteUser(c echo.Context) error {
	username := c.Param("username")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := UserCollection.DeleteOne(ctx, bson.M{"username": username})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete user"})
	}

	if res.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "User deleted successfully"})
}

// Repository Functions

func Login(c echo.Context) error {
	var loginReq LoginRequest

	if err := c.Bind(&loginReq); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}

	user, err := GetUser(loginReq.Username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error fetching user"})
	}
	if user == nil || !CheckPassword(loginReq.Password, user.Password) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid username or password"})
	}

	token, err := GenerateJWT(user.Username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to generate token"})
	}

	return c.JSON(http.StatusOK, echo.Map{"token": token})
}

func Register(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}

	existingUser, err := GetUser(req.Username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}
	if existingUser != nil {
		return c.JSON(http.StatusConflict, echo.Map{"error": "Username already exists"})
	}

	newUser := User{
		ID:       primitive.NewObjectID(),
		Username: req.Username,
		Chats:    []primitive.ObjectID{},
	}
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to hash password"})
	}
	newUser.Password = hashedPassword

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = UserCollection.InsertOne(ctx, newUser)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to register user"})
	}

	return c.JSON(http.StatusCreated, echo.Map{"message": "User registered successfully", "user": newUser})
}

func GetUserHandler(c echo.Context) error {
	username := c.Param("username")

	user, err := GetUser(username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error fetching user"})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	return c.JSON(http.StatusOK, user)
}

func GetUserChats(c echo.Context) error {
	username := c.Param("username")

	user, err := GetUser(username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error fetching user"})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{"chats": user.Chats})
}

// Route Controller

func UserRouteController(e *echo.Echo) {
	e.POST("/register", Register)
	e.POST("/login", Login)

	protected := e.Group("/user")
	protected.Use(JWTMiddleware)
	protected.GET("/:username", GetUserHandler)
	protected.GET("/:username/chats", GetUserChats)
	protected.DELETE("/:username", DeleteUser)
}

// Utility Functions
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

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
	ID       primitive.ObjectID            `json:"id,omitempty" bson:"_id,omitempty"`
	Username string                        `json:"username" bson:"username"`
	Password string                        `json:"password" bson:"password"`
	Chats    map[primitive.ObjectID]string `json:"chats,omitempty" bson:"chats,omitempty"` // Id : title for now where title will simply be the chat id as a string for now
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CRUD functions

func CreateUser(c echo.Context, userDetails LoginRequest) error {
	passwordHash, passwordErr := HashPassword(userDetails.Password)
	if passwordErr != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create user"})
	}

	user := User{
		ID:       primitive.NewObjectID(),
		Username: userDetails.Username,
		Password: passwordHash,
		Chats:    make(map[primitive.ObjectID]string),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := UserCollection.InsertOne(ctx, user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create user"})
	}
	return c.JSON(http.StatusCreated, echo.Map{"message": "SUCCESS CREATING USER"})
}

func GetUser(username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := UserCollection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		log.Println("Error fetching user:", err)
		return nil, err
	}

	return &user, nil
}

func DeleteUser(c echo.Context) error {
	username := c.Param("username")

	user, err := GetUser(username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error fetching user"})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	chatIDs := make([]primitive.ObjectID, 0, len(user.Chats))
	for chatID := range user.Chats {
		chatIDs = append(chatIDs, chatID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if len(chatIDs) > 0 {
		_, err := ChatCollection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": chatIDs}})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete chats"})
		}
	}

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
	var loginRequest LoginRequest
	if err := c.Bind(&loginRequest); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request payload"})
	}

	existingUser, err := GetUser(loginRequest.Username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}
	if existingUser != nil {
		return c.JSON(http.StatusConflict, echo.Map{"error": "Username already exists"})
	}

	return CreateUser(c, loginRequest)
}

func GetUserHandler(c echo.Context) error {
	username, ok := c.Get("username").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Unauthorized access attempt."})
	}

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
	username, ok := c.Get("username").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Unauthorized"})
	}

	user, err := GetUser(username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error fetching user"})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{"chats": user.Chats})
}

func CreateNewChat(c echo.Context) error {
	username, ok := c.Get("username").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Unauthorized"})
	}

	newChat := Chat{
		ID:            primitive.NewObjectID(),
		OwnerUsername: username,
		Content:       make([]map[string]string, 0),
	}
	newChat.Title = newChat.ID.Hex()

	inserted := CreateChat(newChat)
	if !inserted {
		log.Printf("Failed to create new chat for %s", username)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create chat"})
	}

	log.Printf("Successfully created new chat for %s", username)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	update := bson.M{
		"$set": bson.M{
			newChat.ID.Hex(): newChat.Title,
		},
	}
	res, err := UserCollection.UpdateOne(ctx, bson.M{"username": username}, update)
	if err != nil {
		log.Printf("Failed to update user's chats for %s: %v", username, err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update user chats"})
	}
	if res.ModifiedCount == 0 {
		log.Printf("No document was updated when updating chats for user %s", username)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update user chats"})
	}

	return c.NoContent(http.StatusOK)
}

// Route Controller

func UserRouteController(e *echo.Echo) {
	e.POST("/register", Register)
	e.POST("/login", Login)

	protected := e.Group("/user")
	protected.Use(JWTMiddleware)
	protected.GET("", GetUserHandler)
	protected.GET("/chats", GetUserChats)
	protected.POST("/chats", CreateNewChat)
	protected.DELETE("", DeleteUser)
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

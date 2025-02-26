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
)

// Chat Model(s)

type Chat struct {
	ID      primitive.ObjectID  `json:"id,omitempty" bson:"_id,omitempty"`
	OwnerID primitive.ObjectID  `json:"ownerid,omitempty" bson:"ownerid,omitempty"`
	Title   string              `json:"title,omitempty" bson:"title,omitempty"`
	Content []map[string]string `json:"content,omitempty" bson:"content,omitempty"` // list of {user:"", model:""} interactions
}

type ChatInteraction struct {
	ChatID    string
	UserChat  string
	ModelChat string
}

// CRUD functions

func CreateChat(c echo.Context) error {
	username, ok := c.Get("username").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Unauthorized"})
	}

	owningUser, err := GetUser(username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create chat"})
	}

	chat := Chat{
		ID:      primitive.NewObjectID(),
		OwnerID: owningUser.ID,
		Content: make([]map[string]string, 3),
	}
	chat.Title = chat.ID.String()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, insert_err := ChatCollection.InsertOne(ctx, chat)
	if insert_err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create chat"})
	}
	return c.JSON(http.StatusCreated, chat)
}

func GetChat(c echo.Context) (*Chat, error) {
	chatIDParam := c.Param("chatid")
	chatID, err := primitive.ObjectIDFromHex(chatIDParam)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var chat Chat
	err = ChatCollection.FindOne(ctx, bson.M{"_id": chatID}).Decode(&chat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		log.Println("Error fetching chat:", err)
		return nil, err
	}

	return &chat, nil
}

func DeleteChat(c echo.Context) error {
	chatIDParam := c.Param("chatid")
	chatID, err := primitive.ObjectIDFromHex(chatIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid chat ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := ChatCollection.DeleteOne(ctx, bson.M{"_id": chatID})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete chat"})
	}

	if res.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Chat not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Chat deleted successfully"})
}

// Repository Functions

func GetChatHandler(c echo.Context) error {
	chat, err := GetChat(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error retrieving chat"})
	}

	if chat == nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Chat not found"})
	}

	return c.JSON(http.StatusOK, chat)
}

// Route Controller

func ChatRouteController(e *echo.Echo) {
	chatGroup := e.Group("/c")

	chatGroup.Use(JWTMiddleware)
	chatGroup.POST("", CreateChat)
	chatGroup.GET("/:chatid", GetChatHandler)
	chatGroup.DELETE("/:chatid", DeleteChat)
}

// Utility Functions

// Verify if this is correct
func AddInteraction(interaction ChatInteraction) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$push": bson.M{
			"content": bson.M{
				"user":  interaction.UserChat,
				"model": interaction.ModelChat,
			},
		},
	}

	res, err := ChatCollection.UpdateOne(ctx, bson.M{"_id": interaction.ChatID}, update)
	if err != nil {
		return false
	}

	if res.ModifiedCount == 0 {
		return false
	}

	return true
}

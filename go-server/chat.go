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
	UserChat  string `json:"userchat" bson:"userchat"`
	ModelChat string `json:"modelchat" bson:"modelchat"`
}

// CRUD functions

func CreateChat(c echo.Context) error {
	var chat Chat
	if err := c.Bind(&chat); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

	chat.ID = primitive.NewObjectID()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := ChatCollection.InsertOne(ctx, chat)
	if err != nil {
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

func AddInteraction(c echo.Context) error {
	chatIDParam := c.Param("chatid")
	chatID, err := primitive.ObjectIDFromHex(chatIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid chat ID"})
	}

	var interaction ChatInteraction
	if err := c.Bind(&interaction); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

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

	res, err := ChatCollection.UpdateOne(ctx, bson.M{"_id": chatID}, update)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to add interaction"})
	}

	if res.ModifiedCount == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Chat not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Interaction added successfully"})
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
	chatGroup.POST("/:chatid", AddInteraction)
	chatGroup.DELETE("/:chatid", DeleteChat)
}

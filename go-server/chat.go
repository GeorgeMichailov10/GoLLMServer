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
	ID      primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	OwnerID primitive.ObjectID
	Title   string
	Content []map[string]string // list of {user:"", model:""} interactions
}

type ChatInteraction struct {
	UserID    primitive.ObjectID
	UserChat  string `json:"userchat"`
	ModelChat string `json:"modelchat"`
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

func GetChat(chatID string) (*Chat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var chat Chat
	err := ChatCollection.FindOne(ctx, bson.M{"chatid": chatID}).Decode(&chat)
	if err != nil {
		if err == mongo.ErrNilDocument {
			return nil, nil
		}
		log.Println("Error fetching user:", err)
		return nil, err
	}

	return &chat, nil
}

func AddInteraction(c echo.Context) error {
	var chatInteraction ChatInteraction
	if err := c.Bind(&chatInteraction); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

}

func DeleteChat() {}

// Repository Functions

// Route Controller

func ChatRouteController(e *echo.Echo) {

}

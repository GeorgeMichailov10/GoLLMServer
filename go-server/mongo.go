package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoURI           = "mongodb://localhost:27017"
	databaseName       = "playground1"
	userCollectionName = "users"
	chatCollectionName = "chats"
)

var UserCollection *mongo.Collection
var ChatCollection *mongo.Collection

func connectToMongoDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))

	if err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}

	UserCollection = client.Database(databaseName).Collection(userCollectionName)
	ChatCollection = client.Database(databaseName).Collection(chatCollectionName)

}

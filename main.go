package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"time"
)

var client *mongo.Client

func authenticateUser(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	// Parse request
	var user struct {
		Username string
		Password string
	}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check in MongoDB for user credentials
	collection := client.Database("todo").Collection("user")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	var result struct {
		Username string
		Password string
	}
	err = collection.FindOne(ctx, bson.M{"username": user.Username, "password": user.Password}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check for existing API key
	apiKeyCollection := client.Database("todo").Collection("api_keys")
	var apiKeyRecord struct {
		Username string `bson:"username"`
		ApiKey   string `bson:"api_key"`
	}

	err = apiKeyCollection.FindOne(ctx, bson.M{"username": user.Username}).Decode(&apiKeyRecord)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// User does not have an API key, so generate one
			newApiKey := uuid.New().String()
			_, err = apiKeyCollection.InsertOne(ctx, bson.M{"username": user.Username, "api_key": newApiKey})
			if err != nil {
				log.Printf("Error storing new API key: %s\n", err)
				http.Error(w, "Error generating API key", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "User authenticated. New API key generated: %s", newApiKey)
			return
		} else {
			// Error other than no documents found
			log.Printf("Error querying API key collection: %s\n", err)
			http.Error(w, "Error querying API key collection", http.StatusInternalServerError)
			return
		}
	} else {
		// Log the entire apiKeyRecord for debugging
		log.Printf("Retrieved API key record: %+v\n", apiKeyRecord)

		// Check if apiKeyRecord.ApiKey is empty
		if apiKeyRecord.ApiKey == "" {
			log.Println("API key retrieved but is empty")
			http.Error(w, "Retrieved API key is empty", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "User authenticated. Existing API key: %s", apiKeyRecord.ApiKey)
	}
}
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func main() {
	var err error
	client, err = mongo.NewClient(options.Client().ApplyURI("mongodb://todoadm:todoadm@54.208.42.87/todo"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Disconnect(ctx)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you've reached the todo app!")
	})

	// Move this line outside of the above handler
	http.HandleFunc("/login", authenticateUser)

	fmt.Println("Server is starting on port 8086...")
	log.Fatal(http.ListenAndServe(":8086", nil))
}

package db

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

// InitFirestore initializes the app and returns the Firestore client
func InitFirestore() *firestore.Client {
	ctx := context.Background()

	// IMPORTANT: We assume serviceAccountKey.json is in the project ROOT.
	sa := option.WithCredentialsFile("serviceAccountKey.json")

	// Update this with your actual Project ID from the previous step
	conf := &firebase.Config{ProjectID: "wcs-organizer"}

	app, err := firebase.NewApp(ctx, conf, sa)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("error initializing firestore client: %v\n", err)
	}

	fmt.Println("Connected to Firestore")
	return client
}
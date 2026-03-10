package db

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/storage"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// AppClients holds all our GCP/Firebase connections
type AppClients struct {
	Firestore *firestore.Client
	Storage   *storage.Client
	Calendar  *calendar.Service
}

func InitClients() *AppClients {
	ctx := context.Background()
	opt := option.WithCredentialsFile("wcs-booking-creds.json")

	// 1. Init Firebase App (for Firestore & Storage)
	config := &firebase.Config{
		StorageBucket: "wcs-booking-backend.firebasestorage.app",
	}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase app: %v\n", err)
	}

	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Error initializing Firestore: %v\n", err)
	}

	storageClient, err := app.Storage(ctx)
	if err != nil {
		log.Fatalf("Error initializing Storage: %v\n", err)
	}

	// 2. Init Google Calendar Service
	calendarService, err := calendar.NewService(ctx, opt)
	if err != nil {
		log.Fatalf("Error initializing Calendar service: %v\n", err)
	}

	log.Println("Successfully connected to Firestore, Storage, and Calendar APIs")

	return &AppClients{
		Firestore: firestoreClient,
		Storage:   storageClient,
		Calendar:  calendarService,
	}
}
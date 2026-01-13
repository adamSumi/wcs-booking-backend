package main

import (
	"log"
	"net/http"

	"github.com/adamSumi/wcs-admin/internal/db"
	"github.com/adamSumi/wcs-admin/internal/handlers"
)

func main() {
	// 1. Initialize DB
	client := db.InitFirestore()
	defer client.Close()

	// 2. Initialize Handlers
	userHandler := &handlers.UserHandler{Client: client}

	// 3. Define Routes
	// This tells the server: "When someone POSTs to /users, run CreateUser"
	http.HandleFunc("POST /users", userHandler.CreateUser)

	// 4. Start Server
	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
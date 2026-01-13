package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/adamSumi/wcs-admin/internal/models" // UPDATE THIS IMPORT to match your go.mod
)

type UserHandler struct {
	Client *firestore.Client
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// 1. Parse the JSON body
	var newUser models.User
	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 2. Set default values
	newUser.CreatedAt = time.Now()

	ctx := context.Background()

	// 3. Add to Firestore "users" collection
	// We use .Add() to let Firestore auto-generate the ID
	ref, _, err := h.Client.Collection("users").Add(ctx, newUser)
	if err != nil {
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	// 4. Send back the ID
	newUser.ID = ref.ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newUser)
}
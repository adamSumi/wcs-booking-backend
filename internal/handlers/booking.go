package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
	"encoding/json"

	"cloud.google.com/go/firestore"
	gcpstorage "cloud.google.com/go/storage" // Add the core GCP package with an alias
	"firebase.google.com/go/v4/storage"      // Keep the Firebase wrapper
	"github.com/adamSumi/wcs-admin/internal/models"
	"github.com/google/uuid"
	"google.golang.org/api/calendar/v3"
)

type BookingHandler struct {
	Firestore *firestore.Client
	Storage   *storage.Client
	Calendar  *calendar.Service
}

const calendarID = "32744b9f5124af54aa4436631f090b07d15d2f7d0960a59d3885431f0d4c0bec@group.calendar.google.com"

func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	// 1. Limit the upload size (e.g., 50MB max memory parsing)
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	videoURL := ""

	// 2. Extract the file (if one was uploaded)
	file, header, err := r.FormFile("video")
	if err == nil {
		defer file.Close()

		// Get a reference to your default storage bucket
		bucket, err := h.Storage.DefaultBucket()
		if err != nil {
			http.Error(w, "Failed to get storage bucket", http.StatusInternalServerError)
			return
		}

		// Create a unique filename to prevent overwriting
		filename := fmt.Sprintf("booking-videos/%s-%s", uuid.New().String(), header.Filename)
		obj := bucket.Object(filename)
		writer := obj.NewWriter(ctx)

		// Stream the file to Firebase Storage
		if _, err := io.Copy(writer, file); err != nil {
			http.Error(w, "Failed to upload video", http.StatusInternalServerError)
			return
		}
		writer.Close()

		// Make the file publicly accessible (optional, depends on your preference)
		if err := obj.ACL().Set(ctx, gcpstorage.AllUsers, gcpstorage.RoleReader); err != nil {
			fmt.Printf("Warning: Could not set ACL for public read: %v\n", err)
		}

		// Construct the public URL
		videoURL = fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket.BucketName(), filename)
	}

	// 3. Extract the text fields and build the model
	booking := models.Booking{
		Name:          r.FormValue("name"),
		Email:         r.FormValue("email"),
		Timeslot:      r.FormValue("timeslot"),
		Role:          r.FormValue("role"),
		Location:      r.FormValue("location"),
		ContactVia:    r.FormValue("contact-via"),
		ContactHandle: r.FormValue("handle"),
		Notes:         r.FormValue("notes"),
		VideoURL:      videoURL,
		CreatedAt:     time.Now(),
	}

	// If they selected "Other" for location, overwrite with the custom input
	if booking.Location == "other" {
		booking.Location = r.FormValue("other-location")
	}

	// 4. Save to Firestore
	_, _, err = h.Firestore.Collection("bookings").Add(ctx, booking)
	if err != nil {
		http.Error(w, "Failed to save booking to database", http.StatusInternalServerError)
		return
	}

	// 5. Send success response
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Booking successfully created!"))
}

func (h *BookingHandler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	// Force timezone to match your local San Diego time for accurate day boundaries
	loc, _ := time.LoadLocation("America/Los_Angeles")

	// Set the window: from right now until exactly 1 month from now
	now := time.Now().In(loc)
	timeMin := now.Format(time.RFC3339)
	timeMax := now.AddDate(0, 1, 0).Format(time.RFC3339)

	// Fetch events from Google Calendar
	events, err := h.Calendar.Events.List(calendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(timeMin).
		TimeMax(timeMax).
		OrderBy("startTime").
		Do()

	if err != nil {
		http.Error(w, "Failed to fetch calendar events", http.StatusInternalServerError)
		return
	}

	// This map will group our chunks by date. Format: {"2026-03-10": ["06:00 PM", "06:30 PM"]}
	availability := make(map[string][]string)

	for _, item := range events.Items {
		// IMPORTANT: This filters for events specifically named "Available".
		// You can change this to "Teaching Block" or whatever you prefer.
		if item.Summary != "Private Lesson Availability" {
			continue
		}

		// Skip all-day events (they don't have a specific DateTime)
		if item.Start.DateTime == "" {
			continue
		}

		// Parse the start and end times from Google's RFC3339 format
		startTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
		endTime, _ := time.Parse(time.RFC3339, item.End.DateTime)

		// Ensure the parsed times are evaluated in your local timezone
		startTime = startTime.In(loc)
		endTime = endTime.In(loc)

		// The Chunking Engine: Loop through the block in 30-minute increments
		for t := startTime; t.Before(endTime); t = t.Add(30 * time.Minute) {
			dateKey := t.Format("2006-01-02") // Output: YYYY-MM-DD
			timeValue := t.Format("03:04 PM") // Output: HH:MM AM/PM

			// Append the 30-minute chunk to that specific date's array
			availability[dateKey] = append(availability[dateKey], timeValue)
		}
	}

	// Return the grouped slots as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(availability)
}
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
	locationVal := r.FormValue("location")
	if locationVal == "loc1" {
		locationVal = "Infinity Dancesport Studio"
	} else if locationVal == "loc2" {
		locationVal = "Atomic Ballroom"
	} else if locationVal == "other" {
		locationVal = r.FormValue("other-location")
	}

	booking := models.Booking{
		Name:          r.FormValue("name"),
		Email:         r.FormValue("email"),
		Timeslot:      r.FormValue("timeslot"),
		Role:          r.FormValue("role"),
		Location:      locationVal,
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

	// --- GOOGLE CALENDAR INJECTION ---

	// 1. Parse the custom timeslot string from the frontend
	// Example format coming from Svelte: "2026-03-10 06:00 PM"
	timeslotStr := r.FormValue("timeslot")
	loc, _ := time.LoadLocation("America/Los_Angeles")

	startTime, err := time.ParseInLocation("2006-01-02 03:04 PM", timeslotStr, loc)
	if err != nil {
		fmt.Printf("Warning: Failed to parse timeslot for calendar: %v\n", err)
		// We log the error but don't fail the whole request,
		// since the Firestore document was still saved.
	} else {
		// 2. Define the lesson length (assuming 1 hour here)
		endTime := startTime.Add(1 * time.Hour)

		// 3. Construct the Calendar Event
		// We can stuff the optional dropdown data into the description
		description := fmt.Sprintf("Email: %s\nRole: %s\nLocation: %s\nContact Via: %s\nHandle: %s\n\nNotes:\n%s",
			booking.Email,
			booking.Role,
			booking.Location,
			booking.ContactVia,
			booking.ContactHandle,
			booking.Notes,
		)

		newCalEvent := &calendar.Event{
			Summary:     "WCS Lesson: " + r.FormValue("name"),
			Description: description,
			Start: &calendar.EventDateTime{
				DateTime: startTime.Format(time.RFC3339),
				TimeZone: "America/Los_Angeles",
			},
			End: &calendar.EventDateTime{
				DateTime: endTime.Format(time.RFC3339),
				TimeZone: "America/Los_Angeles",
			},
			// Optional: Color code the event (e.g., "1" is Lavender, "9" is Blueberry)
			ColorId: "9",
		}

		// 4. Insert the event into the calendar
		// Uses the same global calendarID constant we defined earlier
		_, err = h.Calendar.Events.Insert(calendarID, newCalEvent).Do()
		if err != nil {
			fmt.Printf("Warning: Failed to create calendar event: %v\n", err)
		} else {
			fmt.Println("Successfully added lesson to Google Calendar!")
		}
	}
	// ---------------------------------

	// 5. Send success response
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Booking successfully created!"))
}

func (h *BookingHandler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Now().In(loc)

	// Fetch from midnight today, instead of this exact second
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	timeMin := startOfDay.Format(time.RFC3339)
	timeMax := now.AddDate(0, 1, 0).Format(time.RFC3339)

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

	type timeBlock struct {
		start time.Time
		end   time.Time
	}
	var availableBlocks []timeBlock
	var busyBlocks []timeBlock

	for _, item := range events.Items {
		if item.Start.DateTime == "" {
			continue // Skip all-day events
		}

		startTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
		endTime, _ := time.Parse(time.RFC3339, item.End.DateTime)

		block := timeBlock{
			start: startTime.In(loc),
			end:   endTime.In(loc),
			name:  item.Summary,
		}

		if item.Summary == "Private Lesson Availability" {
			availableBlocks = append(availableBlocks, block)
		} else {
			busyBlocks = append(busyBlocks, block)
		}
	}

	availability := make(map[string][]string)

	for _, avail := range availableBlocks {
		for t := avail.start; t.Before(avail.end); t = t.Add(30 * time.Minute) {

			slotStart := t
			slotEnd := t.Add(1 * time.Hour)

			if slotEnd.After(avail.end) {
				continue
			}

			if slotStart.Before(now) {
				continue
			}

			isBusy := false
			for _, busy := range busyBlocks {
				if slotStart.Before(busy.end) && slotEnd.After(busy.start) {
					isBusy = true
					break
				}
			}

			if !isBusy {
				dateKey := slotStart.Format("2006-01-02")
				timeValue := slotStart.Format("03:04 PM")
				availability[dateKey] = append(availability[dateKey], timeValue)
			}
		}
	}

	// KILL THE BROWSER CACHE
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(availability)
}
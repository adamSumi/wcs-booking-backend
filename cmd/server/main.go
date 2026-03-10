package main

import (
	"log"
	"net/http"

	"github.com/adamSumi/wcs-admin/internal/db"
	"github.com/adamSumi/wcs-admin/internal/handlers"
	"github.com/adamSumi/wcs-admin/internal/middleware"
)

func main() {
	// Initialize everything
	appClients := db.InitClients()
	defer appClients.Firestore.Close()

	// Pass all three clients to the handler
	bookingHandler := &handlers.BookingHandler{
		Firestore: appClients.Firestore,
		Storage:   appClients.Storage,
		Calendar:  appClients.Calendar,
	}

	mux := http.NewServeMux()

	// Add your routes
	mux.HandleFunc("GET /api/availability", bookingHandler.GetAvailability)
	mux.HandleFunc("POST /api/bookings", bookingHandler.CreateBooking)

	handler := middleware.EnableCORS(mux)

	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
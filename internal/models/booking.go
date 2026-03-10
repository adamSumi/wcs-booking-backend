package models

import "time"

// Booking represents a private lesson request
type Booking struct {
	ID            string    `firestore:"id,omitempty" json:"id,omitempty"`
	Name          string    `firestore:"name" json:"name"`
	Email         string    `firestore:"email" json:"email"`
	Timeslot      string    `firestore:"timeslot" json:"timeslot"`
	Role          string    `firestore:"role" json:"role"`
	Location      string    `firestore:"location" json:"location"`
	ContactVia    string    `firestore:"contactVia" json:"contactVia"`
	ContactHandle string    `firestore:"contactHandle" json:"contactHandle"`
	Notes         string    `firestore:"notes" json:"notes"`
	VideoURL      string    `firestore:"videoUrl" json:"videoUrl"` // URL to the file in Cloud Storage
	CreatedAt     time.Time `firestore:"createdAt" json:"createdAt"`
}
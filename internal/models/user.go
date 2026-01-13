package models

import "time"

// Enums for strict typing
type UserType string

const (
	TypeStudent    UserType = "Student"
	TypeInstructor UserType = "Instructor"
)

type User struct {
	ID        string    `firestore:"-" json:"id"`                   // ID is stored in the document key, not the body
	Name      string    `firestore:"name" json:"name"`
	Email     string    `firestore:"email" json:"email"`
	Role      UserType  `firestore:"role" json:"role"`              // Student or Instructor
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
}
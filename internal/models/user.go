package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name" validate:"required"`
	Username  string    `json:"username" validate:"required"`
	Email     string    `json:"email" validate:"required"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"update_at"`
}

// for registration
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required"`
}

// for login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// for login response
type LoginResponse struct {
	Success        bool   `json:"success"`
	Token          string `json:"token,omitempty"`
	ExpiresIn      int    `json:"expires_in,omitempty"`
	RemainingTries int    `json:"remaining_tries,omitempty"`
	RetryAfter     int    `json:"retry_after,omitempty"`
	Message        string `json:"message,omitempty"`
}

// JWT claims structure

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

/*

Registered claims
The JWT specification defines seven reserved claims that are not required, but are recommended to allow interoperability with third-party applications. These are:

iss (issuer): Issuer of the JWT

sub (subject): Subject of the JWT (the user)

aud (audience): Recipient for which the JWT is intended

exp (expiration time): Time after which the JWT expires

nbf (not before time): Time before which the JWT must not be accepted for processing

iat (issued at time): Time at which the JWT was issued; can be used to determine age of the JWT

jti (JWT ID): Unique identifier; can be used to prevent the JWT from being replayed (allows a token to be used only once)

*/

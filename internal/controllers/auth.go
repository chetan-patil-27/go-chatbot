package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	DB *pgx.Conn
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// validate the request payload
func validateRegisterRequest(req RegisterRequest) string {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)

	if username == "" {
		return "username is required"
	}

	if len(username) < 3 || len(username) > 30 {
		return "username must be between 3 and 30 characters"
	}

	if email == "" {
		return "email is required"
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return "invalid email address"
	}

	if req.Password == "" {
		return "password is required"
	}

	if len(req.Password) < 8 {
		return "password must be at least 6 characters"
	}

	return ""
}

func (ac *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	validationError := validateRegisterRequest(req)

	if validationError != "" {
		http.Error(w, validationError, http.StatusBadRequest)
		return
	}

	// hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// insert the user into the database
	_, err = ac.DB.Exec(
		r.Context(),
		`INSERT INTO users(username,email,password_hash) VALUES($1,$2,$3)`, req.Username, req.Email, string(hashedPassword),
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "username or email already exists", http.StatusConflict)
			return
		}

		fmt.Println("Database error:", err)
		http.Error(w, "failed to register user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User registered successfully",
	})

}

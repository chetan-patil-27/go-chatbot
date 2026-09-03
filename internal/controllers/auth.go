package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-chatbot/internal/models"
	"go-chatbot/internal/utils"
	"net/http"
	"net/mail"
	"strings"
	"time"

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
		return "password must be at least 8 characters"
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

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (ac *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	var user models.User
	err = ac.DB.QueryRow(
		r.Context(),
		`SELECT id,username,email,password_hash FROM users WHERE email=$1`, req.Email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		fmt.Println("Database err :", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	//password verifications
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	accessToken, err := utils.GenerateToken(user.ID, user.Username)
	if err != nil {
		fmt.Println("JWT generation error: ", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		fmt.Println("Refresh token generation error : ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// saving hashed refresh token in the database
	refreshTokeHash := utils.HashRefreshToken(refreshToken)

	_, err = ac.DB.Exec(
		r.Context(),
		`INSERT INTO refresh_tokens(user_id,token_hash,expires_at) 
		 VALUES($1,$2,NOW() + INTERVAL '7 days')`,
		user.ID,
		refreshTokeHash,
	)

	if err != nil {
		fmt.Println("refresh token database error : ", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":       "Login successful",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})

}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshToken validates a refresh token, rotates it, and issues
// a new access token and refresh token.
func (ac *AuthController) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest

	// Decode the refresh token from the request body.
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Make sure a refresh token was provided.
	if req.RefreshToken == "" {
		http.Error(w, "Refresh token is required", http.StatusBadRequest)
		return
	}

	// Hash the provided token so we can safely compare it
	// with the hash stored in the database.
	tokenHash := utils.HashRefreshToken(req.RefreshToken)

	var userID int64
	var expiresAt time.Time
	var revokedAt *time.Time

	// Start a transaction because token validation, revocation,
	// and creation of the new token must happen atomically.
	tx, err := ac.DB.Begin(r.Context())
	if err != nil {
		fmt.Println("Transaction start error : ", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Roll back automatically if the transaction has not been committed.
	defer tx.Rollback(r.Context())

	var tokenID int64

	// Find and lock the refresh-token row.
	// FOR UPDATE prevents two concurrent refresh requests
	// from using the same refresh token at the same time.
	err = tx.QueryRow(
		r.Context(),
		`SELECT id, user_id, expires_at, revoked_at
		 FROM refresh_tokens
		 WHERE token_hash = $1
		 FOR UPDATE`,
		tokenHash,
	).Scan(&tokenID, &userID, &expiresAt, &revokedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "invalid refresh token", http.StatusUnauthorized)
			return
		}

		fmt.Println("Refresh token database error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Reject the token if it has already been used or logged out.
	if revokedAt != nil {
		http.Error(w, "Refresh token has been revoked", http.StatusUnauthorized)
		return
	}

	// Reject the token if its 7-day lifetime has expired.
	if time.Now().After(expiresAt) {
		http.Error(w, "Refresh token has expired", http.StatusUnauthorized)
		return
	}

	// Revoke the old refresh token.
	// This is the key part of refresh-token rotation.
	_, err = tx.Exec(
		r.Context(),
		`UPDATE refresh_tokens
		 SET revoked_at = NOW()
		 WHERE id = $1`,
		tokenID,
	)

	if err != nil {
		fmt.Println("Refresh token revocation error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Generate a completely new refresh token.
	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		fmt.Println("new refresh token generation error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Store only the hash of the new refresh token in the database.
	newRefreshTokenHash := utils.HashRefreshToken(newRefreshToken)

	// Save the new refresh token with a new 7-day expiration.
	_, err = tx.Exec(
		r.Context(),
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, NOW() + INTERVAL '7 days')`,
		userID,
		newRefreshTokenHash,
	)

	if err != nil {
		fmt.Println("New refresh token database error : ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var username string

	// Get the username required to generate the new access token.
	err = tx.QueryRow(
		r.Context(),
		`SELECT username FROM users WHERE id=$1`,
		userID,
	).Scan(&username)

	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	// Generate a new access token for the user.
	accessToken, err := utils.GenerateToken(userID, username)
	if err != nil {
		fmt.Println("JWT generation error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Commit all database changes together:
	// 1. Old refresh token is revoked.
	// 2. New refresh token is stored.
	err = tx.Commit(r.Context())
	if err != nil {
		fmt.Println("Transaction commit error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Return both the new access token and the new refresh token.
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]string{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (ac *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		http.Error(w, "refresh token is  required", http.StatusBadRequest)
		return
	}

	tokenHash := utils.HashRefreshToken(req.RefreshToken)

	commandTag, err := ac.DB.Exec(
		r.Context(),
		`UPDATE refresh_tokens SET revoked_at=NOW() 
		WHERE token_hash=$1 AND revoked_at IS NULL`,
		tokenHash,
	)

	if err != nil {
		fmt.Println("logoute database error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if commandTag.RowsAffected() == 0 {
		http.Error(w, "refresh token not found or already revoked", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logout Successfully",
	})

}

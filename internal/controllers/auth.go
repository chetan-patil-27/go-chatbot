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

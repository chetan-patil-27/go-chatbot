package controllers

import (
	"encoding/json"
	"go-chatbot/internal/middleware"
	"go-chatbot/internal/models"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type ChatController struct {
	DB *pgx.Conn
}

type CreateChatRequest struct {
	Title string `json:"title"`
}

func (c *ChatController) CreateChat(w http.ResponseWriter, r *http.Request) {
	// get user id from jwt middleware
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)

	if !ok {
		http.Error(w, "user id not found", http.StatusUnauthorized)
		return
	}

	// read request body
	var req CreateChatRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	//insert chat into database

	var chat models.Chat

	query := "INSERT INTO chats(user_id,title) VALUES ($1,$2) RETURNING id, user_id, title, created_at"

	err = c.DB.QueryRow(
		r.Context(),
		query,
		userID,
		req.Title,
	).Scan(
		&chat.ID,
		&chat.UserID,
		&chat.Title,
		&chat.CreatedAt,
	)

	if err != nil {
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	// Response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(chat)
}

package controllers

import (
	"encoding/json"
	"go-chatbot/internal/middleware"
	"go-chatbot/internal/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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

func (c *ChatController) GetChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "user id not found", http.StatusUnauthorized)
		return
	}

	query := `
			SELECT id, user_id, title, created_at FROM chats 
			WHERE user_id = $1 ORDER BY created_at DESC
			`
	rows, err := c.DB.Query(r.Context(), query, userID)
	if err != nil {
		http.Error(w, "faild to fatch chats", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var chats []models.Chat

	for rows.Next() {
		var chat models.Chat

		err := rows.Scan(
			&chat.ID,
			&chat.UserID,
			&chat.Title,
			&chat.CreatedAt,
		)

		if err != nil {
			http.Error(w, "faild to read chats", http.StatusInternalServerError)
			return
		}

		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "faild to read chats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}

func (c *ChatController) DeleteChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "user id not found", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)

	chatID, err := strconv.ParseInt(vars["chat_id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	result, err := c.DB.Exec(
		r.Context(),
		`DELETE FROM chats WHERE id = $1 and user_id = $2`,
		chatID,
		userID,
	)

	if err != nil {
		http.Error(w, "failed to delete chat", http.StatusInternalServerError)
		return

	}

	if result.RowsAffected() == 0 {
		http.Error(w, "chat not found or access denied", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

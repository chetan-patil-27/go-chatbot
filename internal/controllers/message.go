package controllers

import (
	"encoding/json"
	"fmt"
	"go-chatbot/internal/bot"
	"go-chatbot/internal/middleware"
	"go-chatbot/internal/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

type MessageController struct {
	DB *pgx.Conn
}

type SendMessageRequest struct {
	Message string `json:"message"`
}

type SendMessageResponse struct {
	UserMessage models.Message `json:"user_message"`
	BotMessage  models.Message `json:"bot_message"`
}

func (c *MessageController) SendMessage(w http.ResponseWriter, r *http.Request) {

	// Get authenticated user ID from JWT
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)

	if !ok {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Get chat ID from URL
	vars := mux.Vars(r)

	chatID, err := strconv.ParseInt(vars["chat_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	fmt.Println("Authenticated User ID:", userID)
	fmt.Println("Chat ID:", chatID)

	// Read request body
	var req SendMessageRequest

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	//start transaction here
	tx, err := c.DB.Begin(r.Context())
	if err != nil {
		http.Error(w, "failed to start transaction", http.StatusInternalServerError)
		return
	}

	// Rollback automatically if we don't reach Commit()
	defer tx.Rollback(r.Context())

	// Insert message only if the chat belongs to the authenticated user
	var userMessage models.Message

	query := `
		INSERT INTO messages (chat_id, sender, message)
		SELECT $1, 'user', $3
		FROM chats
		WHERE id = $1
		AND user_id = $2
		RETURNING id, chat_id, sender, message, created_at
	`

	err = c.DB.QueryRow(
		r.Context(),
		query,
		chatID,
		userID,
		req.Message,
	).Scan(
		&userMessage.ID,
		&userMessage.ChatID,
		&userMessage.Sender,
		&userMessage.Message,
		&userMessage.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Chat not found or access denied", http.StatusForbidden)
			return
		}

		http.Error(w, "Failed to save user message", http.StatusInternalServerError)
		return
	}

	// genarate bot responce here
	botResponse := bot.GetResponse(req.Message)

	//save bot message
	var botMessage models.Message

	err = tx.QueryRow(
		r.Context(),
		`
		INSERT INTO messages (chat_id, sender, message)
		VALUES ($1, 'bot', $2)
		RETURNING id, chat_id, sender, message, created_at
		`,
		chatID,
		botResponse,
	).Scan(
		&botMessage.ID,
		&botMessage.ChatID,
		&botMessage.Sender,
		&botMessage.Message,
		&botMessage.CreatedAt,
	)

	if err != nil {
		http.Error(w, "failed to save bot message", http.StatusInternalServerError)
		return
	}

	//commit transaction
	err = tx.Commit(r.Context())
	if err != nil {
		http.Error(w, "failed to commite messages", http.StatusInternalServerError)
		return
	}

	// return both message
	response := SendMessageResponse{
		UserMessage: userMessage,
		BotMessage:  botMessage,
	}

	// Return saved message
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}

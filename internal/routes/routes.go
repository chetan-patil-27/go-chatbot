package routes

import (
	"go-chatbot/internal/controllers"
	"go-chatbot/internal/middleware"
	"net/http"

	"github.com/gorilla/mux"
)

func SetupRoute(
	authController *controllers.AuthController,
	chatController *controllers.ChatController,
	messageController *controllers.MessageController,

) *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/register", authController.Register).Methods(http.MethodPost)
	router.HandleFunc("/login", authController.Login).Methods(http.MethodPost)
	router.HandleFunc("/refresh", authController.Refresh).Methods(http.MethodPost)

	router.Handle(
		"/api/test",
		middleware.JWTMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("🔐 You accessed a protected route!"))
			}),
		),
	).Methods(http.MethodGet)

	//protected chat route
	router.Handle(
		"/api/chats",
		middleware.JWTMiddleware(
			http.HandlerFunc(chatController.CreateChat),
		),
	).Methods(http.MethodPost)

	// protected message route

	router.Handle(
		"/api/chats/{chat_id}/messages",
		middleware.JWTMiddleware(
			http.HandlerFunc(messageController.SendMessage),
		),
	).Methods(http.MethodPost)

	router.Handle(
		"/api/chats",
		middleware.JWTMiddleware(
			http.HandlerFunc(chatController.GetChats),
		),
	).Methods(http.MethodGet)

	router.Handle(
		"/api/chats/{chat_id}/messages",
		middleware.JWTMiddleware(
			http.HandlerFunc(messageController.GetMessages),
		),
	).Methods(http.MethodGet)

	router.Handle(
		"/api/chats/{chat_id}",
		middleware.JWTMiddleware(
			http.HandlerFunc(chatController.DeleteChat),
		),
	).Methods(http.MethodDelete)

	return router

}

package routes

import (
	"go-chatbot/internal/controllers"
	"net/http"

	"github.com/gorilla/mux"
)

func SetupRoute(authController *controllers.AuthController) *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/register", authController.Register).Methods(http.MethodPost)
	return router
}

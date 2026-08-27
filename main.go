package main

import (
	"context"
	"fmt"
	"go-chatbot/internal/database"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func homehandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "🤖 Welcome to Go ChatBot API")
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("error while loading the .env file")
	}

	// Connect to PostgreSQL
	db, err := database.Connect()
	if err != nil {
		log.Fatal("error while connecting the database: ", err)
	}
	defer db.Close(context.Background())

	fmt.Println("🐘 PostgreSQL connected successfully")

	router := mux.NewRouter()

	router.HandleFunc("/", homehandler).Methods("GET")
	fmt.Println("🤖 ChatBot server started on :8085")

	err = http.ListenAndServe(":8085", router)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

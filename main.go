package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func homehandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "🤖 Welcome to Go ChatBot API")
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/", homehandler).Methods("GET")
	fmt.Println("🤖 ChatBot server started on :8085")

	err := http.ListenAndServe(":8085", router)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

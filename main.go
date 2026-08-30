package main

import (
	"context"
	"fmt"
	"go-chatbot/internal/controllers"
	"go-chatbot/internal/database"
	"go-chatbot/internal/routes"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	authController := &controllers.AuthController{
		DB: db,
	}

	router := routes.SetupRoute(authController)

	server := &http.Server{
		Addr:    ":8085",
		Handler: router,
	}

	//start the server in sepreate goroutine
	go func() {
		fmt.Println("🤖 ChatBot server started on :8085")

		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}

	}()

	// wait for shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	fmt.Println(" \n🛑Shutting down the server...")

	//give active requests some time to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v", err)
		return
	}

	fmt.Println("✅ Server gracefully stopped")

}

package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func Connect() (*pgx.Conn, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	dns := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname)

	conn, err := pgx.Connect(context.Background(), dns)
	if err != nil {
		return nil, fmt.Errorf("error while connecting the database: %v", err)
	}
	return conn, nil
}

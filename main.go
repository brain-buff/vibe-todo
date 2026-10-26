package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"ringbuffer/internal/handlers"
	"ringbuffer/internal/store"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://todo:todo@localhost:5432/todos?sslmode=disable"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	st, err := store.New(ctx, dsn)
	if err != nil {
		log.Fatalf("подключение к БД: %v", err)
	}
	defer st.Close()

	app := fiber.New(fiber.Config{AppName: "todos"})
	app.Use(recover.New())
	app.Use(logger.New())

	handlers.NewTodo(st).Register(app.Group("/api"))

	// Бэкенд раздаёт и статику — фронтенд без фреймворков.
	app.Static("/", "./static")

	log.Printf("сервер слушает :%s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal(err)
	}
}

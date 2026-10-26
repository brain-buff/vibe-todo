package main

import (
	"context"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"ringbuffer/internal/handlers"
	"ringbuffer/internal/middleware"
	"ringbuffer/internal/store"
)

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://todo:todo@localhost:5432/todos?sslmode=disable"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	ttl := 24 * time.Hour
	if v := os.Getenv("JWT_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	st, err := store.New(ctx, dsn)
	if err != nil {
		log.Fatal("подключение к БД", zap.Error(err))
	}
	defer st.Close()

	app := fiber.New(fiber.Config{AppName: "todos"})

	// Глобальные middleware: трассировка, логирование, восстановление от паник.
	app.Use(middleware.TracerMiddleware())
	app.Use(middleware.LoggerMiddleware(log))
	app.Use(middleware.PanicMiddleware(log))

	api := app.Group("/api")

	// Публичные маршруты: регистрация и логин.
	handlers.NewAuth(st, secret, ttl).Register(api)

	// Защищённые маршруты: требуют JWT.
	api.Use(middleware.AuthMiddleware(secret))
	handlers.NewTodo(st).Register(api)

	// Бэкенд раздаёт и статику — фронтенд без фреймворков.
	app.Static("/", "./static")

	log.Info("сервер запущен", zap.String("port", port))
	if err := app.Listen(":" + port); err != nil {
		log.Fatal("listen", zap.Error(err))
	}
}

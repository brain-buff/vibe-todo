package middleware

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ctxKey string

const (
	ctxKeyLogger ctxKey = "logger"
	ctxKeyTrace  ctxKey = "trace_id"
	ctxKeyUserID ctxKey = "user_id"
)

// Logger возвращает zap-логгер из контекста.
func Logger(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKeyLogger).(*zap.Logger); ok {
		return l
	}
	return zap.NewNop()
}

// TraceID возвращает trace_id из контекста.
func TraceID(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKeyTrace).(string); ok {
		return id
	}
	return ""
}

// UserID возвращает ID пользователя из контекста.
func UserID(ctx context.Context) int64 {
	if id, ok := ctx.Value(ctxKeyUserID).(int64); ok {
		return id
	}
	return 0
}

// TracerMiddleware добавляет trace_id в контекст и заголовки ответа.
func TracerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		traceID := c.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}
		c.Set("X-Trace-ID", traceID)

		ctx := context.WithValue(c.UserContext(), ctxKeyTrace, traceID)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// LoggerMiddleware создаёт zap-логгер для запроса и кладёт его в контекст.
func LoggerMiddleware(log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		reqLogger := log.With(
			zap.String("trace_id", TraceID(c.UserContext())),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
		)

		ctx := context.WithValue(c.UserContext(), ctxKeyLogger, reqLogger)
		c.SetUserContext(ctx)

		err := c.Next()

		reqLogger.Info("request",
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("duration", time.Since(start)),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get("User-Agent")),
		)

		return err
	}
}

// PanicMiddleware ловит паники и логирует их.
func PanicMiddleware(log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				Logger(c.UserContext()).Error("panic recovered",
					zap.Any("panic", r),
					zap.String("trace_id", TraceID(c.UserContext())),
					zap.String("method", c.Method()),
					zap.String("path", c.Path()),
				)
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "внутренняя ошибка сервера",
				})
			}
		}()
		return c.Next()
	}
}

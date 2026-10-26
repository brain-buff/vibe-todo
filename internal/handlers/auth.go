package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"ringbuffer/internal/middleware"
	"ringbuffer/internal/models"
	"ringbuffer/internal/store"
)

// Auth — HTTP-обработчики регистрации и логина.
type Auth struct {
	store  *store.Store
	secret string
	ttl    time.Duration
}

func NewAuth(st *store.Store, secret string, ttl time.Duration) *Auth {
	return &Auth{store: st, secret: secret, ttl: ttl}
}

func (h *Auth) Register(api fiber.Router) {
	api.Post("/auth/register", h.RegisterUser)
	api.Post("/auth/login", h.Login)
}

// RegisterUser — регистрация нового пользователя.
func (h *Auth) RegisterUser(c *fiber.Ctx) error {
	log := middleware.Logger(c.UserContext())

	var in models.UserInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "невалидный JSON")
	}

	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email обязателен")
	}
	if len(in.Password) < 6 {
		return fiber.NewError(fiber.StatusBadRequest, "пароль должен быть не короче 6 символов")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("hash password", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "внутренняя ошибка сервера",
		})
	}

	u, err := h.store.CreateUser(c.UserContext(), models.UserInput{
		Email:    in.Email,
		Password: string(hash),
	})
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "email уже занят",
			})
		}
		log.Error("create user", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "внутренняя ошибка сервера",
		})
	}

	token, err := h.generateToken(u.ID)
	if err != nil {
		log.Error("generate token", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "внутренняя ошибка сервера",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(models.AuthResponse{
		Token: token,
		User:  u,
	})
}

// Login — вход по email и паролю.
func (h *Auth) Login(c *fiber.Ctx) error {
	log := middleware.Logger(c.UserContext())

	var in models.UserInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "невалидный JSON")
	}

	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" || in.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email и пароль обязательны")
	}

	u, err := h.store.GetUserByEmail(c.UserContext(), in.Email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "неверный email или пароль",
			})
		}
		log.Error("get user by email", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "внутренняя ошибка сервера",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "неверный email или пароль",
		})
	}

	token, err := h.generateToken(u.ID)
	if err != nil {
		log.Error("generate token", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "внутренняя ошибка сервера",
		})
	}

	return c.JSON(models.AuthResponse{
		Token: token,
		User:  u,
	})
}

func (h *Auth) generateToken(userID int64) (string, error) {
	claims := &models.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.secret))
}

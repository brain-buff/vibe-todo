package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"ringbuffer/internal/models"
	"ringbuffer/internal/store"
)

// Todo — HTTP-обработчики задач.
type Todo struct {
	store *store.Store
}

func NewTodo(st *store.Store) *Todo {
	return &Todo{store: st}
}

func (h *Todo) Register(api fiber.Router) {
	api.Get("/todos", h.List)
	api.Post("/todos", h.Create)
	api.Put("/todos/:id", h.Update)
	api.Patch("/todos/:id/done", h.SetDone)
	api.Delete("/todos/done", h.ClearDone)
	api.Delete("/todos/:id", h.Delete)
	api.Post("/todos/:id/subtasks", h.AddSubtask)
	api.Patch("/todos/:id/subtasks/:sid/done", h.SetSubtaskDone)
	api.Delete("/todos/:id/subtasks/:sid", h.DeleteSubtask)
}

func (h *Todo) List(c *fiber.Ctx) error {
	todos, err := h.store.List(c.Context())
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(todos)
}

func (h *Todo) Create(c *fiber.Ctx) error {
	in, err := parseInput(c)
	if err != nil {
		return err
	}
	t, err := h.store.Create(c.Context(), in)
	if err != nil {
		return fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(t)
}

func (h *Todo) Update(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	in, err := parseInput(c)
	if err != nil {
		return err
	}
	t, err := h.store.Update(c.Context(), id, in)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(t)
}

func (h *Todo) SetDone(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body struct {
		Done *bool `json:"done"`
	}
	if err := c.BodyParser(&body); err != nil || body.Done == nil {
		return fiber.NewError(fiber.StatusBadRequest, "поле done обязательно")
	}
	if err := h.store.SetDone(c.Context(), id, *body.Done); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Todo) ClearDone(c *fiber.Ctx) error {
	n, err := h.store.ClearDone(c.Context())
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"removed": n})
}

func (h *Todo) Delete(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.store.Delete(c.Context(), id); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Todo) AddSubtask(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body models.SubtaskInput
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "невалидный JSON")
	}
	body.Title = strings.TrimSpace(body.Title)
	if body.Title == "" {
		return fiber.NewError(fiber.StatusBadRequest, "текст подзадачи обязателен")
	}
	st, err := h.store.AddSubtask(c.Context(), id, body.Title)
	if err != nil {
		return fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(st)
}

func (h *Todo) SetSubtaskDone(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	sid, err := parseSubID(c)
	if err != nil {
		return err
	}
	var body struct {
		Done *bool `json:"done"`
	}
	if err := c.BodyParser(&body); err != nil || body.Done == nil {
		return fiber.NewError(fiber.StatusBadRequest, "поле done обязательно")
	}
	if err := h.store.SetSubtaskDone(c.Context(), id, sid, *body.Done); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Todo) DeleteSubtask(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	sid, err := parseSubID(c)
	if err != nil {
		return err
	}
	if err := h.store.DeleteSubtask(c.Context(), id, sid); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func parseID(c *fiber.Ctx) (int64, error) {
	id, err := c.ParamsInt("id")
	if err != nil {
		return 0, fiber.NewError(fiber.StatusBadRequest, "неверный id")
	}
	return int64(id), nil
}

func parseSubID(c *fiber.Ctx) (int64, error) {
	id, err := c.ParamsInt("sid")
	if err != nil {
		return 0, fiber.NewError(fiber.StatusBadRequest, "неверный id подзадачи")
	}
	return int64(id), nil
}

func parseInput(c *fiber.Ctx) (models.TodoInput, error) {
	var in models.TodoInput
	if err := c.BodyParser(&in); err != nil {
		return in, fiber.NewError(fiber.StatusBadRequest, "невалидный JSON")
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return in, fiber.NewError(fiber.StatusBadRequest, "название задачи обязательно")
	}
	if in.Priority < models.PriorityLow || in.Priority > models.PriorityHigh {
		in.Priority = models.PriorityMedium
	}
	return in, nil
}

func fail(c *fiber.Ctx, err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "не найдено"})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "внутренняя ошибка сервера"})
}

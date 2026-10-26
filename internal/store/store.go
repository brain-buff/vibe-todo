package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"ringbuffer/internal/models"
)

// ErrNotFound — запись не найдена.
var ErrNotFound = errors.New("не найдено")

const todoColumns = "id, title, description, done, priority, due_at, created_at, updated_at"

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) migrate(ctx context.Context) error {
	// pgx Exec не поддерживает несколько SQL-выражений в одном вызове,
	// поэтому выполняем миграции по одному.
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id         BIGSERIAL PRIMARY KEY,
			email      TEXT NOT NULL UNIQUE,
			password   TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS todos (
			id          BIGSERIAL PRIMARY KEY,
			user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title       TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			done        BOOLEAN NOT NULL DEFAULT FALSE,
			priority    SMALLINT NOT NULL DEFAULT 2 CHECK (priority BETWEEN 1 AND 3),
			due_at      TIMESTAMPTZ,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		// Миграция для существующих БД: добавляем user_id, если колонки нет.
		`ALTER TABLE todos ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id) ON DELETE CASCADE`,
		`CREATE TABLE IF NOT EXISTS subtasks (
			id      BIGSERIAL PRIMARY KEY,
			todo_id BIGINT NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
			title   TEXT NOT NULL,
			done    BOOLEAN NOT NULL DEFAULT FALSE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_subtasks_todo_id ON subtasks(todo_id)`,
		`CREATE INDEX IF NOT EXISTS idx_todos_user_id ON todos(user_id)`,
	}
	for _, stmt := range stmts {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

type scanner interface{ Scan(dest ...any) error }

func scanTodo(r scanner) (models.Todo, error) {
	var t models.Todo
	err := r.Scan(&t.ID, &t.Title, &t.Description, &t.Done, &t.Priority, &t.DueAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	t.Subtasks = []models.Subtask{}
	return t, nil
}

// List возвращает задачи пользователя: невыполненные первыми, затем по приоритету и сроку.
func (s *Store) List(ctx context.Context, userID int64) ([]models.Todo, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+todoColumns+`
		FROM todos
		WHERE user_id = $1
		ORDER BY done ASC, priority DESC, due_at ASC NULLS LAST, created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	todos := []models.Todo{}
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			return nil, fmt.Errorf("scan todo: %w", err)
		}
		todos = append(todos, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	if len(todos) > 0 {
		ids := make([]int64, len(todos))
		byID := make(map[int64]int, len(todos))
		for i := range todos {
			ids[i] = todos[i].ID
			byID[todos[i].ID] = i
		}

		subs, err := s.pool.Query(ctx, `
			SELECT id, todo_id, title, done FROM subtasks
			WHERE todo_id = ANY($1)
			ORDER BY id`, ids)
		if err != nil {
			return nil, fmt.Errorf("list subtasks: %w", err)
		}
		defer subs.Close()

		for subs.Next() {
			var (
				st     models.Subtask
				todoID int64
			)
			if err := subs.Scan(&st.ID, &todoID, &st.Title, &st.Done); err != nil {
				return nil, fmt.Errorf("scan subtask: %w", err)
			}
			todos[byID[todoID]].Subtasks = append(todos[byID[todoID]].Subtasks, st)
		}
		if err := subs.Err(); err != nil {
			return nil, fmt.Errorf("subtask rows: %w", err)
		}
	}

	return todos, nil
}

func (s *Store) Get(ctx context.Context, userID, id int64) (models.Todo, error) {
	t, err := scanTodo(s.pool.QueryRow(ctx, `
		SELECT `+todoColumns+` FROM todos WHERE id = $1 AND user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, fmt.Errorf("get: %w", err)
	}
	if err := s.loadSubtasks(ctx, &t); err != nil {
		return t, err
	}
	return t, nil
}

func (s *Store) Create(ctx context.Context, userID int64, in models.TodoInput) (models.Todo, error) {
	t, err := scanTodo(s.pool.QueryRow(ctx, `
		INSERT INTO todos (user_id, title, description, priority, due_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+todoColumns,
		userID, in.Title, in.Description, in.Priority, in.DueAt))
	if err != nil {
		return t, fmt.Errorf("create: %w", err)
	}
	return t, nil
}

func (s *Store) Update(ctx context.Context, userID, id int64, in models.TodoInput) (models.Todo, error) {
	t, err := scanTodo(s.pool.QueryRow(ctx, `
		UPDATE todos
		SET title = $1, description = $2, priority = $3, due_at = $4, updated_at = NOW()
		WHERE id = $5 AND user_id = $6
		RETURNING `+todoColumns,
		in.Title, in.Description, in.Priority, in.DueAt, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, fmt.Errorf("update: %w", err)
	}
	if err := s.loadSubtasks(ctx, &t); err != nil {
		return t, err
	}
	return t, nil
}

func (s *Store) SetDone(ctx context.Context, userID, id int64, done bool) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE todos SET done = $3, updated_at = NOW() WHERE id = $1 AND user_id = $2`, id, userID, done)
	if err != nil {
		return fmt.Errorf("set done: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, userID, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM todos WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ClearDone(ctx context.Context, userID int64) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM todos WHERE done = TRUE AND user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("clear done: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (s *Store) AddSubtask(ctx context.Context, userID, todoID int64, title string) (models.Subtask, error) {
	var (
		st    models.Subtask
		owner int64
	)
	err := s.pool.QueryRow(ctx, `
		INSERT INTO subtasks (todo_id, title)
		SELECT id, $2 FROM todos WHERE id = $1 AND user_id = $3
		RETURNING id, todo_id, title, done`, todoID, title, userID).
		Scan(&st.ID, &owner, &st.Title, &st.Done)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" { // FK violation
			return st, ErrNotFound
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return st, ErrNotFound
		}
		return st, fmt.Errorf("add subtask: %w", err)
	}
	return st, nil
}

func (s *Store) SetSubtaskDone(ctx context.Context, userID, todoID, subtaskID int64, done bool) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE subtasks SET done = $4
		WHERE id = $2 AND todo_id = $1
		AND todo_id IN (SELECT id FROM todos WHERE id = $1 AND user_id = $3)`,
		todoID, subtaskID, userID, done)
	if err != nil {
		return fmt.Errorf("set subtask done: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteSubtask(ctx context.Context, userID, todoID, subtaskID int64) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM subtasks
		WHERE id = $2 AND todo_id = $1
		AND todo_id IN (SELECT id FROM todos WHERE id = $1 AND user_id = $3)`,
		todoID, subtaskID, userID)
	if err != nil {
		return fmt.Errorf("delete subtask: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) loadSubtasks(ctx context.Context, t *models.Todo) error {
	rows, err := s.pool.Query(ctx, `
		SELECT id, todo_id, title, done FROM subtasks
		WHERE todo_id = $1
		ORDER BY id`, t.ID)
	if err != nil {
		return fmt.Errorf("load subtasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			st     models.Subtask
			todoID int64
		)
		if err := rows.Scan(&st.ID, &todoID, &st.Title, &st.Done); err != nil {
			return fmt.Errorf("scan subtask: %w", err)
		}
		t.Subtasks = append(t.Subtasks, st)
	}
	return rows.Err()
}

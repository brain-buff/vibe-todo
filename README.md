# todos.

Минималистичный todo-лист в палитре Catppuccin Latte: **Go + Fiber + PostgreSQL**.
Бэкенд сам раздаёт статику (`static/`) — фронтенд без фреймворков.

Поля задачи: название, срок (время), описание, приоритет (низкий/средний/высокий),
подзадачи.

## Быстрый старт

```sh
docker compose up --build
```

Открыть http://localhost:3000

## Без Docker (нужен локальный PostgreSQL)

```sh
cp .env.example .env
# создать базу todos и пользователя todo (или поправить DATABASE_URL)
go mod tidy
go run .
```

## API

| Метод  | Путь                              | Описание                      |
| ------ | --------------------------------- | ----------------------------- |
| GET    | `/api/todos`                      | список задач (с подзадачами)  |
| POST   | `/api/todos`                      | создать задачу                |
| PUT    | `/api/todos/:id`                  | обновить задачу               |
| PATCH  | `/api/todos/:id/done`             | отметить выполненной `{done}` |
| DELETE | `/api/todos/done`                 | удалить все выполненные       |
| DELETE | `/api/todos/:id`                  | удалить задачу                |
| POST   | `/api/todos/:id/subtasks`         | добавить подзадачу `{title}`  |
| PATCH  | `/api/todos/:id/subtasks/:sid/done` | отметить подзадачу          |
| DELETE | `/api/todos/:id/subtasks/:sid`    | удалить подзадачу             |

Пример создания задачи:

```json
{
  "title": "Купить кофе",
  "description": "зерно, 500 г",
  "priority": 2,
  "due_at": "2026-10-27T14:00:00Z"
}
```

Приоритет: `1` — низкий, `2` — средний, `3` — высокий. `due_at` — опционально.

## Переменные окружения

| Переменная     | По умолчанию                                      |
| -------------- | ------------------------------------------------- |
| `PORT`         | `3000`                                            |
| `DATABASE_URL` | `postgres://todo:todo@localhost:5432/todos?sslmode=disable` |

Таблицы создаются автоматически при старте (`internal/store/store.go`).

# Task Management App

A task management REST API written in Go (Golang) with a MySQL database. Work in progress - endpoints are added as they are completed.

## Current endpoints

### POST /tasks
Creates a new task.

- Request body: `{"title": "Buy groceries"}`
- Validation: title is required, whitespace is trimmed, maximum 500 characters
- Success: `201 Created` with the created task, e.g. `{"id": 1, "title": "Buy groceries", "status": "todo"}`
- `409 Conflict` if a task with the same title already exists (case-insensitive)
- `400 Bad Request` for invalid JSON or a missing/too-long title

## Tech

- Go 1.27 - net/http with method-based routing (no framework)
- MySQL - github.com/go-sql-driver/mysql via database/sql
- JSON request/response bodies

## Setup

- MySQL running locally (127.0.0.1:3306) with a database named `taskmanagement`
- A `task` table with `title` and a unique `title_key` column
- Database credentials are read from the `DBUSER` and `DBPASS` environment variables (no secrets in code)

## Run

```
go run main.go
```

The server listens on http://localhost:8080.

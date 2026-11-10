# Task Management App

A single-user REST API built with Go's `net/http`, `database/sql`, and MySQL. It supports creating tasks, listing tasks, updating a task's title and/or status, and deleting tasks.

## Stack

- Go (the current `go.mod` declares `1.27.3`)
- Standard-library HTTP routing and JSON encoding/decoding
- MySQL with `github.com/go-sql-driver/mysql`

No authentication is implemented. For local Postman requests, use **No Auth**. This project is not yet Dockerized.

## Setup and run

1. Install a Go toolchain compatible with the version in `go.mod` and run MySQL locally at `127.0.0.1:3306`.
2. Create a database named `taskmanagement` and ensure that a `task` table exists. The repository does not include a schema or migration file, so database setup is not automated.

   The code expects these columns and behavior:

   | Column | Expected behavior |
   | --- | --- |
   | `id` | Unique positive integer, generated automatically on insert |
   | `title` | Stores the displayed title |
   | `title_key` | Stores the lowercased, trimmed title; must have a unique constraint for duplicate detection |
   | `status` | Defaults to `todo` when the INSERT omits this column |

   Exact SQL types and other constraints are not documented in the repository. Use the existing project schema rather than assuming the API creates the table.

3. Set database credentials in your shell. Example for Bash/Zsh; replace the placeholders locally and do not commit credentials:

   ```bash
   export DBUSER='your_mysql_user'
   export DBPASS='your_mysql_password'
   ```

4. From the repository directory, download dependencies and run the server:

   ```bash
   go mod download
   go run main.go
   ```

The program checks the database connection at startup and exits if it fails. On success, it prints `Success` and listens on port `8080`.

Base URL: `http://localhost:8080`

The database name, host, port, and HTTP port are currently set in `main.go`; only the database username and password are read from environment variables.

## Task format

```json
{
  "id": 1,
  "title": "Buy groceries",
  "status": "todo"
}
```

Task IDs in the examples are illustrative. Use IDs returned by your own requests.

## Endpoints

| Method | Path | Purpose | Success |
| --- | --- | --- | --- |
| POST | `/tasks` | Create a task | `201 Created` + task JSON |
| GET | `/tasks` | List all tasks | `200 OK` + JSON array |
| PATCH | `/tasks/{id}` | Update title and/or status | `200 OK` + task JSON |
| DELETE | `/tasks/{id}` | Delete one task | `204 No Content`, empty body |

### POST /tasks

```bash
curl -i -X POST http://localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Buy groceries"}'
```

Example response (`201 Created`):

```json
{"id":1,"title":"Buy groceries","status":"todo"}
```

- Trims leading and trailing whitespace.
- Requires a non-empty title after trimming.
- Rejects titles longer than 500 Unicode code points (Go runes).
- Inserts `title` and its lowercased `title_key`. The database must supply the default status `todo`.
- With the expected unique constraint, duplicate titles differing only by case or surrounding whitespace return `409 Conflict`.

Errors:

| Status | Error body |
| --- | --- |
| 400 | `Invalid JSON` |
| 400 | `Title is required` |
| 400 | `Title must be at most 500 characters` |
| 409 | `Task already exists` |
| 500 | `Internal Server Error` |

### GET /tasks

```bash
curl -i http://localhost:8080/tasks
```

Example response (`200 OK`):

```json
[
  {"id":1,"title":"Buy groceries","status":"todo"},
  {"id":2,"title":"Read Go docs","status":"in_progress"}
]
```

- Returns all tasks, or `[]` if the table is empty.
- No pagination, filtering, or guaranteed ordering is implemented.
- Database query, scan, or iteration errors return `500 Internal Server Error`.

### PATCH /tasks/{id}

Send a title, a status, or both. Fields omitted from the request retain their current values.

Title only:

```bash
curl -i -X PATCH http://localhost:8080/tasks/1 \
  -H 'Content-Type: application/json' \
  -d '{"title":"Buy groceries and milk"}'
```

Status only:

```bash
curl -i -X PATCH http://localhost:8080/tasks/1 \
  -H 'Content-Type: application/json' \
  -d '{"status":"in_progress"}'
```

Both fields:

```bash
curl -i -X PATCH http://localhost:8080/tasks/1 \
  -H 'Content-Type: application/json' \
  -d '{"title":"Buy groceries and milk","status":"done"}'
```

Example response to the last request (`200 OK`):

```json
{"id":1,"title":"Buy groceries and milk","status":"done"}
```

Behavior:

- The ID must parse as a positive integer.
- At least one non-null `title` or `status` must be supplied. `{}` and requests where both fields are omitted or null return `400`.
- A null field is treated like an omitted field. For example, `{"title":null,"status":"done"}` updates only the status.
- A supplied title is trimmed and must not be empty.
- A supplied status must be exactly `todo`, `in_progress`, or `done` (case-sensitive).
- All supplied fields are validated before the database update.
- Title changes also update `title_key`, preserving duplicate detection.
- Sending the same values for an existing task returns `200` with the unchanged task.
- PATCH also rejects titles longer than 500 characters with `400 Title must be at most 500 characters`.

Errors:

| Status | Condition / error body |
| --- | --- |
| 400 | Non-positive or unparseable ID: `id is not a number or a negative` |
| 400 | Invalid JSON: `Invalid JSON` |
| 400 | No non-null update fields: `title or status required` |
| 400 | Empty/whitespace-only title: `Invalid title` |
| 400 | Title longer than 500 characters: `Title must be at most 500 characters` |
| 400 | Unsupported status: `Invalid status` |
| 404 | Valid input, but task not found: `data not found` |
| 409 | Duplicate title key: `Task already exists` |
| 500 | Other database errors: `Internal Server Error` |

### DELETE /tasks/{id}

Use a disposable task when testing deletion.

```bash
curl -i -X DELETE http://localhost:8080/tasks/1
```

Success is `204 No Content` with no response body. Use GET `/tasks` to verify that the task no longer appears.

| Status | Condition / error body |
| --- | --- |
| 400 | Non-positive or unparseable ID: `id must be a positive number` |
| 404 | No row was deleted: `data not found` |
| 500 | Database execution or affected-row-count error: `Internal Server Error` |

Deleting the same ID again returns `404` after a successful deletion.

## Response and implementation notes

- Successful task responses are JSON; errors produced by the handlers are plain text, not JSON error objects.
- SQL values use placeholders rather than string concatenation.
- Input validation does not provide strict JSON-schema validation: unknown JSON fields are not rejected, and the handlers decode only one JSON value.
- The PATCH read and update are separate database operations, not a transaction. This implementation does not provide concurrency guarantees for simultaneous edits or deletes.
- A `500` response does not necessarily mean no database change occurred. For example, retrieving an inserted ID or reading the delete result can fail after the write.
- Docker deployment, authentication, and production hardening are not implemented in this version.
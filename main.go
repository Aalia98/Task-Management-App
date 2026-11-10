package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

type CreateTaskRequest struct {
	Title string `json:"title"`
}

type UpdateTaskRequest struct {
	Status string `json:"status"`
}

type Task struct {
	ID int `json:"id"`
	Title string `json:"title"`
	Status string `json:"status"`
}

var db *sql.DB

func createTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(req.Title)

	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(title) > 500 {
		http.Error(w, "Title must be at most 500 characters", http.StatusBadRequest)
		return
	}

	titleKey := strings.ToLower(title)
	result, err := db.Exec("INSERT INTO task (title, title_key) VALUES (?, ?)", title, titleKey)

	if err != nil {
		var mysqlErr *mysql.MySQLError

		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			http.Error(w, "Task already exists", http.StatusConflict)
			return 
		}

		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Println(err )
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	task := Task {
		ID: int(id),
		Title: title,
		Status: "todo",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(task); err != nil {
		log.Println(err)
	}
}

func listTasks(w http.ResponseWriter, r *http.Request) {
	
	rows, err := db.Query("SELECT id, title, status FROM task")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var t Task
	tasks := []Task{}

	for rows.Next() {
		err := rows.Scan(&t.ID, &t.Title, &t.Status)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		log.Println(err)
	}
}

func updateTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		http.Error(w, "id is not a number or a negative", http.StatusBadRequest)
		return
	}

	var req UpdateTaskRequest

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Status != "todo" && req.Status != "in_progress" && req.Status != "done"{
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	var t Task

	err = db.QueryRow("SELECT id, title, status FROM task WHERE id = ?", id).Scan(&t.ID, &t.Title, &t.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "data not found", http.StatusNotFound)
			return
		}

		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if req.Status == t.Status {
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(t); err != nil {
			log.Println(err)
		}
		return
	}

	_, err = db.Exec("UPDATE task SET status = ? WHERE id = ?", req.Status, id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	t.Status = req.Status
	
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(t); err != nil {
		log.Println(err)
	}
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		http.Error(w, "id must be a positive number", http.StatusBadRequest)
		return
	}

	result, err := db.Exec("DELETE FROM task WHERE id = ?", id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	count, err := result.RowsAffected()
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if count == 0 {
		http.Error(w, "data not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func main() {
	cfg := mysql.NewConfig()
	cfg.DBName = "taskmanagement"
	cfg.User = os.Getenv("DBUSER")
	cfg.Passwd = os.Getenv("DBPASS")
	cfg.Addr = "127.0.0.1:3306"
	cfg.Net = "tcp"

	var err error

	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Success")

	http.HandleFunc("POST /tasks", createTask)
	http.HandleFunc("GET /tasks", listTasks)
	http.HandleFunc("PATCH /tasks/{id}", updateTask)
	http.HandleFunc("DELETE /tasks/{id}", deleteTask)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
package main

import (
	"fmt"
	"os"
	"github.com/go-sql-driver/mysql"
	"database/sql"
	"log"
	"net/http"
	"encoding/json"
	"strings"
	"unicode/utf8"
	"errors"
)

type CreateTaskRequest struct {
	Title string `json:"title"`
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

	log.Fatal(http.ListenAndServe(":8080", nil))
}
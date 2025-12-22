package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

// DB global connection
var db *sql.DB

// Models
type Book struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Page struct {
	ID      int    `json:"id"`
	BookID  int    `json:"bookId"`
	Number  int    `json:"number"`
	Content string `json:"content"`
}

func main() {
	var err error
	// Connect to database
	db, err = sql.Open("sqlite3", "./books.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initDB(); err != nil {
		log.Fatal(err)
	}

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// API Endpoints
	http.HandleFunc("/api/books", handleBooks)
	http.HandleFunc("/api/books/", handleBookItem) // For DELETE /api/books/{id}
	http.HandleFunc("/api/pages", handlePages)
	http.HandleFunc("/api/pages/", handlePageItem) // For DELETE /api/pages/{id}

	port := "8000"
	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDB() error {
	// Create tables if they don't exist (based on current DB state, but ensuring schema)
	createBooksTable := `
	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`

	createPagesTable := `
	CREATE TABLE IF NOT EXISTS pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		book_id INTEGER NOT NULL,
		number INTEGER,
		content TEXT NOT NULL
	);`

	if _, err := db.Exec(createBooksTable); err != nil {
		return err
	}
	if _, err := db.Exec(createPagesTable); err != nil {
		return err
	}

	// Check if 'name' column exists and try to drop it (migration)
	// SQLite DROP COLUMN support added in 3.35.0 (2021).
	// If we can't drop it easily without complex migration, we can just ignore it.
	// But let's try a simple approach: if it exists, we just ignore it in queries, which is safer than complex table rebuilds for now.
	// If the user demanded schema cleanup we'd rename table -> copy -> drop old -> rename new.
	// For now, removing it from CREATE statement ensures new DBs are clean.
	// Existing DBs will still have the column but we won't use it.

	return nil
}

// Handler Constants
const (
	GET    = "GET"
	POST   = "POST"
	DELETE = "DELETE"
)

// -- Handlers --

func handleBooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case GET:
		rows, err := db.Query("SELECT id, name FROM books")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		books := []Book{}
		for rows.Next() {
			var b Book
			if err := rows.Scan(&b.ID, &b.Name); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			books = append(books, b)
		}
		json.NewEncoder(w).Encode(books)

	case POST:
		var b Book
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := db.Exec("INSERT INTO books (name) VALUES (?)", b.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		b.ID = int(id)
		json.NewEncoder(w).Encode(b)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleBookItem(w http.ResponseWriter, r *http.Request) {
	// Expect path /api/books/{id}
	idStr := filepath.Base(r.URL.Path)
	if idStr == "" || idStr == "books" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case DELETE:
		// First delete pages associated with book
		_, err := db.Exec("DELETE FROM pages WHERE book_id = ?", idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = db.Exec("DELETE FROM books WHERE id = ?", idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "PUT":
		var b Book
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, err := db.Exec("UPDATE books SET name = ? WHERE id = ?", b.Name, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		b.ID, _ = strconv.Atoi(idStr)
		json.NewEncoder(w).Encode(b)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case GET:
		// Check for bookId query param
		bookID := r.URL.Query().Get("bookId")
		query := "SELECT id, book_id, number, content FROM pages"
		var args []interface{}
		if bookID != "" {
			query += " WHERE book_id = ?"
			args = append(args, bookID)
		}
		query += " ORDER BY number ASC"

		rows, err := db.Query(query, args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		pages := []Page{}
		for rows.Next() {
			var p Page
			if err := rows.Scan(&p.ID, &p.BookID, &p.Number, &p.Content); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			pages = append(pages, p)
		}
		json.NewEncoder(w).Encode(pages)

	case POST:
		var p Page
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := db.Exec("INSERT INTO pages (book_id, number, content) VALUES (?, ?, ?)", p.BookID, p.Number, p.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		p.ID = int(id)
		json.NewEncoder(w).Encode(p)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePageItem(w http.ResponseWriter, r *http.Request) {
	idStr := filepath.Base(r.URL.Path)
	if idStr == "" || idStr == "pages" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case DELETE:
		_, err := db.Exec("DELETE FROM pages WHERE id = ?", idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "PUT":
		var p Page
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Update page
		_, err := db.Exec("UPDATE pages SET number = ?, content = ? WHERE id = ?", p.Number, p.Content, idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Return updated page (or just OK)
		p.ID, _ = strconv.Atoi(idStr)
		json.NewEncoder(w).Encode(p)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

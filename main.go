package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// DB global connection
var db *sql.DB

// Parsed templates
var templates *template.Template

// Models
type Book struct {
	ID   int
	Name string
}

type Page struct {
	ID      int
	BookID  int
	Name    string
	Number  int
	Content string
}

// Template data structures
type IndexData struct {
	Books       []Book
	CurrentBook *Book
	Pages       []Page
	Flash       string
	FlashType   string // "success" or "error"
	EditBookID  int
	EditPageID  int
	EditPage    *Page
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

	// Parse templates
	templates = template.Must(template.New("").Funcs(template.FuncMap{
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			// Truncate at rune boundary for Arabic text
			runes := []rune(s)
			if len(runes) <= n {
				return s
			}
			return string(runes[:n]) + "…"
		},
		"inc": func(v int) int {
			return v + 1
		},
	}).ParseGlob("templates/*.html"))

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Page routes
	http.HandleFunc("GET /{$}", handleIndex)
	http.HandleFunc("GET /books/{id}", handleBookView)
	http.HandleFunc("GET /books/{id}/edit", handleBookEdit)
	http.HandleFunc("GET /books/{bookID}/pages/{pageID}/edit", handlePageEdit)

	// Form actions
	http.HandleFunc("POST /books", handleCreateBook)
	http.HandleFunc("POST /books/{id}/update", handleUpdateBook)
	http.HandleFunc("POST /books/{id}/delete", handleDeleteBook)
	http.HandleFunc("POST /books/{id}/pages", handleCreatePage)
	http.HandleFunc("POST /books/{bookID}/pages/{pageID}/update", handleUpdatePage)
	http.HandleFunc("POST /books/{bookID}/pages/{pageID}/delete", handleDeletePage)

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
		name TEXT,
		number INTEGER,
		content TEXT NOT NULL
	);`

	if _, err := db.Exec(createBooksTable); err != nil {
		return err
	}
	if _, err := db.Exec(createPagesTable); err != nil {
		return err
	}

	// Check if 'name' column exists in 'pages' table (migration for existing DB)
	rows, err := db.Query("SELECT name FROM pages LIMIT 1")
	if err != nil {
		// Assume column missing
		fmt.Println("Adding 'name' column to pages table...")
		_, err = db.Exec("ALTER TABLE pages ADD COLUMN name TEXT DEFAULT ''")
		if err != nil {
			return fmt.Errorf("failed to add name column: %v", err)
		}
	} else {
		rows.Close()
	}

	return nil
}

// --- Helper functions ---

func getAllBooks() ([]Book, error) {
	rows, err := db.Query("SELECT id, name FROM books ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Name); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, nil
}

func getBook(id int) (*Book, error) {
	var b Book
	err := db.QueryRow("SELECT id, name FROM books WHERE id = ?", id).Scan(&b.ID, &b.Name)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func getPages(bookID int) ([]Page, error) {
	rows, err := db.Query("SELECT id, book_id, COALESCE(name, ''), number, content FROM pages WHERE book_id = ? ORDER BY number ASC", bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.BookID, &p.Name, &p.Number, &p.Content); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, nil
}

func getPage(id int) (*Page, error) {
	var p Page
	err := db.QueryRow("SELECT id, book_id, COALESCE(name, ''), number, content FROM pages WHERE id = ?", id).Scan(&p.ID, &p.BookID, &p.Name, &p.Number, &p.Content)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func filterBooks(books []Book, query string) []Book {
	if query == "" {
		return books
	}
	query = strings.ToLower(query)
	var filtered []Book
	for _, b := range books {
		if strings.Contains(strings.ToLower(b.Name), query) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// --- Handlers ---

func handleIndex(w http.ResponseWriter, r *http.Request) {
	books, err := getAllBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter books if search query present
	query := r.URL.Query().Get("q")
	displayBooks := filterBooks(books, query)

	data := IndexData{
		Books:     displayBooks,
		Flash:     r.URL.Query().Get("flash"),
		FlashType: r.URL.Query().Get("flash_type"),
	}

	if err := templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleBookView(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	books, err := getAllBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	book, err := getBook(id)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/?flash=Book+not+found&flash_type=error", http.StatusSeeOther)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pages, err := getPages(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter books if search query present
	query := r.URL.Query().Get("q")
	displayBooks := filterBooks(books, query)

	data := IndexData{
		Books:       displayBooks,
		CurrentBook: book,
		Pages:       pages,
		Flash:       r.URL.Query().Get("flash"),
		FlashType:   r.URL.Query().Get("flash_type"),
	}

	if err := templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleBookEdit(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	books, err := getAllBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	book, err := getBook(id)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/?flash=Book+not+found&flash_type=error", http.StatusSeeOther)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pages, err := getPages(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := IndexData{
		Books:       books,
		CurrentBook: book,
		Pages:       pages,
		EditBookID:  id,
	}

	if err := templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handlePageEdit(w http.ResponseWriter, r *http.Request) {
	bookIDStr := r.PathValue("bookID")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	pageIDStr := r.PathValue("pageID")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		http.Error(w, "Invalid page ID", http.StatusBadRequest)
		return
	}

	books, err := getAllBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	book, err := getBook(bookID)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/?flash=Book+not+found&flash_type=error", http.StatusSeeOther)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pages, err := getPages(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page, err := getPage(pageID)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, fmt.Sprintf("/books/%d?flash=Page+not+found&flash_type=error", bookID), http.StatusSeeOther)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := IndexData{
		Books:       books,
		CurrentBook: book,
		Pages:       pages,
		EditPageID:  pageID,
		EditPage:    page,
	}

	if err := templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleCreateBook(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/?flash=Book+name+is+required&flash_type=error", http.StatusSeeOther)
		return
	}

	res, err := db.Exec("INSERT INTO books (name) VALUES (?)", name)
	if err != nil {
		http.Redirect(w, r, "/?flash=Failed+to+create+book&flash_type=error", http.StatusSeeOther)
		return
	}

	id, _ := res.LastInsertId()
	http.Redirect(w, r, fmt.Sprintf("/books/%d?flash=Book+created&flash_type=success", id), http.StatusSeeOther)
}

func handleUpdateBook(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, fmt.Sprintf("/books/%d/edit?flash=Book+name+is+required&flash_type=error", id), http.StatusSeeOther)
		return
	}

	_, err = db.Exec("UPDATE books SET name = ? WHERE id = ?", name, id)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/books/%d?flash=Failed+to+update+book&flash_type=error", id), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/books/%d?flash=Book+updated&flash_type=success", id), http.StatusSeeOther)
}

func handleDeleteBook(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	// First delete pages associated with book
	_, err := db.Exec("DELETE FROM pages WHERE book_id = ?", idStr)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/books/%s?flash=Failed+to+delete+book&flash_type=error", idStr), http.StatusSeeOther)
		return
	}

	_, err = db.Exec("DELETE FROM books WHERE id = ?", idStr)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/books/%s?flash=Failed+to+delete+book&flash_type=error", idStr), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/?flash=Book+deleted&flash_type=success", http.StatusSeeOther)
}

func handleCreatePage(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	bookID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	numberStr := r.FormValue("number")
	content := strings.TrimSpace(r.FormValue("content"))

	if content == "" {
		http.Redirect(w, r, fmt.Sprintf("/books/%d?flash=Page+content+is+required&flash_type=error", bookID), http.StatusSeeOther)
		return
	}

	number, _ := strconv.Atoi(numberStr)

	_, err = db.Exec("INSERT INTO pages (book_id, name, number, content) VALUES (?, ?, ?, ?)", bookID, name, number, content)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/books/%d?flash=Failed+to+create+page&flash_type=error", bookID), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/books/%d?flash=Page+added&flash_type=success", bookID), http.StatusSeeOther)
}

func handleUpdatePage(w http.ResponseWriter, r *http.Request) {
	bookIDStr := r.PathValue("bookID")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	pageIDStr := r.PathValue("pageID")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		http.Error(w, "Invalid page ID", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	numberStr := r.FormValue("number")
	content := strings.TrimSpace(r.FormValue("content"))

	if content == "" {
		http.Redirect(w, r, fmt.Sprintf("/books/%d/pages/%d/edit?flash=Page+content+is+required&flash_type=error", bookID, pageID), http.StatusSeeOther)
		return
	}

	number, _ := strconv.Atoi(numberStr)

	_, err = db.Exec("UPDATE pages SET name = ?, number = ?, content = ? WHERE id = ?", name, number, content, pageID)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/books/%d?flash=Failed+to+update+page&flash_type=error", bookID), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/books/%d?flash=Page+updated&flash_type=success", bookID), http.StatusSeeOther)
}

func handleDeletePage(w http.ResponseWriter, r *http.Request) {
	bookIDStr := r.PathValue("bookID")
	pageIDStr := r.PathValue("pageID")

	_, err := db.Exec("DELETE FROM pages WHERE id = ?", pageIDStr)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/books/%s?flash=Failed+to+delete+page&flash_type=error", bookIDStr), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/books/%s?flash=Page+deleted&flash_type=success", bookIDStr), http.StatusSeeOther)
}

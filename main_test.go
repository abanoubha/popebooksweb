package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_books.db")

	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := initDB(); err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})
}

func TestPageCRUDWithName(t *testing.T) {
	setupTestDB(t)

	// 1. Create a Book
	bookPayload := []byte(`{"name":"Test Book"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewBuffer(bookPayload))
	rec := httptest.NewRecorder()
	handleBooks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var createdBook Book
	if err := json.NewDecoder(rec.Body).Decode(&createdBook); err != nil {
		t.Fatalf("failed to decode created book: %v", err)
	}

	// 2. Create Page (POST) with Name
	pageInput := Page{
		BookID:  createdBook.ID,
		Name:    "Chapter 1 - Introduction",
		Number:  1,
		Content: "Welcome to chapter one.",
	}
	pageBytes, _ := json.Marshal(pageInput)
	req = httptest.NewRequest(http.MethodPost, "/api/pages", bytes.NewBuffer(pageBytes))
	rec = httptest.NewRecorder()
	handlePages(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var createdPage Page
	if err := json.NewDecoder(rec.Body).Decode(&createdPage); err != nil {
		t.Fatalf("failed to decode created page: %v", err)
	}
	if createdPage.ID == 0 {
		t.Errorf("expected non-zero page ID")
	}
	if createdPage.Name != "Chapter 1 - Introduction" {
		t.Errorf("expected page name 'Chapter 1 - Introduction', got '%s'", createdPage.Name)
	}
	if createdPage.Number != 1 {
		t.Errorf("expected page number 1, got %d", createdPage.Number)
	}

	// 3. Read Pages (GET list by bookId)
	req = httptest.NewRequest(http.MethodGet, "/api/pages?bookId="+strconv.Itoa(createdBook.ID), nil)
	rec = httptest.NewRecorder()
	handlePages(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var pages []Page
	if err := json.NewDecoder(rec.Body).Decode(&pages); err != nil {
		t.Fatalf("failed to decode pages: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if pages[0].Name != "Chapter 1 - Introduction" {
		t.Errorf("expected page name 'Chapter 1 - Introduction', got '%s'", pages[0].Name)
	}

	// 4. Read single Page by ID (GET /api/pages/{id})
	pageIDStr := strconv.Itoa(createdPage.ID)
	req = httptest.NewRequest(http.MethodGet, "/api/pages/"+pageIDStr, nil)
	rec = httptest.NewRecorder()
	handlePageItem(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on GET /api/pages/{id}, got %d: %s", rec.Code, rec.Body.String())
	}
	var singlePage Page
	if err := json.NewDecoder(rec.Body).Decode(&singlePage); err != nil {
		t.Fatalf("failed to decode single page: %v", err)
	}
	if singlePage.Name != "Chapter 1 - Introduction" {
		t.Errorf("expected page name 'Chapter 1 - Introduction', got '%s'", singlePage.Name)
	}

	// 5. Update Page (PUT /api/pages/{id}) including Name
	updateInput := Page{
		BookID:  createdBook.ID,
		Name:    "Chapter 1 - Updated Title",
		Number:  2,
		Content: "Updated content.",
	}
	updateBytes, _ := json.Marshal(updateInput)
	req = httptest.NewRequest(http.MethodPut, "/api/pages/"+pageIDStr, bytes.NewBuffer(updateBytes))
	rec = httptest.NewRecorder()
	handlePageItem(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on PUT, got %d: %s", rec.Code, rec.Body.String())
	}
	var updatedPage Page
	if err := json.NewDecoder(rec.Body).Decode(&updatedPage); err != nil {
		t.Fatalf("failed to decode updated page: %v", err)
	}
	if updatedPage.Name != "Chapter 1 - Updated Title" {
		t.Errorf("expected updated page name 'Chapter 1 - Updated Title', got '%s'", updatedPage.Name)
	}
	if updatedPage.Number != 2 {
		t.Errorf("expected updated page number 2, got %d", updatedPage.Number)
	}

	// Verify update persisted via GET
	req = httptest.NewRequest(http.MethodGet, "/api/pages/"+pageIDStr, nil)
	rec = httptest.NewRecorder()
	handlePageItem(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on GET after update, got %d: %s", rec.Code, rec.Body.String())
	}
	var getUpdatedPage Page
	if err := json.NewDecoder(rec.Body).Decode(&getUpdatedPage); err != nil {
		t.Fatalf("failed to decode page: %v", err)
	}
	if getUpdatedPage.Name != "Chapter 1 - Updated Title" {
		t.Errorf("expected persisted page name 'Chapter 1 - Updated Title', got '%s'", getUpdatedPage.Name)
	}

	// 6. Delete Page (DELETE /api/pages/{id})
	req = httptest.NewRequest(http.MethodDelete, "/api/pages/"+pageIDStr, nil)
	rec = httptest.NewRecorder()
	handlePageItem(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on DELETE, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify deletion
	req = httptest.NewRequest(http.MethodDelete, "/api/pages/"+pageIDStr, nil)
	rec = httptest.NewRecorder()
	handlePageItem(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on DELETE, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify deletion
	req = httptest.NewRequest(http.MethodGet, "/api/pages/"+pageIDStr, nil)
	rec = httptest.NewRecorder()
	handlePageItem(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 after deletion, got %d", rec.Code)
	}
}

func TestPageScanWithNullName(t *testing.T) {
	setupTestDB(t)

	// Insert book
	res, err := db.Exec("INSERT INTO books (name) VALUES ('Test Book')")
	if err != nil {
		t.Fatalf("insert book failed: %v", err)
	}
	bookID, _ := res.LastInsertId()

	// Insert page with NULL name directly into DB
	_, err = db.Exec("INSERT INTO pages (book_id, name, number, content) VALUES (?, NULL, 1, 'Content without name')", bookID)
	if err != nil {
		t.Fatalf("insert page with NULL name failed: %v", err)
	}

	// Query via GET /api/pages
	req := httptest.NewRequest(http.MethodGet, "/api/pages?bookId="+strconv.Itoa(int(bookID)), nil)
	rec := httptest.NewRecorder()
	handlePages(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var pages []Page
	if err := json.NewDecoder(rec.Body).Decode(&pages); err != nil {
		t.Fatalf("failed to decode pages: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if pages[0].Name != "" {
		t.Errorf("expected empty string for NULL name, got '%s'", pages[0].Name)
	}
}

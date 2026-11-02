# Pope Books for Web

A web user interface for the books of Pope Shenouda III.

## Why ?

I created this web UI project to facilitate adding, editing, and proofreading the books.

## Architecture

Server-side rendered HTML using Go's `html/template` package and `net/http`. No frontend framework or JavaScript API calls — just plain HTML forms with POST/redirect/GET pattern.

### Structure

```
├── main.go              # HTTP handlers, DB queries, template rendering
├── templates/
│   ├── layout.html      # Base layout with head, flash messages, sidebar+main
│   ├── sidebar.html     # Book list navigation, add book form, search
│   ├── book_view.html   # Book details, page cards, edit forms
│   └── empty_state.html # Shown when no book is selected
├── static/
│   └── style.css        # Dark theme CSS
├── books.db             # SQLite database (not committed)
├── go.mod
└── go.sum
```

### Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Home — book list, empty state |
| GET | `/books/{id}` | View book and its pages |
| GET | `/books/{id}/edit` | Inline edit book name |
| GET | `/books/{bookID}/pages/{pageID}/edit` | Inline edit page |
| POST | `/books` | Create a new book |
| POST | `/books/{id}/update` | Update book name |
| POST | `/books/{id}/delete` | Delete book and its pages |
| POST | `/books/{id}/pages` | Create a new page |
| POST | `/books/{bookID}/pages/{pageID}/update` | Update a page |
| POST | `/books/{bookID}/pages/{pageID}/delete` | Delete a page |

## Wanna read Pope's books ?

Download the app on your Android phone from [Google Play](https://play.google.com/store/apps/details?id=com.softwarepharaoh.popebooks)! (books are in Arabic)

## vision

- include all pope's books
- include all pope's published articles
- include all poems
- (re)compose them as technical components such as H1, P, UL, OL, IMG, .. then scroll vertically as it is more familiar for people nowadays.
- convert each separate topic into (nested/interlinked) infographic(s) using Gemini Nano Banana Pro (AI/LLM)

## commands

```sh
go run ./...
```

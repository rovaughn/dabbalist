package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func InternalError(ctx string, w http.ResponseWriter, err error) {
	log.Println(ctx, err)
	http.Error(w, "Internal server error.", http.StatusInternalServerError)
}

type Book struct {
	ID         int
	Title      string
	Author     string
	Translator string
}

type Verse struct {
	No    int
	Verse string
}

func GetBooks(db *sql.DB) ([]Book, error) {
	books := make([]Book, 0)

	rows, err := db.Query(`SELECT id, title, author, translator FROM book`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var book Book

		if err := rows.Scan(
			&book.ID, &book.Title, &book.Author, &book.Translator,
		); err != nil {
			return nil, err
		}

		books = append(books, book)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return books, nil
}

func GetBook(db *sql.DB, id int) (book Book, err error) {
	book.ID = id

	err = db.QueryRow(
		`SELECT title, author, translator FROM book WHERE id = $1`, id,
	).Scan(&book.Title, &book.Author, &book.Translator)

	return
}

func GetVerses(db *sql.DB, book int) ([]Verse, error) {
	verses := make([]Verse, 0)

	rows, err := db.Query(`
		SELECT no, verse FROM verse WHERE book = $1 ORDER BY no
	`, book)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var verse Verse

		if err := rows.Scan(&verse.No, &verse.Verse); err != nil {
			return nil, err
		}

		verses = append(verses, verse)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return verses, nil
}

// TODO: This isn't actually HTML safe.
func FormatVerse(verse string) template.HTML {
	return template.HTML(strings.Replace(verse, "\n", "<br>", -1))
}

func main() {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"FormatVerse": FormatVerse,
	}).ParseGlob("*.html")
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", "user=postgres password=postgres dbname=dabbalist")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/book/", func(w http.ResponseWriter, r *http.Request) {
		bookID, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/book/"))
		if err != nil {
			http.Error(w, "Book ID must be an integer.", http.StatusBadRequest)
			return
		}

		book, err := GetBook(db, bookID)
		if err == sql.ErrNoRows {
			http.Error(w, "That book doesn't exist.", http.StatusNotFound)
			return
		} else if err != nil {
			InternalError("/book/ GetBook", w, err)
			return
		}

		verses, err := GetVerses(db, bookID)
		if err != nil {
			InternalError("/book/ GetVerses", w, err)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "book.html", struct {
			Book   Book
			Verses []Verse
		}{
			Book:   book,
			Verses: verses,
		}); err != nil {
			InternalError("/book/ render", w, err)
		}
	})

	http.HandleFunc("/main.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "main.css")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		books, err := GetBooks(db)
		if err != nil {
			InternalError("/ GetBooks", w, err)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "index.html", struct {
			Books []Book
		}{
			Books: books,
		}); err != nil {
			InternalError("/ render", w, err)
		}
	})

	log.Println("Listening on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

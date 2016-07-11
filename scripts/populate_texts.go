package main

import (
	"database/sql"
	"github.com/go-yaml/yaml"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"path/filepath"
)

func main() {
	db, err := sql.Open("postgres", "user=postgres password=postgres dbname=dabbalist")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS book (
			id serial primary key,
			title text,
			author text,
			translator text,
			unique (title, author, translator)
		);
	`); err != nil {
		panic(err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS verse (
			book integer,
			no integer,
			verse text,
			primary key (book, no)
		);
	`); err != nil {
		panic(err)
	}

	filenames, err := filepath.Glob("books/*.yml")
	if err != nil {
		panic(err)
	}

	for _, filename := range filenames {
		log.Println("Processing", filename, "...")

		var book struct {
			Title       string   `yaml:"book"`
			Author      string   `yaml:"author"`
			Translator  string   `yaml:"translator"`
			Verses      []string `yaml:"verses"`
			VerseMargin bool     `yaml:"verse-margin"`
		}

		yaml_bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(err)
		}

		if err := yaml.Unmarshal(yaml_bytes, &book); err != nil {
			panic(err)
		}

		var bookId int

		if err := db.QueryRow(`
			INSERT INTO book (title, author, translator)
			VALUES ($1, $2, $3)
			ON CONFLICT ON CONSTRAINT book_title_author_translator_key
			DO UPDATE SET title = EXCLUDED.title
			RETURNING id;
		`, book.Title, book.Author, book.Translator).Scan(&bookId); err != nil {
			panic(err)
		}

		for no, verse := range book.Verses {
			if book.VerseMargin {
				verse += "\n"
			}

			if _, err := db.Exec(`
				INSERT INTO verse (book, no, verse)
				VALUES ($1, $2, $3)
				ON CONFLICT ON CONSTRAINT verse_pkey
				DO UPDATE SET verse = $3;
			`, bookId, no+1, verse); err != nil {
				panic(err)
			}
		}
	}
}

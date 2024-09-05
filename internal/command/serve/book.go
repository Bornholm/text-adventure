package serve

import (
	"bytes"
	"context"
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	sloghttp "github.com/samber/slog-http"
	"github.com/urfave/cli/v2"
	"github.com/yuin/goldmark"
)

//go:embed templates/page.gotmpl
var rawPageTemplate string
var pageTemplate *template.Template

//go:embed templates/index.gotmpl
var rawIndexTemplate string
var indexTemplate *template.Template

//go:embed public
var publicFS embed.FS

func init() {
	pageTemplate = template.Must(template.New("").Funcs(template.FuncMap{
		"md": func(text string) (template.HTML, error) {
			var buff bytes.Buffer
			if err := goldmark.Convert([]byte(text), &buff); err != nil {
				return "", err
			}
			return template.HTML(buff.String()), nil
		},
	}).Parse(rawPageTemplate))

	indexTemplate = template.Must(template.New("").Parse(rawIndexTemplate))
}

func BookCommand() *cli.Command {
	return &cli.Command{
		Name:  "book",
		Usage: "Serve a book",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "address",
				Value: ":3000",
			},
		},
		Action: func(ctx *cli.Context) error {
			address := ctx.String("address")

			mux := http.NewServeMux()

			mux.HandleFunc("GET /{$}", handleIndex)
			mux.HandleFunc("GET /{book}/p/{page}", handlePage)
			mux.HandleFunc("GET /{book}/cover.png", handleCover)

			publicDir, err := fs.Sub(publicFS, "public")
			if err != nil {
				return err
			}

			handlePublic := http.FileServer(http.FS(publicDir))

			mux.Handle("GET /", handlePublic)

			handler := sloghttp.New(slog.Default())(mux)

			slog.Info("starting server", slog.Any("address", address))

			if err := http.ListenAndServe(address, handler); err != nil {
				return err
			}

			return nil
		},
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	books, err := loadBooks(r.Context())
	if err != nil {
		slog.Error("could not load books", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = indexTemplate.Execute(w, struct {
		Books []*Book
	}{
		Books: books,
	})
	if err != nil {
		slog.Error("could not generate page", slog.Any("error", err))
	}
}

func handleCover(w http.ResponseWriter, r *http.Request) {
	book := r.PathValue("book")
	http.ServeFile(w, r, fmt.Sprintf("%s/cover.png", book))
}

func handlePage(w http.ResponseWriter, r *http.Request) {
	bookName := r.PathValue("book")
	pageIndex := r.PathValue("page")

	// TODO Prevent potential path traversal
	page, err := os.ReadFile(fmt.Sprintf("%s/%s.md", bookName, pageIndex))
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		slog.Error("could not read page", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	book, err := loadBook(r.Context(), bookName)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		slog.Error("could not parse book", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	pageTemplate.Execute(w, struct {
		Book *Book
		Page string
	}{
		Book: book,
		Page: string(page),
	})
}

type Book struct {
	Name      string
	Title     string    `json:"title"`
	Story     string    `json:"story"`
	Authors   []string  `json:"authors"`
	Model     string    `json:"model"`
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"createdAt"`
}

func loadBooks(ctx context.Context) ([]*Book, error) {
	files, err := filepath.Glob("./*/book.json")
	if err != nil {
		return nil, err
	}

	books := make([]*Book, 0, len(files))

	for _, f := range files {
		dirname := filepath.Dir(f)

		b, err := loadBook(ctx, dirname)
		if err != nil {
			return nil, err
		}

		books = append(books, b)
	}

	return books, nil
}

func loadBook(ctx context.Context, dirname string) (*Book, error) {
	data, err := os.ReadFile(filepath.Join(dirname, "book.json"))
	if err != nil {
		return nil, err
	}

	b := &Book{}

	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}

	b.Name = dirname

	return b, nil
}

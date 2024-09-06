package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"text/template"
	"time"

	"github.com/Bornholm/ai-adventure/internal/data"
	"github.com/sashabaranov/go-openai"
	"github.com/urfave/cli/v2"

	_ "embed"
)

//go:embed prompts/system.gotmpl
var systemPromptTmpl string

//go:embed prompts/page.gotmpl
var pagePromptTmpl string

//go:embed prompts/title.gotmpl
var titlePromptTmpl string

func BookCommand() *cli.Command {
	return &cli.Command{
		Name:  "book",
		Usage: "Generate a new book",
		Subcommands: []*cli.Command{
			CoverCommand(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "api-token",
				EnvVars: []string{"API_TOKEN"},
			},
			&cli.Int64Flag{
				Name:    "total-pages",
				EnvVars: []string{"TOTAL_PAGES"},
				Value:   10,
			},
			&cli.StringSliceFlag{
				Name:    "authors",
				EnvVars: []string{"AUTHORS"},
				Value:   cli.NewStringSlice("David Gemmel", "H.P. Lovecrat", "Robert E. Howard"),
			},
			&cli.StringFlag{
				Name:    "story",
				EnvVars: []string{"STORY"},
				Value:   `Une épopée d'un jeune chasseur perdu en pleine forêt après une journée de chasse, et qui sera confronté à des rencontres de plus en plus effrayantes et fantastique au fur et à mesure qu'il s'enfoncera dans la forêt`,
			},
			&cli.StringFlag{
				Name:    "language",
				EnvVars: []string{"LANGUAGE"},
				Value:   `français`,
			},
		},
		Action: func(ctx *cli.Context) error {
			apiToken := ctx.String("api-token")
			totalPages := ctx.Int("total-pages")
			authors := ctx.StringSlice("authors")
			story := ctx.String("story")
			language := ctx.String("language")

			contextPromptData := ContextPromptData{
				Authors:  authors,
				Story:    story,
				Language: language,
			}

			client := openai.NewClient(apiToken)

			remaining := data.NewSet[int]()
			for p := range totalPages {
				remaining.Add(1 + p)
			}

			remaining.Remove(1)

			root := Branch{
				Page:     1,
				Children: pickPages(remaining, 2),
			}

			branches, err := generateNext(ctx.Context, client, remaining, root, 2, 4, contextPromptData)
			if err != nil {
				return err
			}

			title, err := generateTitle(ctx.Context, client, contextPromptData)
			if err != nil {
				return err
			}

			slog.Info("generated book title", slog.Any("title", title))

			metadata := BookMetadata{
				Title:     title,
				Story:     story,
				Authors:   authors,
				Language:  language,
				Model:     "gpt-4o",
				CreatedAt: time.Now(),
			}

			slog.Info("saving book metadata", slog.Any("metadata", metadata))

			if err := saveBookMetadata(ctx.Context, metadata); err != nil {
				return err
			}

			var next *Branch
			for {
				if len(branches) == 0 {
					break
				}

				next, branches = branches[0], branches[1:]

				for _, p := range next.Children {
					branch := Branch{
						Parent:   &next.Page,
						Page:     p,
						Children: pickPages(remaining, 2),
					}

					newBranches, err := generateNext(ctx.Context, client, remaining, branch, 1, 4, contextPromptData)
					if err != nil {
						return err
					}

					slog.Info("remaining pages", slog.Any("pages", remaining.Len()))

					branches = append(branches, newBranches...)

				}
			}

			return nil
		},
	}
}

type Branch struct {
	Parent   *int
	Page     int
	Children []int
}

func intPtr(v int) *int {
	return &v
}

func asIntOrNil(v *int) any {
	if v == nil {
		return nil
	}

	return *v
}

func generateNext(ctx context.Context, client *openai.Client, pages *data.Set[int], branch Branch, minLinks int, maxLinks int, contextPromptData ContextPromptData) ([]*Branch, error) {
	var (
		parentPage []byte
		err        error
	)
	if branch.Parent != nil {
		parentPage, err = os.ReadFile(fmt.Sprintf("%d.md", *branch.Parent))
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if os.IsNotExist(err) {
			slog.Warn("could not find parent page", slog.Any("page", asIntOrNil(branch.Parent)))
		}
	}

	var pageContent string
	if pageData, err := os.ReadFile(fmt.Sprintf("%d.md", branch.Page)); err == nil {
		pageContent = string(pageData)
	}

	if pageContent != "" {
		slog.Info("page already generated", slog.Any("page", branch.Page))
	} else {
		prompt, err := generatePagePrompt(ctx, PagePromptData{
			ContextPromptData: contextPromptData,
			Page:              branch.Page,
			Children:          branch.Children,
			Parent:            string(parentPage),
		})
		if err != nil {
			return nil, err
		}

		slog.Info("generating page", slog.Any("page", branch.Page), slog.Any("parent", asIntOrNil(branch.Parent)), slog.Any("children", branch.Children))

		pageContent, err = generate(ctx, client, prompt)
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(fmt.Sprintf("%d.md", branch.Page), []byte(pageContent), 0644); err != nil {
			return nil, err
		}

		data, err := json.MarshalIndent(branch, "", " ")
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(fmt.Sprintf("%d.json", branch.Page), data, 0644); err != nil {
			return nil, err
		}
	}

	parentPageContent := pageContent

	branches := make([]*Branch, 0)

	for _, p := range branch.Children {
		links := rand.IntN(maxLinks)
		if links < minLinks {
			links = minLinks
		}

		children := pickPages(pages, links)

		branch := &Branch{
			Parent:   intPtr(branch.Page),
			Page:     p,
			Children: children,
		}

		var pageContent string
		if pageData, err := os.ReadFile(fmt.Sprintf("%d.md", p)); err == nil {
			pageContent = string(pageData)
		}

		if pageContent != "" {
			slog.Info("child page already generated", slog.Any("page", branch.Page))
		} else {
			prompt, err := generatePagePrompt(ctx, PagePromptData{
				ContextPromptData: contextPromptData,
				Page:              p,
				Children:          children,
				Parent:            parentPageContent,
			})
			if err != nil {
				return nil, err
			}

			slog.Info("generating child page", slog.Any("page", p), slog.Any("parent", branch.Page), slog.Any("children", children))

			content, err := generate(ctx, client, prompt)
			if err != nil {
				return nil, err
			}

			if err := os.WriteFile(fmt.Sprintf("%d.md", p), []byte(content), 0644); err != nil {
				return nil, err
			}

			data, err := json.MarshalIndent(branch, "", " ")
			if err != nil {
				return nil, err
			}

			if err := os.WriteFile(fmt.Sprintf("%d.json", p), data, 0644); err != nil {
				return nil, err
			}
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

func pickPages(set *data.Set[int], count int) []int {
	pages := []int{}
	for range count {
		available := set.All()
		if len(available) == 0 {
			break
		}

		randIndex := rand.IntN(len(available))
		page := available[randIndex]
		set.Remove(page)
		pages = append(pages, page)
	}

	return pages
}

func generateTitle(ctx context.Context, client *openai.Client, contextPromptData ContextPromptData) (string, error) {
	firstPage, err := os.ReadFile("1.md")
	if err != nil {
		return "", err
	}

	prompt, err := generateTitlePrompt(ctx, TitlePromptData{
		ContextPromptData: contextPromptData,
		FirstPage:         string(firstPage),
	})
	if err != nil {
		return "", err
	}

	title, err := generate(ctx, client, prompt)
	if err != nil {
		return "", err
	}

	return title, nil
}

func generate(ctx context.Context, client *openai.Client, prompt string) (string, error) {
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: systemPromptTmpl,
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: `D'accord, je comprends ! Je suis prêt à écrire des pages en fonction du contexte et des embranchements demandés. Donne-moi le contexte et les choix que tu souhaites, et je m'occuperai du reste !`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

type SystemPromptData struct {
}

func generateSystemPrompt(ctx context.Context, data SystemPromptData) (string, error) {
	tmpl, err := template.New("").Parse(systemPromptTmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type ContextPromptData struct {
	Authors  []string
	Language string
	Story    string
}

type PagePromptData struct {
	ContextPromptData
	Children []int
	Page     int
	Parent   string
}

func generatePagePrompt(ctx context.Context, data PagePromptData) (string, error) {
	tmpl, err := template.New("").Parse(pagePromptTmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type TitlePromptData struct {
	ContextPromptData
	FirstPage string
}

func generateTitlePrompt(ctx context.Context, data TitlePromptData) (string, error) {
	tmpl, err := template.New("").Parse(titlePromptTmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type BookMetadata struct {
	Title     string    `json:"title"`
	Story     string    `json:"story"`
	Authors   []string  `json:"authors"`
	Model     string    `json:"model"`
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"createdAt"`
}

func saveBookMetadata(ctx context.Context, metadata BookMetadata) error {
	data, err := json.MarshalIndent(metadata, "", " ")
	if err != nil {
		return err
	}

	if err := os.WriteFile("book.json", data, 0640); err != nil {
		return err
	}

	return nil
}

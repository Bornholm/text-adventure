package generate

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"log/slog"
	"os"
	"text/template"

	"github.com/sashabaranov/go-openai"
	"github.com/urfave/cli/v2"

	_ "embed"
)

//go:embed prompts/cover.gotmpl
var coverPromptTmpl string

func CoverCommand() *cli.Command {
	return &cli.Command{
		Name:  "cover",
		Usage: "Generate a new cover",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "api-token",
				EnvVars: []string{"API_TOKEN"},
			},
		},
		Action: func(ctx *cli.Context) error {
			apiToken := ctx.String("api-token")

			metadata := BookMetadata{}

			slog.Info("reading book metadata", slog.Any("file", "book.json"))

			rawMetadata, err := os.ReadFile("book.json")
			if err != nil {
				return err
			}

			if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
				return err
			}

			firstPage, err := os.ReadFile("1.md")
			if err != nil {
				return err
			}

			prompt, err := generateCoverPrompt(ctx.Context, CoverPromptData{
				Story:     metadata.Story,
				Title:     metadata.Title,
				FirstPage: string(firstPage),
			})
			if err != nil {
				return err
			}

			client := openai.NewClient(apiToken)

			if err := generateCover(ctx.Context, client, prompt); err != nil {
				return err
			}

			return nil
		},
	}
}

type CoverPromptData struct {
	Story     string
	Title     string
	FirstPage string
}

func generateCoverPrompt(ctx context.Context, data CoverPromptData) (string, error) {
	tmpl, err := template.New("").Parse(coverPromptTmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func generateCover(ctx context.Context, client *openai.Client, prompt string) error {
	req := openai.ImageRequest{
		Prompt:         prompt,
		Size:           openai.CreateImageSize1024x1792,
		Model:          openai.CreateImageModelDallE3,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		N:              1,
	}

	res, err := client.CreateImage(ctx, req)
	if err != nil {
		return err
	}

	imgData, err := base64.StdEncoding.DecodeString(res.Data[0].B64JSON)
	if err != nil {
		return err
	}

	r := bytes.NewReader(imgData)

	pngData, err := png.Decode(r)
	if err != nil {
		return err
	}

	file, err := os.Create("cover.png")
	if err != nil {
		return err
	}

	defer file.Close()

	if err := png.Encode(file, pngData); err != nil {
		return err
	}

	return nil
}

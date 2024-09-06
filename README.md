# Text Adventure

A quick and dirty experiment to build "game-books" with LLMs (OpenAI API for now).

## Demo

[![asciicast](https://asciinema.org/a/AsZhBBWyY5ua1ihZTrgrm58pi.svg)](https://asciinema.org/a/AsZhBBWyY5ua1ihZTrgrm58pi)

Some of the books i've generated are visible here (in french):

https://text-adventure.dev.lookingfora.name/

## Getting started

### Generating a new book

```bash
# Export your OpenAI API secret token
export API_TOKEN=<token>

# Create book directory
mkdir books/my-book

# Generate your book
go run ./cmd/cli --workdir books/my-book generate book --story "<your story context>" --authors "<name of a well known author>"
```

**Example**

```bash
mkdir books/my-polar

go run ./cmd/cli --workdir books/my-polar generate book --story "Un polar dans le style roman noir/thriller, racontant l'enquête pour meurtre d'un vieil enquêteur désabusé dans un quartier malfamé de la ville Lilian Sorrow. Écris à la première personne." --authors "James Lee Burkee" --authors "Raymond Chandler" --authors "Peter Cheyney"
```

### Serving book(s)

```bash
go run ./cmd/cli --workdir books serve book
```

Then open http://localhost:3000 in your browser

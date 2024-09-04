FROM golang:1.23 AS build

COPY . /src

WORKDIR /src

RUN CGO_ENABLED=0 go build -o ./text-adventure ./cmd/cli

FROM scratch AS run

COPY --from=build /src/text-adventure /app/text-adventure

WORKDIR /app

COPY data /books

EXPOSE 3000

CMD ["/app/text-adventure", "--workdir", "/books", "serve", "book"]

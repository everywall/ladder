# Building the binary of the App
FROM golang:1.21 AS build

WORKDIR /go/src/ladder

COPY . .

# Create VERSION file for embed directive
RUN echo "docker-build" > handlers/VERSION

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o ladder cmd/main.go

FROM debian:12-slim AS release

WORKDIR /app

COPY --from=build /go/src/ladder/ladder .
RUN chmod +x /app/ladder

RUN apt update && apt install -y ca-certificates && rm -rf /var/lib/apt/lists/*

#EXPOSE 8080

#ENTRYPOINT ["/usr/bin/dumb-init", "--"]

CMD ["sh", "-c", "/app/ladder"]
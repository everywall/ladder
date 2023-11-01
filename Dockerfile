# Building the binary of the App
FROM golang:1.21 AS build

WORKDIR /go/src/ladder

# Copy all the Code and stuff to compile everything
COPY . .

# Downloads all the dependencies in advance (could be left out, but it's more clear this way)
RUN go mod download

# Builds the application as a staticly linked one, to allow it to run on alpine
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o ladder cmd/main.go
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ladder cmd/main.go
#RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -o ladder cmd/main.go
#RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags netgo -a -installsuffix cgo -o ladder cmd/main.go
#RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags netgo -a -o ladder cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o ladder cmd/main.go


# Moving the binary to the 'final Image' to make it smaller
FROM debian:12-slim as release
#FROM debian:latest as release
#FROM ubuntu:latest as release
#FROM golang:bookworm as release

WORKDIR /app

# Create the `public` dir and copy all the assets into it
RUN mkdir ./public
COPY ./public ./public

COPY --from=build /go/src/ladder/ladder .
RUN chmod +x /app/ladder

RUN apt update && apt install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Exposes port 2000 and 3000 because our program listens on that port
#EXPOSE 2000
#EXPOSE 3000

#ENTRYPOINT ["/usr/bin/dumb-init", "--"]
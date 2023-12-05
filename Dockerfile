# Building the binary of the App
FROM golang:1.21 AS build

WORKDIR /go/src/ladder

COPY . .

RUN go mod download

RUN make build

FROM debian:12-slim as release

WORKDIR /app

COPY --from=build /go/src/ladder/ladder .
RUN chmod +x /app/ladder

RUN apt update && apt install -y ca-certificates && rm -rf /var/lib/apt/lists/*

CMD ["sh", "-c", "/app/ladder"]
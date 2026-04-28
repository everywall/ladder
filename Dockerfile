# Build the binary of the App
ARG VERSION=build

FROM golang:1.26 AS build
WORKDIR /go/src/ladder

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X ladder/handlers.version=${VERSION}" -o ladder cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o healthcheck cmd/healthcheck/main.go

# Build the release image
FROM gcr.io/distroless/static-debian13:nonroot AS release

WORKDIR /app

COPY --from=build /go/src/ladder/ladder .
COPY --from=build /go/src/ladder/healthcheck .

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/app/healthcheck"]

EXPOSE 8080

ENTRYPOINT ["/app/ladder"]
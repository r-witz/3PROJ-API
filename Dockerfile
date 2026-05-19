FROM golang:1.25.10-alpine AS build

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api/main.go

FROM gcr.io/distroless/static-debian13

WORKDIR /

COPY --from=build /app/main /main

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/main"]

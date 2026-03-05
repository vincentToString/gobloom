
# Stage1: Builder
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile
RUN CGO_ENABLED=0 GOOS=linux go build -o /bloombox-server ./cmd/main.go

# Stage2: Runner
FROM alpine:latest

WORKDIR /app

# Copy the 15MB compiled binary from the 'builder' stage, leaving the 500MB compiler behind
COPY --from=builder /bloombox-server ./bloombox-server

EXPOSE 50051
EXPOSE 8080
# When the container turns on, run this!
CMD ["./bloombox-server"]


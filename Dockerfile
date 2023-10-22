# Stage 1: Download dependencies
FROM golang:1.20-alpine AS dependencies

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Stage 2: Build the application
FROM golang:1.20-alpine AS build

WORKDIR /app

COPY . .

COPY --from=dependencies /app/go.mod /app/go.sum ./ 
RUN go build -o main .

# Stage 3: Create the final image
FROM alpine:latest

WORKDIR /app

COPY --from=build /app .

CMD ["./main"]
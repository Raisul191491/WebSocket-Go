# Build Stage
FROM golang:1.22 AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o server .

# Final Stage
FROM alpine:latest

WORKDIR /app

COPY --from=build /app/server .

EXPOSE 9000

CMD ["./server"]

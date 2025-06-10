FROM golang:1.24-alpine
WORKDIR /exercise-2
COPY . .
RUN go mod download
RUN go build -o exercise-2 ./cmd/main.go
EXPOSE 3030
CMD ["./exercise-2"]


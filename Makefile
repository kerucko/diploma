BINARY_NAME=server

build:
	go build -o $(BINARY_NAME) ./server/cmd

clean:
	go clean
	rm -f $(BINARY_NAME)

run-server: build
	./$(BINARY_NAME)


docker-up: docker-compose.yaml
	docker compose up -d

docker-down: docker-compose.yaml
	docker compose down


deps:
	go install github.com/pressly/goose/v3/cmd/goose@latest


migrate-up:
	goose -dir ./migrations postgres "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" up

migrate-down:
	goose -dir ./migrations postgres "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" down

migrate-reset:
	goose -dir ./migrations postgres "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" reset
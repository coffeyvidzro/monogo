.PHONY: test test-race vet build sqlc compose-validate

test:
	cd server && go test ./...

test-race:
	cd server && go test -race ./...

vet:
	cd server && go vet ./...

build:
	cd server && go build ./...

sqlc:
	cd server && sqlc generate

compose-validate:
	docker compose --env-file .env.example -f deploy/compose.yaml config --quiet

.PHONY: run build test lint tidy docker-up docker-down docker-reset migrate seed openapi-validate openapi-ui openapi-gen-ts

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api
	go build -o bin/migrate ./cmd/migrate
	go build -o bin/seed ./cmd/seed

test:
	go test ./...

tidy:
	go mod tidy

lint:
	go vet ./...

docker-up:
	docker compose -f deployments/docker-compose.yml up -d

docker-down:
	docker compose -f deployments/docker-compose.yml down

docker-reset:
	docker compose -f deployments/docker-compose.yml down -v

migrate:
	go run ./cmd/migrate

migrate-status:
	go run ./cmd/migrate -status

seed:
	go run ./cmd/seed

openapi-validate:
	docker run --rm -v "$(PWD)/api:/api" stoplight/spectral lint -F hint /api/openapi.yaml || \
	docker run --rm -v "$(PWD)/api:/api" openapitools/openapi-generator-cli validate -i /api/openapi.yaml

openapi-ui:
	@echo "Swagger UI: http://localhost:8081"
	docker run --rm -p 8081:8080 \
		-e SWAGGER_JSON=/spec/openapi.yaml \
		-v "$(PWD)/api/openapi.yaml:/spec/openapi.yaml" \
		swaggerapi/swagger-ui

openapi-gen-ts:
	mkdir -p clients/typescript
	docker run --rm -v "$(PWD):/local" openapitools/openapi-generator-cli generate \
		-i /local/api/openapi.yaml \
		-g typescript-axios \
		-o /local/clients/typescript \
		--additional-properties=supportsES6=true,npmName=tinh-tien-api-client
	@echo "Generated: clients/typescript/"

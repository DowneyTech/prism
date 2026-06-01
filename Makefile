.PHONY: up down logs migrate build

up:
	cp -n .env.example .env 2>/dev/null || true
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f backend

migrate:
	docker compose run --rm migrate

build:
	docker compose build

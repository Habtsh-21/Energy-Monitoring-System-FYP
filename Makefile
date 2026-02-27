.PHONY: up down build logs reset

up:
	docker compose up -d

down:
	docker compose down

build:
	docker compose up --build -d

logs:
	docker compose logs -f

reset:
	docker compose down -v
	docker compose up --build -d

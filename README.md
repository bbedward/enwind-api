# enwind-api

## DB

Create/mod entities in ent/schema

Re-generate code `make ent`

## DB Interactions

Contains exclusively in `internal/repositories`

Create one per entity type, `internal/repositories/user`

Re-generate interfaces after adding methods/changing signatures `make interfaces`

Add new ones to meta `internal/repositories/repositories` and re-gen interfaces again `make interfaces`

Kinda sucks but interfaces allow mocking.

## API Handlers
Created in `internal/api/handlers` - registered in `cmd/api/main.go

## Run

`go run ./cmd/api`
# DUSKFORGE API

REST backend for a social network dedicated to movies.

> This repository contains only the **API server** (Go + Gin). Web and mobile clients communicate with this backend.

## Documentation

- **Technical documentation**: see [`docs/TECHNICAL.md`](docs/TECHNICAL.md)
- **API reference (Swagger)**: `http://localhost:8080/docs/index.html` once the server is running

## Quick Start

```bash
cp .env.example .env
# Edit .env with your secrets

docker compose up --build
```

The API will be available at `http://localhost:8080` and the Swagger docs at `http://localhost:8080/docs/index.html`.

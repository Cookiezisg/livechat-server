# livechat-server

![Go](https://img.shields.io/badge/Go-1.21-00ADD8?style=flat-square&logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?style=flat-square&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis&logoColor=white)
![React](https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react&logoColor=black)
![Vite](https://img.shields.io/badge/Vite-5-646CFF?style=flat-square&logo=vite&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?style=flat-square&logo=typescript&logoColor=white)

A full-stack web chat app with Go, WebSocket, PostgreSQL, Redis, and a React/Vite frontend. Includes auth, rooms, DMs, message history, presence, typing indicators, and file uploads.

## Overview

This project is a complete web chat system built for learning and extension. It combines:

- A Go HTTP API for auth, rooms, users, history, and uploads
- A WebSocket gateway for real-time chat events
- PostgreSQL for persistent data
- Redis for future scaling (fan-out, presence, queues)
- A React/Vite client with a clean, responsive UI

## Features

- Real-time updates via WebSocket (rooms, DMs, typing, presence)
- REST APIs for auth, rooms, users, history, uploads
- Persistent storage with PostgreSQL
- Modern frontend with React + Vite + Zustand + React Query
- Clean modular backend for future extensions

## Tech Stack

- Backend: Go 1.21 + chi + pgx + JWT + WebSocket
- Database: PostgreSQL
- Cache/queue: Redis (wired in compose, ready to extend)
- Frontend: Vite + React + TypeScript + Zustand + React Query

## Architecture

- **REST API** handles registration/login, user search, room CRUD, and history queries.
- **WebSocket Hub** pushes new messages, typing indicators, and presence updates.
- **Store layer** isolates all Postgres reads/writes.
- **Frontend** uses Zustand for state, React Query for data fetches, and a WebSocket client for live updates.

## Data Model (PostgreSQL)

- `users`: account + profile fields
- `rooms`: public/private chat rooms
- `room_members`: membership + last read timestamps
- `messages`: room messages and DMs

## Quick Start

### 1) Start infrastructure

```bash
docker-compose up -d
```

### 2) Initialize database

```bash
psql "postgres://livechat:livechat@localhost:5432/livechat?sslmode=disable" -f db/migrations/001_init.sql
```

### 3) Start backend

```bash
go run .
```

Default address: `http://localhost:8080`

### 4) Start frontend

```bash
cd web
npm install
npm run dev
```

Open: `http://localhost:5173`

## WebSocket Protocol

Client -> Server:

- `{"type":"join_room","roomId":"..."}`
- `{"type":"leave_room","roomId":"..."}`
- `{"type":"message","roomId":"...","body":"...","attachmentUrl":"..."}` for rooms
- `{"type":"message","recipientId":"...","body":"...","attachmentUrl":"..."}` for DMs
- `{"type":"typing","roomId":"...","isTyping":true}`
- `{"type":"typing","recipientId":"...","isTyping":true}`
- `{"type":"read","roomId":"..."}`

Server -> Client:

- `presence`: online/offline status
- `typing`: typing indicator
- `message`: new message payload
- `room_joined`: confirmation of join

## API Snapshot

- `POST /api/auth/register` Register
- `POST /api/auth/login` Login
- `GET /api/auth/me` Current user
- `GET /api/rooms` List rooms
- `POST /api/rooms` Create room
- `POST /api/rooms/:id/join` Join room
- `GET /api/rooms/:id/messages` Room history
- `GET /api/dms/:userId/messages` DM history
- `POST /api/uploads` Upload file
- `GET /ws?token=` WebSocket connect

## Configuration

Environment variables:

- `HTTP_ADDR`: server address, default `:8080`
- `DATABASE_URL`: Postgres DSN
- `REDIS_URL`: Redis DSN (optional)
- `JWT_SECRET`: JWT secret
- `TOKEN_TTL`: token lifetime, default `24h`
- `CORS_ORIGINS`: frontend origin, default `http://localhost:5173`
- `UPLOAD_DIR`: upload directory, default `./uploads`
- `MAX_UPLOAD_MB`: upload size limit (MB), default `10`

## Local Development Tips

- Use `docker-compose up -d` to start Postgres/Redis locally.
- Run migrations with the provided SQL file in `db/migrations/001_init.sql`.
- For the web app, set `VITE_API_BASE` if your API is not on `http://localhost:8080`.

## Structure

- `main.go`: API + WS entrypoint
- `internal/http`: REST routes and handlers
- `internal/ws`: WebSocket hub
- `internal/store`: database access layer
- `db/migrations`: schema migrations
- `web/`: frontend app

## Deployment Notes

- Put the API behind an HTTPS reverse proxy (nginx/Caddy) for production.
- Configure `CORS_ORIGINS` to match your frontend host.
- Store uploads in a persistent volume or object storage for durability.

## Next Ideas

- Read receipts, mentions, search, archives
- Redis Pub/Sub for multi-instance WS
- File type validation, image previews, CDN

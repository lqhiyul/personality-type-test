# API Overview

All API responses are JSON unless the export endpoint is requested as CSV. Cookie-authenticated unsafe requests require the `X-CSRF-Token` header matching the `csrf_token` cookie.

## Public

- `GET /healthz`
- `POST /api/submit`
- `GET /api/users/{username}`
- `GET /api/users/{username}/comments`

## Auth

- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`

## Current User

- `GET /api/me/results`
- `POST /api/me/results/{id}/primary`
- `DELETE /api/me/results/{id}`
- `PATCH /api/me/profile`

## Social And Safety

- `POST /api/friends/request`
- `GET /api/friends`
- `DELETE /api/friends/{id}`
- `GET /api/friends/requests`
- `POST /api/friends/requests/{id}/accept`
- `POST /api/users/{username}/comments`
- `DELETE /api/profile-comments/{id}`
- `GET /api/messages/conversations`
- `POST /api/messages/start`
- `GET /api/messages/conversations/{id}`
- `POST /api/messages/conversations/{id}`
- `DELETE /api/messages/{id}`
- `GET /api/blocks`
- `POST /api/blocks`
- `DELETE /api/blocks/{username}`
- `POST /api/reports`

## Admin

- `POST /api/login`
- `POST /api/logout`
- `GET /api/results`
- `DELETE /api/results`
- `DELETE /api/results/{id}`
- `GET /api/results/export`
- `GET /api/stats`
- `GET /api/admin/reports`
- `POST /api/admin/reports/{id}/status`

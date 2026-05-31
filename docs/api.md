# API Documentation

All API responses are JSON unless `/api/results/export` returns CSV. Unsafe methods (`POST`, `PUT`, `PATCH`, `DELETE`, etc.) require `X-CSRF-Token` with the same value as the readable `csrf_token` cookie.

## Conventions

### Error Shape

The API uses this error shape:

```json
{
  "error": "authentication required"
}
```

Common status codes:

| Status | Meaning |
| --- | --- |
| `400` | Invalid JSON, invalid path/body value, or validation error. |
| `401` | Missing/invalid user or admin session. |
| `403` | CSRF failure or forbidden user action. |
| `404` | Resource not found. |
| `405` | Method not allowed. Response includes `Allow`. |
| `409` | Duplicate/conflicting state. |
| `429` | Login rate limit. Response includes `Retry-After`. |
| `500` | Unexpected server/storage error. |

### Auth And CSRF

| Requirement | Meaning |
| --- | --- |
| Public | No login required. |
| User session | Requires `mbti_user_session` HttpOnly cookie. |
| Admin session | Requires `mbti_admin_session` HttpOnly cookie. |
| CSRF | Unsafe request must include `X-CSRF-Token`. |

## Public

### `GET /healthz`

Health check.

- Auth: public
- CSRF: no
- Success `200 text/plain`: `ok`

### `POST /api/submit`

Submits weighted slider quiz answers. If the request has a valid user session, the result is also saved to that user account.

- Auth: public, optionally user session
- CSRF: yes

Request:

```json
{
  "name": "Yehor",
  "answers": [
    100, 100, 100, 100, 100, 100, 100,
    100, 100, 100, 100, 100, 100, 100,
    0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0
  ],
  "duration": 180
}
```

`answers` must contain exactly 28 integers. Each value distributes 100 points between the question's left and right poles: `0` means full left preference, `50` means balance, and `100` means full right preference. The backend owns the question-to-dimension mapping and calculates the weighted score.

Success `200`:

```json
{
  "type": "INTJ",
  "profile": {
    "type": "INTJ",
    "code": "INTJ",
    "dimensions": [
      {
        "key": "EI",
        "label": "Energy",
        "leftCode": "E",
        "leftLabel": "Extraversion",
        "leftScore": 0,
        "leftPercent": 0,
        "rightCode": "I",
        "rightLabel": "Introversion",
        "rightScore": 700,
        "rightPercent": 100,
        "winner": "I",
        "percent": 100,
        "margin": 100,
        "balanceLevel": "strong"
      }
    ]
  },
  "result": {
    "id": "b8e81824f40d",
    "name": "Yehor",
    "type": "INTJ",
    "answers": "100,100,100,100,100,100,100,100,100,100,100,100,100,100,0,0,0,0,0,0,0,0,0,0,0,0,0,0",
    "duration": 180,
    "created": "2026-05-28T10:00:00Z"
  },
  "savedToAccount": false
}
```

Common errors:

- `400 {"error":"invalid JSON request"}`
- `400 {"error":"name must be 1 to 64 characters"}`
- `400 {"error":"expected 28 answers"}`
- `400 {"error":"answer 1 must be between 0 and 100"}`
- `403 {"error":"csrf token is missing or invalid"}`
- `500 {"error":"could not save result"}`

### `GET /api/users/{username}`

Returns a public profile without private account fields.

- Auth: public, optionally user session for viewer friendship/block context
- CSRF: no

Success `200`:

```json
{
  "username": "demo-alice",
  "displayName": "Demo Alice",
  "avatarKey": "gradient-blue",
  "profileVisibility": "public",
  "isPrivate": false,
  "bio": "Seeded demo account for local development.",
  "primaryType": "INFJ",
  "completedTestsCount": 1,
  "showPrimaryResult": true,
  "showCompletedCount": true,
  "showFriends": true
}
```

Common errors:

- `404 {"error":"profile not found"}`
- `500 {"error":"could not load profile"}`

### `GET /api/users/{username}/comments`

Lists public profile comments.

- Auth: public
- CSRF: no

Success `200`:

```json
{
  "comments": [
    {
      "id": 12,
      "author": {
        "username": "demo-bob",
        "displayName": "Demo Bob",
        "avatarKey": "gradient-blue"
      },
      "body": "Great profile",
      "createdAt": "2026-05-28T10:00:00Z"
    }
  ]
}
```

Common errors:

- `403 {"error":"comments are hidden for private profiles"}`
- `404 {"error":"profile not found"}`

## User Auth

### `POST /api/auth/register`

Creates a user and starts a user session.

- Auth: public
- CSRF: yes

Request:

```json
{
  "username": "demo-alice",
  "email": "demo-alice@example.com",
  "password": "DemoPassword123"
}
```

Success `201`:

```json
{
  "id": 1,
  "username": "demo-alice",
  "email": "demo-alice@example.com",
  "displayName": "demo-alice",
  "bio": "",
  "avatarKey": "",
  "profileVisibility": "public",
  "showPrimaryResult": true,
  "showCompletedCount": true,
  "showFriends": true
}
```

Common errors:

- `400 {"error":"username must be 3 to 32 characters"}`
- `400 {"error":"email is not valid"}`
- `400 {"error":"password must be at least 8 characters"}`
- `409 {"error":"username or email is already registered"}`

### `POST /api/auth/login`

Starts a user session.

- Auth: public
- CSRF: yes

Request:

```json
{
  "emailOrUsername": "demo-alice",
  "password": "DemoPassword123"
}
```

Success `200`: same body shape as registration.

Common errors:

- `400 {"error":"invalid JSON request"}`
- `401 {"error":"invalid credentials"}`
- `429 {"error":"Too many login attempts. Try again later."}`

### `POST /api/auth/logout`

Revokes the current user session and clears the user cookie.

- Auth: user session if present
- CSRF: yes

Success `200`:

```json
{
  "ok": true
}
```

Common errors:

- `500 {"error":"could not log out"}`

### `GET /api/auth/me`

Returns the current authenticated user.

- Auth: user session
- CSRF: no

Success `200`:

```json
{
  "id": 1,
  "username": "demo-alice",
  "email": "demo-alice@example.com",
  "displayName": "Demo Alice",
  "bio": "Seeded demo account for local development.",
  "avatarKey": "gradient-blue",
  "profileVisibility": "public",
  "showPrimaryResult": true,
  "showCompletedCount": true,
  "showFriends": true,
  "createdAt": "2026-05-28T10:00:00Z"
}
```

Common errors:

- `401 {"error":"authentication required"}`

## Current User

### `PATCH /api/me/profile`

Updates the current user's profile/privacy settings.

- Auth: user session
- CSRF: yes

Request:

```json
{
  "displayName": "Demo Alice",
  "bio": "Backend-friendly profile",
  "avatarKey": "gradient-blue",
  "profileVisibility": "public",
  "showPrimaryResult": true,
  "showCompletedCount": true,
  "showFriends": true
}
```

Success `200`: current user response, same shape as `/api/auth/me`.

Common errors:

- `400 {"error":"display name must be 1 to 64 characters"}`
- `400 {"error":"bio must be 280 characters or fewer"}`
- `401 {"error":"authentication required"}`

### `GET /api/me/results`

Lists saved quiz results for the current user.

- Auth: user session
- CSRF: no

Success `200`:

```json
{
  "results": [
    {
      "id": 1,
      "mbtiType": "INTJ",
      "durationSeconds": 180,
      "isPrimary": true,
      "createdAt": "2026-05-28T10:00:00Z"
    }
  ]
}
```

Common errors:

- `401 {"error":"authentication required"}`
- `500 {"error":"could not load saved results"}`

### `POST /api/me/results/{id}/primary`

Sets one of the current user's saved results as primary.

- Auth: user session
- CSRF: yes
- Request body: none

Success `200`: a saved result object.

Common errors:

- `401 {"error":"authentication required"}`
- `404 {"error":"result not found"}`
- `500 {"error":"could not set primary result"}`

### `DELETE /api/me/results/{id}`

Deletes one saved result owned by the current user.

- Auth: user session
- CSRF: yes

Success `200`:

```json
{
  "ok": true
}
```

Common errors:

- `401 {"error":"authentication required"}`
- `404 {"error":"result not found"}`
- `500 {"error":"could not delete result"}`

## Friends And Compatibility

### `POST /api/friends/request`

Creates a friend request.

- Auth: user session
- CSRF: yes

Request:

```json
{
  "username": "demo-bob"
}
```

Success `201`:

```json
{
  "id": 10,
  "requesterId": 1,
  "addresseeId": 2,
  "status": "pending",
  "createdAt": "2026-05-28T10:00:00Z",
  "updatedAt": "2026-05-28T10:00:00Z"
}
```

Common errors:

- `400 {"error":"username is required"}`
- `400 {"error":"cannot send a friend request to yourself"}`
- `403 {"error":"cannot interact with blocked user"}`
- `404 {"error":"target user not found"}`
- `409 {"error":"friend request or friendship already exists"}`

### `GET /api/friends`

Lists accepted friends. Compatibility is included in each friend item when MBTI types are available.

- Auth: user session
- CSRF: no

Success `200`:

```json
{
  "friends": [
    {
      "friendshipId": 10,
      "id": 2,
      "username": "demo-bob",
      "displayName": "Demo Bob",
      "avatarKey": "gradient-blue",
      "primaryType": "ENTP",
      "compatibility": {
        "available": true,
        "friendship": 71,
        "relationship": 64,
        "work": 73
      }
    }
  ]
}
```

Common errors:

- `401 {"error":"authentication required"}`
- `500 {"error":"could not load friends"}`

### `GET /api/friends/requests`

Lists incoming pending friend requests.

- Auth: user session
- CSRF: no

Success `200`:

```json
{
  "requests": [
    {
      "id": 10,
      "status": "pending",
      "requester": {
        "id": 1,
        "username": "demo-alice",
        "displayName": "Demo Alice",
        "avatarKey": "gradient-blue",
        "primaryType": "INFJ"
      },
      "createdAt": "2026-05-28T10:00:00Z"
    }
  ]
}
```

### `POST /api/friends/requests/{id}/accept`

Accepts an incoming friend request.

- Auth: user session
- CSRF: yes

Success `200`: friendship object.

Common errors:

- `403 {"error":"friendship action is not allowed"}`
- `404 {"error":"friendship not found"}`
- `409 {"error":"friend request is not pending"}`

There is no separate reject endpoint today. Removing/ignoring rejected requests is a roadmap item.

### `DELETE /api/friends/{id}`

Removes an accepted friendship.

- Auth: user session
- CSRF: yes

Success `200`:

```json
{
  "ok": true
}
```

Common errors:

- `403 {"error":"friendship action is not allowed"}`
- `404 {"error":"friendship not found"}`

## Comments

### `POST /api/users/{username}/comments`

Creates a comment on a public profile.

- Auth: user session
- CSRF: yes

Request:

```json
{
  "body": "Great profile"
}
```

Success `201`: profile comment object.

Common errors:

- `400 {"error":"comment cannot be empty"}`
- `400 {"error":"comment is too long"}`
- `403 {"error":"cannot interact with blocked user"}`
- `403 {"error":"comments are hidden for private profiles"}`

### `DELETE /api/profile-comments/{id}`

Deletes a comment when allowed by ownership rules.

- Auth: user session
- CSRF: yes

Success `200`:

```json
{
  "ok": true
}
```

Common errors:

- `403 {"error":"comment action is not allowed"}`
- `404 {"error":"comment not found"}`

## Messages

### `GET /api/messages/conversations`

Lists current user's conversations.

- Auth: user session
- CSRF: no

Success `200`:

```json
{
  "conversations": [
    {
      "id": 1,
      "participants": [],
      "blocked": false,
      "createdAt": "2026-05-28T10:00:00Z",
      "updatedAt": "2026-05-28T10:00:00Z"
    }
  ]
}
```

### `POST /api/messages/start`

Starts or returns a one-to-one conversation with another user.

- Auth: user session
- CSRF: yes

Request:

```json
{
  "username": "demo-bob"
}
```

Success `200`: conversation object.

Common errors:

- `400 {"error":"cannot start a conversation with yourself"}`
- `403 {"error":"cannot interact with blocked user"}`
- `404 {"error":"target user not found"}`

### `GET /api/messages/conversations/{id}`

Returns a conversation and recent messages.

- Auth: user session
- CSRF: no

Success `200`:

```json
{
  "conversation": {
    "id": 1,
    "participants": [],
    "blocked": false,
    "createdAt": "2026-05-28T10:00:00Z",
    "updatedAt": "2026-05-28T10:00:00Z"
  },
  "messages": []
}
```

### `POST /api/messages/conversations/{id}`

Sends a message.

- Auth: user session
- CSRF: yes

Request:

```json
{
  "body": "Hello"
}
```

Success `201`:

```json
{
  "id": 4,
  "conversationId": 1,
  "senderId": 1,
  "body": "Hello",
  "createdAt": "2026-05-28T10:00:00Z"
}
```

Common errors:

- `400 {"error":"message cannot be empty"}`
- `400 {"error":"message is too long"}`
- `403 {"error":"cannot interact with blocked user"}`
- `403 {"error":"message action is not allowed"}`
- `404 {"error":"message not found"}`

### `DELETE /api/messages/{id}`

Deletes a message when allowed by ownership rules.

- Auth: user session
- CSRF: yes

Success `200`:

```json
{
  "ok": true
}
```

## Blocks And Reports

### `GET /api/blocks`

Lists users blocked by the current user.

- Auth: user session
- CSRF: no

Success `200`:

```json
{
  "blocks": [
    {
      "id": 2,
      "username": "demo-bob",
      "displayName": "Demo Bob",
      "avatarKey": "gradient-blue",
      "createdAt": "2026-05-28T10:00:00Z"
    }
  ]
}
```

### `POST /api/blocks`

Blocks another user. Blocking prevents friend requests, comments, and messages through the implemented flows.

- Auth: user session
- CSRF: yes

Request:

```json
{
  "username": "demo-bob"
}
```

Success `200`: blocked user object.

Common errors:

- `400 {"error":"cannot block yourself"}`
- `404 {"error":"target user not found"}`

### `DELETE /api/blocks/{username}`

Unblocks a user.

- Auth: user session
- CSRF: yes

Success `200`:

```json
{
  "ok": true
}
```

### `POST /api/reports`

Creates a safety report for a profile, comment, or message.

- Auth: user session
- CSRF: yes

Profile report request:

```json
{
  "targetType": "profile",
  "username": "demo-bob",
  "reason": "harassment",
  "details": "Optional details"
}
```

Comment/message report request:

```json
{
  "targetType": "message",
  "targetId": 4,
  "reason": "harassment"
}
```

Success `201`:

```json
{
  "id": 3,
  "targetType": "message",
  "targetId": 4,
  "status": "open",
  "createdAt": "2026-05-28T10:00:00Z"
}
```

Common errors:

- `400 {"error":"report target type is invalid"}`
- `400 {"error":"report reason is required"}`
- `403 {"error":"message is not available"}`
- `404 {"error":"comment not found"}`

## Admin

Admin endpoints use a separate admin session cookie and are intentionally simple for this portfolio project.

### `POST /api/login`

Starts an admin session.

- Auth: public
- CSRF: yes

Request:

```json
{
  "password": "change-me"
}
```

Success `200`:

```json
{
  "ok": true
}
```

Common errors:

- `400 {"error":"invalid JSON request"}`
- `401 {"error":"invalid admin password"}`
- `429 {"error":"Too many login attempts. Try again later."}`

Audit actions: `admin_login_invalid_json`, `admin_login_failure`, `admin_login_rate_limited`, `admin_login_success`.

### `POST /api/logout`

Revokes the admin session.

- Auth: admin session if present
- CSRF: yes

Success `200`:

```json
{
  "ok": true
}
```

Audit action: `admin_logout`.

### `GET /api/results`

Lists anonymous quiz submissions.

- Auth: admin session
- CSRF: no

Success `200`:

```json
{
  "results": [
    {
      "id": "b8e81824f40d",
      "name": "Yehor",
      "type": "INTJ",
      "answers": "100,100,100,100,100,100,100,100,100,100,100,100,100,100,0,0,0,0,0,0,0,0,0,0,0,0,0,0",
      "duration": 180,
      "created": "2026-05-28T10:00:00Z"
    }
  ]
}
```

Common errors:

- `401 {"error":"admin authentication required"}`

### `DELETE /api/results`

Clears all anonymous quiz submissions.

- Auth: admin session
- CSRF: yes

Success `200`:

```json
{
  "ok": true
}
```

Audit action: `admin_clear_results`.

### `DELETE /api/results/{id}`

Deletes one anonymous quiz submission.

- Auth: admin session
- CSRF: yes

Success `200`:

```json
{
  "ok": true
}
```

Audit action: `admin_delete_result`.

### `GET /api/results/export`

Exports anonymous quiz submissions. CSV is the default. `?format=json` returns JSON.

- Auth: admin session
- CSRF: no

CSV success `200 text/csv`: attachment `mbti-results.csv`.

JSON success `200`:

```json
{
  "results": [],
  "generatedAt": "2026-05-28T10:00:00Z"
}
```

Audit action: `admin_export_results`.

### `GET /api/stats`

Returns aggregate stats for anonymous quiz submissions.

- Auth: admin session
- CSRF: no

Success `200`:

```json
{
  "total": 10,
  "averageDurationSeconds": 140,
  "byType": {
    "INTJ": 3
  },
  "topTypes": [
    {
      "type": "INTJ",
      "count": 3
    }
  ],
  "axisDistribution": {
    "E": 4,
    "I": 6,
    "S": 5,
    "N": 5,
    "T": 7,
    "F": 3,
    "J": 8,
    "P": 2
  }
}
```

### `GET /api/admin/reports`

Lists safety reports for admin review.

- Auth: admin session
- CSRF: no
- Query: optional `status`, optional `limit`

Success `200`:

```json
{
  "reports": [
    {
      "id": 3,
      "targetType": "message",
      "targetId": 4,
      "reason": "harassment",
      "details": "Optional details",
      "status": "open",
      "createdAt": "2026-05-28T10:00:00Z"
    }
  ]
}
```

### `POST /api/admin/reports/{id}/status`

Updates a safety report status.

- Auth: admin session
- CSRF: yes

Request:

```json
{
  "status": "reviewed"
}
```

Allowed statuses: `open`, `reviewed`, `dismissed`.

Success `200`: admin report object.

Common errors:

- `400 {"error":"report status is invalid"}`
- `404 {"error":"report not found"}`

Audit action: `admin_update_report_status`.

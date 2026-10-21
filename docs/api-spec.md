# URL Shortener API Specification

## 1. Overview

This document specifies the RESTful API contracts, request/response models, validation logic, HTTP status codes, and operational behavior for the Go URL Shortener service.

---

## 2. Global Conventions

- **Content-Type:** `application/json; charset=utf-8` for all REST endpoints.
- **Timestamp Format:** ISO 8601 / RFC 3339 UTC format (e.g., `2026-10-21T07:16:00Z`).
- **Correlation ID:** Every request receives or carries an `X-Request-ID` header, returned in all responses.
- **Redirect Semantics:** Temporary redirects use **HTTP 307 Temporary Redirect** with `Cache-Control: no-cache, no-store, must-revalidate` to ensure every click reaches the server for accurate analytics while preserving the request method.
- **Short Code Format:** Base62 character set (`[0-9a-zA-Z]`), default length 7 characters (approx. 3.5 trillion unique combinations).
- **Reserved Short Codes:** `api`, `healthz`, `readyz`, `docs`, `metrics`, `favicon.ico`, `static`.

---

## 3. Error Response Format

All error responses adhere to a consistent JSON structure:

```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "The provided URL is malformed or uses an unsupported scheme",
    "details": [
      {
        "field": "url",
        "issue": "must start with http:// or https://"
      }
    ],
    "request_id": "c1f7b6a4-8f24-469b-81f2-1b12b2345678"
  }
}
```

### Standard Error Codes

| Code | HTTP Status | Description |
|---|---|---|
| `INVALID_INPUT` | 400 Bad Request | Request body contains malformed JSON or invalid field values |
| `CONFLICT` | 409 Conflict | Custom code or requested resource already exists |
| `NOT_FOUND` | 404 Not Found | Requested short code does not exist or has been deleted |
| `RATE_LIMITED` | 429 Too Many Requests | Client exceeded per-IP rate limit |
| `INTERNAL_ERROR` | 500 Internal Server Error | Unexpected server or database failure |
| `SERVICE_UNAVAILABLE` | 503 Service Unavailable | Readiness probe failed (e.g. database unreachable) |

---

## 4. Endpoints

### 4.1 Create Short URL
- **Method:** `POST`
- **Path:** `/api/v1/urls`
- **Request Headers:** `Content-Type: application/json`
- **Request Body:**
```json
{
  "url": "https://example.com/very/long/path?param=value",
  "custom_code": "my-promo" 
}
```
*Note:* `custom_code` is optional. If omitted, a collision-resistant 7-character Base62 code is automatically generated.
- **Validation Rules:**
  - `url`: Required, max 2048 characters, must have scheme `http` or `https`, valid host format, safe from local file injection (`file:`, `javascript:`, `data:`).
  - `custom_code`: Optional, 4–32 characters, regex `^[a-zA-Z0-9_-]+$`, must not be in reserved list.
- **Success Response:** `201 Created`
  - `Location: /api/v1/urls/{code}`
```json
{
  "id": "e6a2b8e8-3a9d-426b-bb9b-b53e7f6f1a23",
  "code": "my-promo",
  "original_url": "https://example.com/very/long/path?param=value",
  "short_url": "http://localhost:8080/my-promo",
  "created_at": "2026-10-21T07:16:00Z",
  "updated_at": "2026-10-21T07:16:00Z"
}
```
- **Error Responses:** `400 Bad Request`, `409 Conflict` (if custom code is taken), `429 Too Many Requests`.

---

### 4.2 List URLs (Paginated)
- **Method:** `GET`
- **Path:** `/api/v1/urls`
- **Query Parameters:**
  - `limit` (integer, optional, default: 20, min: 1, max: 100)
  - `offset` (integer, optional, default: 0, min: 0)
- **Success Response:** `200 OK`
```json
{
  "data": [
    {
      "id": "e6a2b8e8-3a9d-426b-bb9b-b53e7f6f1a23",
      "code": "my-promo",
      "original_url": "https://example.com/very/long/path?param=value",
      "short_url": "http://localhost:8080/my-promo",
      "clicks": 142,
      "last_accessed_at": "2026-10-21T08:00:00Z",
      "created_at": "2026-10-21T07:16:00Z",
      "updated_at": "2026-10-21T07:16:00Z"
    }
  ],
  "pagination": {
    "limit": 20,
    "offset": 0,
    "total": 1
  }
}
```

---

### 4.3 Get URL Details
- **Method:** `GET`
- **Path:** `/api/v1/urls/{code}`
- **Success Response:** `200 OK`
```json
{
  "id": "e6a2b8e8-3a9d-426b-bb9b-b53e7f6f1a23",
  "code": "my-promo",
  "original_url": "https://example.com/very/long/path?param=value",
  "short_url": "http://localhost:8080/my-promo",
  "clicks": 142,
  "last_accessed_at": "2026-10-21T08:00:00Z",
  "created_at": "2026-10-21T07:16:00Z",
  "updated_at": "2026-10-21T07:16:00Z"
}
```
- **Error Response:** `404 Not Found` if short code does not exist.

---

### 4.4 Update Target URL
- **Method:** `PATCH`
- **Path:** `/api/v1/urls/{code}`
- **Request Body:**
```json
{
  "url": "https://example.com/updated/destination"
}
```
- **Validation:** Same URL validation as in creation.
- **Success Response:** `200 OK` (returns updated URL model).
- **Error Responses:** `400 Bad Request`, `404 Not Found`.

---

### 4.5 Delete URL
- **Method:** `DELETE`
- **Path:** `/api/v1/urls/{code}`
- **Success Response:** `204 No Content` (empty response body).
- **Error Response:** `404 Not Found`.

---

### 4.6 Get Analytics / Stats
- **Method:** `GET`
- **Path:** `/api/v1/urls/{code}/stats`
- **Success Response:** `200 OK`
```json
{
  "code": "my-promo",
  "original_url": "https://example.com/very/long/path?param=value",
  "clicks": 142,
  "last_accessed_at": "2026-10-21T08:00:00Z",
  "created_at": "2026-10-21T07:16:00Z"
}
```
- **Error Response:** `404 Not Found`.

---

### 4.7 Redirect to Destination
- **Method:** `GET`
- **Path:** `/{code}`
- **Success Response:** `307 Temporary Redirect`
  - `Location: https://example.com/very/long/path?param=value`
  - `Cache-Control: no-cache, no-store, must-revalidate`
- **Side Effect:** Atomic increment of `clicks = clicks + 1` and update of `last_accessed_at = NOW()`.
- **Error Response:** `404 Not Found` with problem JSON if code is unknown.

---

### 4.8 Liveness Check
- **Method:** `GET`
- **Path:** `/healthz`
- **Success Response:** `200 OK`
```json
{
  "status": "ok",
  "time": "2026-10-21T07:16:00Z"
}
```

---

### 4.9 Readiness Check
- **Method:** `GET`
- **Path:** `/readyz`
- **Success Response:** `200 OK` (when database ping succeeds)
```json
{
  "status": "ready",
  "database": "connected",
  "time": "2026-10-21T07:16:00Z"
}
```
- **Failure Response:** `503 Service Unavailable` (when database ping fails)
```json
{
  "status": "not ready",
  "database": "disconnected",
  "error": "connection refused",
  "time": "2026-10-21T07:16:00Z"
}
```

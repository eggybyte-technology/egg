# egg/httpx

## Overview

`httpx` provides thin HTTP utilities for binding/validation, standard JSON error responses, security headers, and CORS middleware. It keeps transport adapters minimal and consistent.

## Key Features

- Bind and validate JSON requests (validator integration)
- Standard JSON error responses (404/405/custom)
- Security headers middleware with sensible defaults
- CORS middleware with configurable options

## Dependencies

Layer: L2 (Capabilities)

Depends on: `core`, `validator`

## Installation

```bash
go get github.com/eggybyte-technology/egg/httpx@latest
```

## Basic Usage

```go
import (
    "net/http"
    "github.com/eggybyte-technology/egg/httpx"
)

type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2"`
    Email string `json:"email" validate:"required,email"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := httpx.BindAndValidate(r, &req); err != nil {
        httpx.WriteError(w, err, http.StatusBadRequest)
        return
    }
    httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
```

### Security headers

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/users", handler)

secure := httpx.SecureMiddleware(httpx.DefaultSecurityHeaders())
srv := secure(mux)
_ = srv
```

### CORS

```go
corsOpts := httpx.CORSOptions{
    AllowedOrigins: []string{"https://example.com"},
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders: []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge: 3600,
}

handler := httpx.CORSMiddleware(corsOpts)(mux)
_ = handler
```

## Stability

Stable since v0.1.0.

## License

This package is part of the EggyByte framework and is licensed under the MIT License. See the root LICENSE file for details.

 


# ETag Caching for Static Content

This package provides ETag-based caching for static content in the Passenger Go application.

## Features

- **ETag Generation**: Creates ETags based on file modification time and size
- **Conditional Requests**: Handles `If-None-Match` headers to return `304 Not Modified`
- **File Type Aware Caching**: Different cache strategies for different file types
- **Selective Application**: Only applies to `/static/*` paths

## Cache Strategy

| File Type | Cache Duration | Cache Control |
|-----------|----------------|---------------|
| CSS/JS    | 1 year        | `public, max-age=31536000, immutable` |
| Images    | 1 year        | `public, max-age=31536000, immutable` |
| Fonts     | 1 year        | `public, max-age=31536000, immutable` |
| JSON      | 1 hour        | `public, max-age=3600` |
| Others    | 1 hour        | `public, max-age=3600` |

## Usage

The middleware is automatically applied to all static file requests in the frontend controller:

```go
// Apply ETag middleware to static routes
router.Use(cache.StaticETagMiddleware("frontend/static"))
```

## How It Works

1. **First Request**: Browser requests `/static/css/app.css`
   - Server returns file with ETag: `"a1b2c3d4"`
   - Cache-Control: `public, max-age=31536000, immutable`

2. **Subsequent Request**: Browser sends `If-None-Match: "a1b2c3d4"`
   - If file unchanged: Returns `304 Not Modified` (no file transfer)
   - If file changed: Returns new file with new ETag

## Benefits

- **Reduced Bandwidth**: No unnecessary downloads of unchanged files
- **Faster Load Times**: Especially for returning users
- **Better User Experience**: Faster page loads
- **Server Efficiency**: Less I/O for static file serving

## Testing

Run the tests with:

```bash
go test ./frontend/utilities/cache/...
```

Or use the test script:

```bash
./test_etag.sh
```

## Security

- Only applies to static content (`/static/*` paths)
- Does not cache sensitive data or API responses
- ETags are based on file metadata, not content hashes

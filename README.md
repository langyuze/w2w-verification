# w2w-verification

HTTP server for storing and retrieving verification request data. Submit a request payload containing an encrypted credential request, receive a UUID. Share the URL with the encryption key in the fragment to let users present their digital credentials via the Digital Credentials API.

## API

### Store data

```
POST /verify
Body: <request payload>
```

Response (`200 OK`, `application/json`):
```json
{
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://demo.verifiedbygoogle.com/getVerificationRequest?requestId=550e8400-e29b-41d4-a716-446655440000"
}
```

### Retrieve data (API)

```
GET /getVerificationRequest?requestId={uuid}
Accept: application/octet-stream
```

Response (`200 OK`, `application/octet-stream`): raw blob bytes.

### Retrieve data (Browser)

```
GET /getVerificationRequest?requestId={uuid}#<base64url-encoded-AES-key>
Accept: text/html
```

When opened in a browser, serves an HTML page that:
1. Reads the AES-256-GCM key from the URL fragment (never sent to server)
2. Fetches the encrypted payload from the server
3. Decrypts it (IV = first 12 bytes, remainder = ciphertext + auth tag)
4. Extracts `dcRequest` and repackages it under `digital`
5. Shows a "Verify" button; on click calls `navigator.credentials.get()`
6. Persists the credential response back to the server

### Get verification response

```
GET /getVerificationResponse?requestId={uuid}
```

Returns the credential response payload if available, or empty string if not yet submitted.

### Set verification response

```
POST /setVerificationResponse?requestId={uuid}
Body: <credential response JSON>
```

Stores the credential response for the given request ID.

## Build & Run

```bash
go build -o w2w-verification .
./w2w-verification -addr :8080 -db w2w.db
```

### Flags

| Flag        | Default                              | Description                          |
|-------------|--------------------------------------|--------------------------------------|
| `-addr`     | `:8080`                              | Listen address                       |
| `-db`       | `w2w.db`                             | SQLite database path                 |
| `-base-url` | `https://demo.verifiedbygoogle.com`  | Public base URL for retrieval links  |

## Examples

```bash
# Store
curl -X POST -d '{"encrypted":"payload"}' http://localhost:8080/verify
# {"requestId":"550e8400-...","url":"https://demo.verifiedbygoogle.com/getVerificationRequest?requestId=550e8400-..."}

# Retrieve raw payload
curl "http://localhost:8080/getVerificationRequest?requestId=550e8400-..."

# Check verification response
curl "http://localhost:8080/getVerificationResponse?requestId=550e8400-..."
```

## Run Tests

```bash
go test ./...
```

## Docker

```bash
docker build -t w2w-verification .
docker run -p 8080:8080 -v ./data:/data w2w-verification
```

## Deployment & Domain Wiring

### Cloud Container Service

Build and push the image to your container registry, then deploy to Cloud Run, ECS/Fargate, or similar. The container listens on port 8080 and stores data in SQLite at `/data/w2w.db` (mount a persistent volume there).

### Domain Setup

1. **DNS**: Point your domain (A record or CNAME) to the cloud service endpoint.
2. **HTTPS via reverse proxy** (Nginx + Let's Encrypt):
   ```nginx
   server {
       server_name yourdomain.com;
       location / {
           proxy_pass http://127.0.0.1:8080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```
   Then run: `certbot --nginx -d yourdomain.com`
3. **Cloud-native**: Most container services (Cloud Run, App Runner) provide built-in HTTPS — just map your custom domain in the console.

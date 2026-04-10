# w2w-verification

HTTP server for storing and retrieving verification request data. Submit a blob of data, receive a UUID. Use the UUID to retrieve the data later.

## API

### Store data

```
GET /verify?request={url_encoded_data}
```

Response (`200 OK`, `application/json`):
```json
{
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://demo.verifiedbygoogle.com/getVerificationRequest?requestId=550e8400-e29b-41d4-a716-446655440000"
}
```

### Retrieve data

```
GET /getVerificationRequest?requestId={uuid}
```

Response (`200 OK`, `application/octet-stream`): raw blob bytes.

Returns `404` if the UUID is not found, `400` if the UUID format is invalid.

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
curl -G --data-urlencode "request=hello world" http://localhost:8080/verify
# {"requestId":"550e8400-e29b-41d4-a716-446655440000","url":"https://demo.verifiedbygoogle.com/getVerificationRequest?requestId=550e8400-e29b-41d4-a716-446655440000"}

# Retrieve
curl "http://localhost:8080/getVerificationRequest?requestId=550e8400-e29b-41d4-a716-446655440000"
# hello world
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

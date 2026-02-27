# Service Monitor Frontend

This is the public-facing frontend for the Service Monitor system, hosted on Cloudflare Pages.

## Files

- `index.html` - Main status page
- `style.css` - Styling
- `app.js` - Frontend logic (polls status every 10 seconds)
- `functions/api/update.js` - Cloudflare Function to receive updates from backend
- `functions/api/status.js` - Cloudflare Function to serve status to frontend

## Deployment to Cloudflare Pages

### Prerequisites

1. Cloudflare account
2. Domain configured in Cloudflare (status.triplepoint.me)

### Steps

1. **Create KV Namespace**
   ```bash
   # Install Wrangler CLI
   npm install -g wrangler

   # Login to Cloudflare
   wrangler login

   # Create KV namespace
   wrangler kv:namespace create STATUS_STORE --preview=false
   ```

   This will output a namespace ID like: `abcd1234efgh5678`

2. **Generate API Key**
   ```bash
   # On Linux/Mac
   openssl rand -hex 32

   # Or use any secure random generator
   ```

   Save this key - you'll need it for both Cloudflare and the backend.

3. **Deploy to Cloudflare Pages**

   **Option A: Using Wrangler CLI**
   ```bash
   cd frontend
   wrangler pages deploy . --project-name=service-monitor
   ```

   **Option B: Using Cloudflare Dashboard**
   - Go to Cloudflare Dashboard → Pages
   - Click "Create a project"
   - Connect your Git repository (or direct upload)
   - Set build settings:
     - Build command: (leave empty)
     - Build output directory: /
   - Deploy

4. **Configure Environment Variables**

   In Cloudflare Pages dashboard:
   - Go to Settings → Environment variables
   - Add these variables:
     - `API_KEY`: Your generated 64-character key

   For KV binding:
   - Go to Settings → Functions → KV namespace bindings
   - Add binding:
     - Variable name: `STATUS_STORE`
     - KV namespace: Select the namespace you created

5. **Configure Custom Domain**
   - In Cloudflare Pages → Custom domains
   - Add: `status.triplepoint.me`
   - Cloudflare will automatically handle DNS

6. **Update Backend Configuration**

   In your backend `config.json`:
   ```json
   {
     "frontend_url": "https://status.triplepoint.me/api/update",
     "api_key": "YOUR_64_CHAR_API_KEY"
   }
   ```

## Testing

1. Deploy frontend to Cloudflare Pages
2. Test status endpoint:
   ```bash
   curl https://status.triplepoint.me/api/status
   ```

3. Test update endpoint:
   ```bash
   curl -X POST https://status.triplepoint.me/api/update \
     -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "services": [
         {
           "id": 1,
           "name": "Test Service",
           "ip_address": "192.168.1.1",
           "status": "up",
           "response_time_ms": 10
         }
       ],
       "last_update_time": "'$(date -u +%Y-%m-%dT%H:%M:%S.000Z)'"
     }'
   ```

4. Open `https://status.triplepoint.me` in browser

## Troubleshooting

- **Functions not working**: Ensure KV namespace binding is set correctly
- **Unauthorized errors**: Check API_KEY matches between backend and Cloudflare
- **CORS errors**: Cloudflare Functions automatically handle CORS for /api/* routes
- **Status not updating**: Check backend logs for push errors

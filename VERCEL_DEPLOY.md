# Deploying to Vercel - Fixed Instructions

## The Issue

Vercel was showing "404: NOT_FOUND" because it didn't know where to find the frontend files.

## Solution

The project structure has been updated with proper configuration.

---

## Quick Deploy to Vercel

### Step 1: Push to GitHub (Already Done ✅)

```bash
# Already done, but if you need to update:
git add .
git commit -m "Update Vercel configuration"
git push origin main
```

### Step 2: Deploy on Vercel

1. **Go to [vercel.com](https://vercel.com)**
2. **Sign in with GitHub**
3. **Click "Add New..." → "Project"**
4. **Import your repository:** `vishesh711/Inference-Gateway`

### Step 3: Configure Build Settings

**Important:** Set these settings:

```
Framework Preset: Next.js
Root Directory: frontend
Build Command: npm run build
Output Directory: .next
Install Command: npm install
```

### Step 4: Environment Variables

Add this environment variable:

```
Name: NEXT_PUBLIC_API_URL
Value: http://localhost:8000
```

(Change this to your production backend URL when you have it)

### Step 5: Deploy

Click **"Deploy"**

Vercel will:
- Install dependencies
- Build the Next.js app
- Deploy to a URL like: `your-project.vercel.app`

---

## Expected Result

After deployment, you should see:
- ✅ Chat interface loads
- ⚠️ Backend connection will fail (you need to deploy backend separately)

**Next:** Deploy backend to get full functionality.

---

## Option 1: Deploy Backend Locally (For Testing)

If you just want to test the UI on Vercel with your local backend:

1. **Start your local backend:**
   ```bash
   ./bin/gateway
   ```

2. **Use ngrok to expose it:**
   ```bash
   # Install ngrok
   brew install ngrok  # macOS
   
   # Expose port 8000
   ngrok http 8000
   ```

3. **Update Vercel environment variable:**
   - Go to your Vercel project settings
   - Update `NEXT_PUBLIC_API_URL` to your ngrok URL
   - Example: `https://abc123.ngrok.io`
   - Redeploy

Now your Vercel frontend will talk to your local backend!

---

## Option 2: Deploy Backend to Production

For a proper production setup, deploy the backend to a server:

### A. Deploy to DigitalOcean (Recommended)

1. **Create a Droplet** (Ubuntu 22.04, $12/month CPU-only)

2. **SSH into server:**
   ```bash
   ssh root@your-server-ip
   ```

3. **Install Go:**
   ```bash
   wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
   export PATH=$PATH:/usr/local/go/bin
   ```

4. **Clone your repo:**
   ```bash
   git clone https://github.com/vishesh711/Inference-Gateway.git
   cd Inference-Gateway
   ```

5. **Build gateway:**
   ```bash
   go build -o gateway cmd/gateway/main.go
   ```

6. **Install and run llama.cpp:**
   ```bash
   git clone https://github.com/ggerganov/llama.cpp
   cd llama.cpp && make
   # Download model and run server
   ```

7. **Run gateway with systemd:**
   
   Create `/etc/systemd/system/gateway.service`:
   ```ini
   [Unit]
   Description=Inference Gateway
   After=network.target

   [Service]
   Type=simple
   User=root
   WorkingDirectory=/root/Inference-Gateway
   ExecStart=/root/Inference-Gateway/gateway
   Restart=always

   [Install]
   WantedBy=multi-user.target
   ```

   ```bash
   sudo systemctl enable gateway
   sudo systemctl start gateway
   ```

8. **Setup Nginx (optional but recommended):**
   ```bash
   sudo apt install nginx
   ```

   Create `/etc/nginx/sites-available/gateway`:
   ```nginx
   server {
       listen 80;
       server_name your-domain.com;

       location / {
           proxy_pass http://localhost:8000;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection 'upgrade';
           proxy_set_header Host $host;
           proxy_cache_bypass $http_upgrade;
       }
   }
   ```

   ```bash
   sudo ln -s /etc/nginx/sites-available/gateway /etc/nginx/sites-enabled/
   sudo nginx -t
   sudo systemctl restart nginx
   ```

9. **Setup HTTPS with Let's Encrypt:**
   ```bash
   sudo apt install certbot python3-certbot-nginx
   sudo certbot --nginx -d your-domain.com
   ```

10. **Update Vercel:**
    - Go to your Vercel project settings
    - Update `NEXT_PUBLIC_API_URL` to: `https://your-domain.com`
    - Redeploy

---

## Option 3: Simple Static Demo (No Backend)

If you just want to show the UI without backend functionality:

1. **Update frontend to handle missing backend gracefully**

2. **Add demo mode** that shows the interface with mock data

3. **Deploy to Vercel** - UI will work, but API calls will fail gracefully

This is useful for portfolio/demos where you just want to show the interface design.

---

## Troubleshooting Vercel Deployment

### "404: NOT_FOUND" Error

**Problem:** Vercel can't find the app

**Solution:**
1. Make sure `Root Directory` is set to `frontend`
2. Check `Framework Preset` is `Next.js`
3. Verify files are in `frontend/` directory

### "Build Failed" Error

**Problem:** Build process failed

**Solution:**
1. Check build logs in Vercel dashboard
2. Make sure `package.json` is in `frontend/` directory
3. Try building locally first: `cd frontend && npm run build`

### "API Fetch Failed" Error

**Problem:** Frontend can't reach backend

**Solution:**
1. Check `NEXT_PUBLIC_API_URL` environment variable
2. Make sure backend is deployed and accessible
3. Check CORS is enabled in backend
4. Test backend directly: `curl https://your-backend.com/health`

### Environment Variable Not Working

**Problem:** API URL not being used

**Solution:**
1. Environment variables need `NEXT_PUBLIC_` prefix for client-side
2. Redeploy after changing environment variables
3. Check in browser console: `console.log(process.env.NEXT_PUBLIC_API_URL)`

---

## Testing Your Deployment

### 1. Test Frontend Only

Visit your Vercel URL:
```
https://your-project.vercel.app
```

Expected:
- ✅ Page loads
- ✅ Chat interface visible
- ⚠️ API calls may fail (if backend not deployed)

### 2. Test Backend

```bash
curl https://your-backend.com/health
# Should return: {"status":"ok"}

curl -X POST https://your-backend.com/v1/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"tinyllama","prompt":"Hello","max_tokens":20}'
```

### 3. Test End-to-End

1. Open your Vercel URL
2. Type a message in chat
3. Should see streaming response
4. Metrics should update

---

## Cost Breakdown

### Vercel (Frontend)
- **Hobby Plan:** FREE (perfect for personal projects)
- **Pro Plan:** $20/month (for commercial use)

### Backend Options

**Option A: DigitalOcean CPU Droplet**
- Cost: $12/month
- Performance: Good for testing, ~2 req/s
- Setup: 30 minutes

**Option B: DigitalOcean + GPU**
- Cost: $150-500/month
- Performance: Production-ready, ~50+ req/s
- Setup: 1 hour

**Option C: Ngrok (Development Only)**
- Cost: FREE (with limits)
- Performance: Same as local
- Setup: 2 minutes
- Note: Not for production

---

## Recommended Setup

**For Portfolio/Demo:**
- Frontend: Vercel (free)
- Backend: Local + ngrok (free)
- Total: $0/month

**For Small Production:**
- Frontend: Vercel (free or $20)
- Backend: DigitalOcean CPU ($12)
- Total: $12-32/month

**For Real Production:**
- Frontend: Vercel Pro ($20)
- Backend: GPU server ($150+)
- Total: $170+/month

---

## Quick Commands Reference

```bash
# Local development
./start-demo.sh

# Build frontend locally
cd frontend && npm run build

# Test production build locally
cd frontend && npm start

# Deploy backend
git push origin main
# Then deploy on DigitalOcean/AWS

# Expose local backend
ngrok http 8000

# Check Vercel logs
vercel logs your-deployment-url
```

---

## Need Help?

1. **Vercel Issues:** Check [Vercel Docs](https://vercel.com/docs)
2. **Backend Issues:** See [DEPLOYMENT.md](DEPLOYMENT.md)
3. **CORS Issues:** Check `cmd/gateway/main.go` CORS middleware
4. **Build Issues:** Try `cd frontend && npm run build` locally first

---

**Your frontend should now deploy correctly to Vercel!** 🚀

Next step: Deploy the backend or use ngrok for testing.

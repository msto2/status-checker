# Quick Start Guide

Get your Service Monitor up and running in 15 minutes!

## Step 1: Prepare Proxmox Container (5 minutes)

1. Create LXC container in Proxmox:
   - Template: Debian 12 or Ubuntu 22.04
   - RAM: 512MB
   - CPU: 1 core
   - Disk: 8GB
   - Network: Bridge to your local network

2. Start container and note its IP address

## Step 2: Install Backend (5 minutes)

```bash
# SSH into container
ssh root@YOUR_CONTAINER_IP

# Download and extract (replace with your method)
# Option A: Clone from git
git clone YOUR_REPO_URL
cd service-monitor

# Option B: Upload files via SCP
# From your machine: scp -r service-monitor root@YOUR_CONTAINER_IP:/tmp/

# Run installation
cd /tmp/service-monitor
chmod +x scripts/install.sh
./scripts/install.sh
```

**IMPORTANT**: Save the API key shown during installation!

Example output:
```
=========================================
IMPORTANT: Your API Key
=========================================
a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2
=========================================
```

## Step 3: Deploy Frontend to Cloudflare (5 minutes)

```bash
# On your local machine (not the container)

# Install Wrangler
npm install -g wrangler

# Login to Cloudflare
wrangler login

# Create KV namespace
wrangler kv:namespace create STATUS_STORE --preview=false
```

Output will show:
```
{ binding = "STATUS_STORE", id = "abcd1234efgh5678" }
```

**Save this ID!**

```bash
# Deploy frontend
cd frontend
wrangler pages deploy . --project-name=service-monitor
```

## Step 4: Configure Cloudflare (2 minutes)

1. Go to Cloudflare Dashboard → Workers & Pages → service-monitor

2. Click **Settings** → **Functions**

3. Add **KV namespace binding**:
   - Variable name: `STATUS_STORE`
   - KV namespace: Select the namespace you created

4. Click **Settings** → **Environment variables** → **Add variable**:
   - Name: `API_KEY`
   - Value: Paste the API key from Step 2

5. Click **Custom domains** → **Set up a custom domain**:
   - Enter: `status.triplepoint.me`
   - Save

## Step 5: Start Monitoring (1 minute)

```bash
# Back on the Proxmox container
systemctl start service-monitor
systemctl enable service-monitor

# Check it's running
systemctl status service-monitor
```

## Step 6: Add Your First Service

1. Open browser: `http://YOUR_CONTAINER_IP:8080/admin.html`

2. Click **"Add Service"**

3. Enter:
   - Name: `Router`
   - IP Address: `192.168.1.1` (your router's IP)
   - Description: `Home router`

4. Click **"Save"**

5. Wait 1-2 minutes, then click **"Check"** button to test immediately

## Step 7: View Public Status Page

Open: **https://status.triplepoint.me**

You should see your service(s) with their status!

## Next Steps

### Add Email Alerts

1. In admin interface, go to **"Configuration"** tab

2. Enter SMTP settings:
   - For Gmail:
     - SMTP Host: `smtp.gmail.com`
     - SMTP Port: `587`
     - Username: `your-email@gmail.com`
     - Password: [Get App Password](https://support.google.com/accounts/answer/185833)
     - From: `Service Monitor <alerts@yourdomain.com>`

3. Click **"Save Configuration"**

4. Test by clicking **"Test Email"**

### Add Alert Recipients

1. Go to **"Email Alerts"** tab

2. Click **"Add Recipient"**

3. Enter email address

4. Click **"Add"**

Now you'll receive emails when services go down!

## Troubleshooting

### Can't access admin interface?

```bash
# Check service is running
systemctl status service-monitor

# Check firewall
ufw status

# Verify port is open
curl http://localhost:8080/health
```

### Frontend not updating?

```bash
# Check backend logs
tail -f /var/log/service-monitor/app.log

# Look for "Successfully pushed status update"
```

### Need help?

Check the full [README.md](README.md) for detailed troubleshooting.

## Summary

✅ Backend installed on Proxmox container
✅ Frontend deployed to Cloudflare Pages
✅ Custom domain configured (status.triplepoint.me)
✅ First service added and monitored
✅ Public status page accessible

**Total time**: ~15 minutes

**Next**: Add more services, configure email alerts, customize settings!

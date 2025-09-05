# Docker Containerization Setup

## Files Created

### Backend
- `backend/Dockerfile` - Multi-stage Go build (already existed)
- `backend/.dockerignore` - Excludes unnecessary files

### Frontend  
- `frontend/Dockerfile` - Multi-stage React build with Nginx
- `frontend/nginx.conf` - Nginx configuration for SPA routing

### Orchestration
- `docker-compose.yml` - Runs both services together
- `.env` - Environment variables for docker-compose

## Local Testing Commands

```bash
# Start Docker Desktop first, then:

# Build and run both services
docker-compose up --build

# Or run individually:
docker build -t drumkit-backend ./backend
docker build -t drumkit-frontend ./frontend

# Run with environment variables
docker run -p 8080:8080 --env-file backend/.env drumkit-backend
docker run -p 3000:80 drumkit-frontend
```

## Environment Setup

1. Copy your Turvo credentials to the root `.env` file
2. Update `frontend/.env.production` with your production API URL
3. For AWS deployment, update API URLs to point to your AWS backend

## Next Steps for AWS

1. Push images to Amazon ECR (Elastic Container Registry)
2. Deploy using ECS (Elastic Container Service) or EKS (Kubernetes)
3. Set up Application Load Balancer
4. Configure environment variables in AWS

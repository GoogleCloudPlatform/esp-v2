# Use Cases

Google Cloud Endpoints and ESPv2 support both local and remote backends.

The diagram below displays ESPv2 in reverse proxy mode for remote backends.
ESPv2 is deployed as a Google Cloud Run service, and it proxies requests to remote backends
deployed on serverless backends. The backends can be deployed anywhere, including:
- Google Cloud Run
- Google Cloud Function
- Google App Engine
- Non-GCP platforms

![Reverse Proxy Deployment](images/api-gateway-deployment.jpg)

The diagram below displays ESPv2 in sidecar deployment mode for local backends.
ESPv2 is deployed as a container on Google Compute Engine, Google
Kubernetes Engine, or any non-GCP Kubernetes cluster. The backend is also
deployed alongside ESPv2.

![Sidecar Deployment](images/sidecar-deployment.jpg)
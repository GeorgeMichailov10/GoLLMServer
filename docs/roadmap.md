## Iterative Roadmap
The project development will proceed in three main iterations:

### Iteration 1: MVP – Basic End-to-End Query Streaming
Objective:
Establish a functional chain from the vLLM model through the Go backend to the Python (Streamlit) frontend.
Key Deliverables:
Running vLLM model (e.g., neo-125M) with streaming response capability.
Go server implementation with WebSocket and gRPC integration.
Python frontend to send queries and display responses securely using JWT and TLS.
Outcome:
A working minimal viable product (MVP) that demonstrates end-to-end communication and streaming query handling.

#### Iteration 2: Mobile Integration & Metadata Management
Objective:
Expand the front-end by integrating a Flutter mobile application, and implement persistent conversation history and metadata storage using MongoDB.
Key Deliverables:
A Flutter-based mobile app with a chat interface and secure API connectivity.
Enhanced Python and Go backend modules to manage conversation contexts.
Integration with MongoDB to store and retrieve conversation histories.
Outcome:
A unified user experience across web and mobile, with full history tracking and metadata management.

#### Iteration 3: Kubernetes-Based Scalability
Objective:
Containerize the services and deploy using Kubernetes to support scalability, high availability, and streamlined maintenance.
Key Deliverables:
Docker images for all components (Go server, Python frontend, vLLM, MongoDB).
Kubernetes manifests (Deployments, Services, Ingress, Secrets).
Automated CI/CD pipelines for build, test, and deployment.
Outcome:
A production-ready system capable of dynamic scaling and resilient operation.
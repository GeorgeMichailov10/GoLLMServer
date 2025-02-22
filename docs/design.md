## Custom Chat App with vLLM
### Overview
The Custom Chat App with vLLM project is designed to provide an end-to-end solution where users can interact with a custom fine-tuned language model through both browser- and mobile-based frontends. The model is served via vLLM, while a Go server handles WebSocket and gRPC communications for efficient, real-time streaming. The system further integrates MongoDB for conversation history and metadata storage, and is built with a phased approach to ultimately incorporate Kubernetes for scalability.

### Architecture & Design
#### Components
##### vLLM Model Server:
Hosts a fine-tuned model (starting with a lightweight model such as neo-125M) that processes user queries and streams responses. It is configured to emulate the OpenAI API style, managing conversation context where needed.

##### Go Server (Backend):
Serves as the central communication layer implementing WebSocket and gRPC protocols for streaming queries and responses. It interfaces directly with the vLLM instance and enforces security policies (JWT, TLS).

##### Python Frontend (Streamlit):
Provides a web-based chat interface that allows users to send queries and view streamed responses. The Python layer integrates with the backend and includes JWT-based authentication.

##### Mobile Frontend (Flutter):
A cross-platform mobile application designed with Flutter. It mimics the web chat experience while including additional features such as conversation history and metadata management. It also communicates with the backend using REST/WebSocket/gRPC.

##### MongoDB:
Stores conversation histories and additional metadata. This component provides persistence for user interactions, enabling future enhancements like analytics and model fine-tuning based on real-world usage.

##### (Future) Kubernetes:
Planned to manage container orchestration and scaling. Kubernetes will be integrated to deploy the application components in a production environment, ensuring high availability and simplified maintenance.
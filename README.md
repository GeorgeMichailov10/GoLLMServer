# GoLLMServer
By George Michailov

### Project Overview

GoLLMServer is a server implementation designed to emulate interactions with a custom LLM. This server aims to provide efficient and scalable LLM services. The server is built in Go, using WebSockets and gRPC, ensuring high performance and reliability.

### Key Features

- **Token Streaming** : Using web sockets and gRPC, tokens are being streamed straight to the user as they are coming out of the model for minimal delay.
- **Query Handling** : Using vLLM to host the model asynchronously and implementing server-side logic that only allows 5 active queries and a limit on the queries themselves, this ensures that the model can run batch inference to multiple users simultaneously and respond to waiting users in a timely manner.
- **Security** : Using JWT for the HTTP requests and Web Socket interactions, all communication is secure.
- **Chat History** : Using MongoDB, all user-model interactions are stored so that users can come back to previous conversations and continue them. 
- **User-Friendly UI** : Built in Python, this UI is extremely easy to navigate and was extremely fast to build with easy debugging.

### Implementation Details

The frontend page does not allow access to any page other than login and registration until sign in. Upon sign-in the user is routed to the chat interface page. From here, the user may chat with the model, access old chats, create new chats, and delete their account. If the user chooses to access an old chat, their chat history is immediately loaded in and displayed. When they create a new chat, it is not added to the database until an interaction occurs. When the user sends a chat, it connects to the go server using web sockets and sends the query, chatid (if not a new chat), and its JWT token. Following verification, if the chat doesn't exist yet, it is added to the user. The server adds this request to an internal queue manager that is responsible for making sure the LLM is not overloaded with requests (would run out of memory and crash). If the request sits in the queue for too long, it times out and apologizes to the user. If it doesn't, it gets sent to the LLM and then as the response tokens are flowing out one by one, they are sent down a channel and sent to the user through the web socket. This incredibly fast optimization allows it to appear as if responses are coming back right away.

I really wanted to add Kubernetes around the vLLM server, but I don't have the hardware to try this on. If I have time in the future, I will come back and rebuild the model service to have Kubernetes to scale under demand, change the implementation logic in the model_service.go file to accomodate this, and use Kubernetes on the go server itself.

### Future Additions

TODO: Create Docker image

IMMEDIATE: RESET TIMEOUTS TO NONTESTING VALUES
1. Stream tokens out in the front end.
2. Implement TLS for secure web socket communication.
3. Kubernetes (Wrap entire goserver + vLLM for simplicity for first iteration)
4. Create service with small LLM that gives titles to chats rather than just id.
5. Implement conversation memory using LangChain
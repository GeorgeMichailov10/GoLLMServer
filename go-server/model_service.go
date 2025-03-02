package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	pb "github.com/GeorgeMichailov/personalllmchat/go-server/model-service"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	gracePeriod      = 5 * time.Second
	scanInterval     = 500 * time.Millisecond
	grpcQueryTimeout = 120 * time.Second // Increased timeout for gRPC query
	grpcDialTimeout  = 60 * time.Second  // Increased dial timeout
	websocketTimeout = grpcDialTimeout   // gracePeriod + 2*time.Second // Increased WS timeout and added margin
)

type Request struct {
	query      string
	responseCh chan string
	createdAt  time.Time
	isActive   bool
	isComplete bool
	ChatID     string
}

type IncomingWSMessage struct {
	Claims Claims `json:"claims,omitempty"`
	Query  string `json:"query"`
	ChatID string `json:"chatid"`
}

type RequestQueueManager struct {
	queue            []*Request
	activeQueries    int8
	maxActiveQueries int8
	mu               sync.Mutex
	cond             *sync.Cond
}

func createRequestQueueManager() *RequestQueueManager {
	rqm := &RequestQueueManager{
		queue:            make([]*Request, 0, 30),
		maxActiveQueries: 5,
		activeQueries:    0,
	}
	rqm.cond = sync.NewCond(&rqm.mu)
	return rqm
}

func (rqm *RequestQueueManager) AddRequest(req *Request) {
	log.Printf("[Queue] Adding new request: %s", req.query)
	rqm.mu.Lock()
	rqm.queue = append(rqm.queue, req)
	rqm.mu.Unlock()
	rqm.cond.Signal()
}

func (rqm *RequestQueueManager) Start(ctx context.Context) {
	// Routine: Scans for expired requests. Closes their channel and removes from queue.
	go func() {
		ticker := time.NewTicker(scanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rqm.mu.Lock()
				var freshQueue []*Request
				for _, req := range rqm.queue {
					if req.isComplete {
						continue
					}
					now := time.Now()

					// If request is stale: log and close the channel.
					if now.Sub(req.createdAt) > gracePeriod && !req.isActive {
						log.Printf("[Queue Timeout] Request expired for query: %s", req.query)
						close(req.responseCh)
						continue
					}

					// If request is pending and there is capacity: dispatch it.
					if !req.isActive && rqm.activeQueries < rqm.maxActiveQueries {
						log.Printf("[Dispatch] Dispatching query: %s", req.query)
						req.isActive = true
						rqm.activeQueries++
						go func(r *Request) {
							vLlmInteractor(r)
							rqm.mu.Lock()
							r.isActive = false
							rqm.activeQueries--
							rqm.mu.Unlock()
						}(req)
						continue
					}

					freshQueue = append(freshQueue, req)
				}
				rqm.queue = freshQueue
				rqm.mu.Unlock()
			case <-ctx.Done():
				log.Printf("[Queue] Context done, stopping queue manager.")
				return
			}
		}
	}()
}

func vLlmInteractor(req *Request) {
	_, dialCancel := context.WithTimeout(context.Background(), grpcDialTimeout)
	defer dialCancel()

	log.Printf("[gRPC NewClient] Attempting to create a new client for query: %s", req.query)
	grpcConn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("[Timeout: gRPC NewClient] Failed to create new client: %v for query: %s", err, req.query)
		close(req.responseCh)
		return
	}
	defer grpcConn.Close()
	log.Printf("[gRPC NewClient] Client created successfully for query: %s", req.query)
	queryCtx, queryCancel := context.WithTimeout(context.Background(), grpcQueryTimeout)
	defer queryCancel()

	gReq := &pb.QueryRequest{Query: req.query}
	client := pb.NewVLLMServiceClient(grpcConn)
	log.Printf("[gRPC Query] Sending query to gRPC server: %s", req.query)
	stream, err := client.Query(queryCtx, gReq)
	if err != nil {
		log.Printf("[gRPC Query Error] %v for query: %s", err, req.query)
		close(req.responseCh)
		return
	}

	// Read tokens from the gRPC stream
	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Printf("[gRPC Recv Error] %v for query: %s", err, req.query)
			break
		}
		log.Printf("[gRPC Token] Received token: '%s' for query: %s", resp.Token, req.query)
		req.responseCh <- resp.Token
		if resp.Token == "[END]" {
			log.Printf("[gRPC Complete] Finished query: %s", req.query)
			break
		}
	}
	close(req.responseCh)
	req.isComplete = true
}

var rqManager *RequestQueueManager = createRequestQueueManager()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// TODO: Add origin check here
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket Upgrade Error] %v", err)
		return
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("[WebSocket Read Error] %v", err)
		return
	}

	var incoming IncomingWSMessage
	if err := json.Unmarshal(message, &incoming); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Invalid JSON format"}`))
		return
	}
	log.Printf("Received message for ChatID [%s]: %s\n", incoming.ChatID, incoming.Query)

	if len(incoming.ChatID) == 0 {
		success, newchatid := CreateNewUserChat(incoming.Claims.Username)

		if !success {
			conn.WriteMessage(websocket.TextMessage, []byte(`{"error" : "Failed to add chat to user."}`))
			return
		}

		incoming.ChatID = newchatid.Hex()
	}

	req := &Request{
		query:      incoming.Query,
		responseCh: make(chan string, 10),
		createdAt:  time.Now(),
		isActive:   false,
		isComplete: false,
		ChatID:     incoming.ChatID,
	}

	rqManager.AddRequest(req)
	var modelResponse string

	for {
		select {
		case token, ok := <-req.responseCh:
			if !ok {
				log.Printf("[WebSocket] Response channel closed for query: %s", incoming.Query)
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, []byte(token)); err != nil {
				log.Printf("[WebSocket Write Error] %v for query: %s", err, incoming.Query)
				return
			}

			if token != "[END]" {
				modelResponse += token
				return
			} else {
				log.Printf("[WebSocket] Completed sending tokens for query: %s", incoming.Query)
				interaction := ChatInteraction{
					ChatID:    req.ChatID,
					UserChat:  incoming.Query,
					ModelChat: modelResponse,
				}

				go AddInteraction(interaction)
				return
			}
		case <-time.After(websocketTimeout):
			log.Printf("[Timeout: WebSocket Response] No response received in %v for query: %s", websocketTimeout, incoming.Query)
			conn.WriteMessage(websocket.TextMessage, []byte("Timeout: no response received."))
			return
		}
	}
}

func SimulationWsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket Upgrade Error] %v", err)
		return
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("[WebSocket Read Error] %v", err)
		return
	}

	var incoming IncomingWSMessage
	if err := json.Unmarshal(message, &incoming); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Invalid JSON format"}`))
		return
	}
	log.Printf("Received message for ChatID [%s]: %s\n", incoming.ChatID, incoming.Query)

	if len(incoming.ChatID) == 0 {
		success, newchatid := CreateNewUserChat(incoming.Claims.Username)

		if !success {
			conn.WriteMessage(websocket.TextMessage, []byte(`{"error" : "Failed to add chat to user."}`))
			return
		}

		incoming.ChatID = newchatid.Hex()
	}

	modelResponse := "Simulated response"

	if err := conn.WriteMessage(websocket.TextMessage, []byte(modelResponse)); err != nil {
		log.Printf("[WebSocket Write Error] %v for query: %s", err, incoming.Query)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("[END]")); err != nil {
		log.Printf("[WebSocket Write Error] %v for query: %s", err, incoming.Query)
		return
	}

	log.Printf("[WebSocket] Completed sending simulated tokens for query: %s", incoming.Query)

	interaction := ChatInteraction{
		ChatID:    incoming.ChatID,
		UserChat:  incoming.Query,
		ModelChat: modelResponse,
	}
	go AddInteraction(interaction)
}

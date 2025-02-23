package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	pb "github.com/GeorgeMichailov/personalllmchat/go-server/model-service"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
)

const (
	gracePeriod  = 5 * time.Second
	scanInterval = 500 * time.Millisecond
)

type Request struct {
	query      string
	responseCh chan string
	createdAt  time.Time
	isActive   bool
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
	rqm.mu.Lock()
	rqm.queue = append(rqm.queue, req)
	rqm.mu.Unlock()
	rqm.cond.Signal()
}

func (rqm *RequestQueueManager) Start(ctx context.Context) {
	// Routine 1: Scans for expired requests. Closes their channel and removes from queue
	go func() {
		ticker := time.NewTicker(scanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rqm.mu.Lock()
				var freshQueue []*Request
				for _, req := range rqm.queue {
					now := time.Now()

					// If request is stale: close the channel and continue without adding
					if now.Sub(req.createdAt) > gracePeriod && !req.isActive {
						close(req.responseCh)
						continue
					}

					// If request is pending and have capacity: Dispatch it
					if !req.isActive && rqm.activeQueries < rqm.maxActiveQueries {
						req.isActive = true
						rqm.activeQueries++

						// Dispatch request
						go func(r *Request) {
							vLlmInteractor(r)
							rqm.mu.Lock()
							r.isActive = false
							rqm.activeQueries -= 1
							rqm.mu.Unlock()
						}(req)
						continue
					}

					freshQueue = append(freshQueue, req)

				}
				rqm.queue = freshQueue
				rqm.mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func vLlmInteractor(req *Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	grpcConn, err := grpc.Dial("vllm-container:50051", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(6*time.Second))
	if err != nil {
		log.Printf("Failed to dial gRPC server: %v", err)
		req.responseCh <- "Internal server error"
		close(req.responseCh)
		return
	}
	defer grpcConn.Close()

	gReq := &pb.QueryRequest{Query: req.query}
	client := pb.NewVLLMServiceClient(grpcConn)
	stream, err := client.Query(ctx, gReq)
	if err != nil {
		log.Printf("gRPC Query error: %v", err)
		req.responseCh <- "Internal server error"
		close(req.responseCh)
		return
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving from gRPC stream: %v", err)
			break
		}
		req.responseCh <- resp.Token
		if resp.Token == "[END]" {
			break
		}
	}
	close(req.responseCh)
}

var rqManager *RequestQueueManager = createRequestQueueManager()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// TODO: Add origin check here
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection from HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Web Socket upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	// Read prompt from client
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("Error reading WebSocket message: %v", err)
		return
	}
	query := string(message)
	log.Printf("Received query: %s", query)
	// Create Request and add to queue
	req := &Request{
		query:      query,
		responseCh: make(chan string, 10),
		createdAt:  time.Now(),
		isActive:   false,
	}

	rqManager.AddRequest(req)

	// Stream tokens as they come in through channel to the web socket.
	for {
		select {
		case token, ok := <-req.responseCh:
			if !ok { // Closed channel
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, []byte(token)); err != nil {
				log.Printf("Error writing to WebSocket: %v", err)
				return
			}

			if token == "[END]" {
				return
			}
		case <-time.After(gracePeriod + time.Second):
			conn.WriteMessage(websocket.TextMessage, []byte("Timeout: no response received."))
			return
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rqManager.Start(ctx)

	http.HandleFunc("/ws", wsHandler)

	addr := ":8080"
	log.Printf("Starting server on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("ListenAndServe error: %v", err)
	}
}

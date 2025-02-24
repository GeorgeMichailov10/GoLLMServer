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
	// Create a dialing context with a timeout.
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
	query := string(message)
	log.Printf("[WebSocket Received] Query: %s", query)

	req := &Request{
		query:      query,
		responseCh: make(chan string, 10),
		createdAt:  time.Now(),
		isActive:   false,
	}

	rqManager.AddRequest(req)

	for {
		select {
		case token, ok := <-req.responseCh:
			if !ok {
				log.Printf("[WebSocket] Response channel closed for query: %s", query)
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, []byte(token)); err != nil {
				log.Printf("[WebSocket Write Error] %v for query: %s", err, query)
				return
			}

			if token == "[END]" {
				log.Printf("[WebSocket] Completed sending tokens for query: %s", query)
				return
			}
		case <-time.After(websocketTimeout):
			log.Printf("[Timeout: WebSocket Response] No response received in %v for query: %s", websocketTimeout, query)
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

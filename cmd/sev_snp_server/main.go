package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/IntelliXLabs/wasmvm-tee/dtvm"
)

var (
	grpcPort   = flag.Int("grpc-port", 50051, "gRPC server port")
	httpPort   = flag.Int("http-port", 8080, "HTTP server port")
	enableHTTP = flag.Bool("enable-http", true, "Enable HTTP/REST API gateway")
	enableGRPC = flag.Bool("enable-grpc", true, "Enable gRPC server")
)

func main() {
	flag.Parse()

	log.Printf("Starting DTVM TEE server...")
	log.Printf("gRPC port: %d, HTTP port: %d", *grpcPort, *httpPort)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start gRPC server if enabled
	if *enableGRPC {
		go startGRPCServer(ctx, *grpcPort)
	}

	// Start HTTP server if enabled
	if *enableHTTP {
		// Wait a moment for gRPC server to start
		time.Sleep(100 * time.Millisecond)
		go startHTTPServer(ctx, *httpPort, *grpcPort)
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down servers...")
	cancel()

	// Give servers time to gracefully shutdown
	time.Sleep(2 * time.Second)
	log.Println("Server shutdown complete")
}

// startGRPCServer starts the gRPC server
func startGRPCServer(ctx context.Context, port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port %d: %v", port, err)
	}

	// Create gRPC server instance
	grpcServer := grpc.NewServer()

	// Register DTVM TEE service
	dtvmServer := &dtvm.Server{}
	dtvm.RegisterDTVMTeeServiceServer(grpcServer, dtvmServer)

	log.Printf("✅ gRPC server listening at %v", listener.Addr())

	// Start serving in a goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down gRPC server...")
	grpcServer.GracefulStop()
}

// startHTTPServer starts the HTTP server with grpc-gateway
func startHTTPServer(ctx context.Context, httpPort, grpcPort int) {
	// Create grpc-gateway mux with custom options
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{}),
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			// Forward all headers
			return key, true
		}),
		runtime.WithErrorHandler(customErrorHandler),
	)

	// Setup gRPC connection options
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	grpcServerEndpoint := fmt.Sprintf("localhost:%d", grpcPort)

	// Register DTVM service handler
	err := dtvm.RegisterDTVMTeeServiceHandlerFromEndpoint(ctx, mux, grpcServerEndpoint, opts)
	if err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	// Create HTTP server with enhanced routing
	httpMux := http.NewServeMux()

	// Register grpc-gateway routes
	httpMux.Handle("/", corsHandler(mux))

	// Add health check endpoint
	httpMux.HandleFunc("/health", corsHandlerFunc(healthCheckHandler))

	// Add API info endpoint
	httpMux.HandleFunc("/api/info", corsHandlerFunc(apiInfoHandler))

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", httpPort),
		Handler:      httpMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("✅ HTTP server listening at http://localhost:%d", httpPort)
	log.Printf("📋 API endpoints available:")
	log.Printf("   POST http://localhost:%d/v1/dtvm/execute", httpPort)
	log.Printf("   GET  http://localhost:%d/health", httpPort)
	log.Printf("   GET  http://localhost:%d/api/info", httpPort)

	// Start serving in a goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP server: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down HTTP server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}

// corsHandler adds CORS headers to support cross-origin requests for Handlers
func corsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Log requests
		log.Printf("HTTP %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(w, r)
	})
}

// corsHandlerFunc adds CORS headers to support cross-origin requests for HandlerFuncs
func corsHandlerFunc(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Log requests
		log.Printf("HTTP %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		fn(w, r)
	}
}

// healthCheckHandler provides a simple health check endpoint
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","service":"wasmvm-tee","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// apiInfoHandler provides API documentation
func apiInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	apiInfo := map[string]interface{}{
		"service":     "WASMVM TEE Service",
		"version":     "1.0.0",
		"description": "Trusted Execution Environment for WASMVM",
		"endpoints": map[string]interface{}{
			"execute": map[string]interface{}{
				"method":      "POST",
				"path":        "/v1/dtvm/execute",
				"description": "Execute WASMVM bytecode in TEE environment",
				"example": map[string]interface{}{
					"execution": map[string]interface{}{
						"version":    "1.0",
						"request_id": "test-001",
						"bytecode":   "AGFzbQEAAAABCgJgAX8AYAF/AX8DAgEBBQMBAAEHBwEDZmliAAAKHgEcACAAQQJIBH9BAQUgAEEBaxAAIABBAmsQAGoLCwsHAQBBAAsBeA==",
						"fn_name":    "fib",
						"inputs":     []string{"5"},
						"timestamp":  0,
					},
					"runtime_config": map[string]interface{}{
						"mode": 0,
						"gas_limit": map[string]interface{}{
							"use_gas_limit": false,
							"limit":         0,
						},
					},
				},
			},
			"health": map[string]interface{}{
				"method":      "GET",
				"path":        "/health",
				"description": "Service health check",
			},
		},
	}

	jsonData, _ := json.Marshal(apiInfo)
	w.Write(jsonData)
}

// customErrorHandler handles grpc-gateway errors
func customErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("gRPC Gateway error: %v", err)
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}

package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	pb "pubsub/gen" // замените на ваш реальный путь
	"pubsub/internal/config"
	"pubsub/internal/grpcservice"
	"pubsub/internal/pubsub"

	"google.golang.org/grpc"
)

func main() {
	// configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	portStr := strconv.Itoa(cfg.Port)
	shutdownTimeout := cfg.Timeout()

	// init dependencies
	serviveCtx, serviceCancel := context.WithCancel(context.Background())
	ps := pubsub.NewSubPub(cfg.ChannelBufferSize)
	grpcService := grpcservice.NewPubSubService(serviveCtx, ps)

	// creating gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingUnaryInterceptor),
		grpc.StreamInterceptor(loggingStreamInterceptor),
	)
	pb.RegisterPubSubServer(grpcServer, grpcService)

	// starting server
	lis, err := net.Listen("tcp", ":"+portStr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		log.Printf("gRPC server starting on port %s", portStr)
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// waiting SIGTERM or SIGINT for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	log.Println("Initiating graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := ps.Close(shutdownCtx); err != nil {
		log.Printf("error closing pubsub: %v", err)
	}
	log.Printf("pubsub service is closed")

	stopped := make(chan struct{})
	go func() {
		serviceCancel()
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("gRPC server stopped gracefully")
	case <-shutdownCtx.Done():
		log.Println("gRPC server forced to stop due to timeout")
		grpcServer.Stop()
	}

	log.Println("Shutdown completed")
}

func loggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Printf("unary call: %s", info.FullMethod)
	return handler(ctx, req)
}

func loggingStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Printf("stream call: %s", info.FullMethod)
	return handler(srv, ss)
}

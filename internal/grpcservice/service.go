package grpcservice

import (
	"context"
	"errors"
	"log"
	pb "pubsub/gen"
	"pubsub/internal/pubsub"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PubSubService struct {
	pb.UnimplementedPubSubServer
	subpub pubsub.SubPub
}

func NewPubSubService(subpub pubsub.SubPub) *PubSubService {
	return &PubSubService{
		subpub: subpub,
	}
}

func (s *PubSubService) Subscribe(req *pb.SubscribeRequest, stream pb.PubSub_SubscribeServer) error {
	ctx := stream.Context()
	key := req.GetKey()

	if key == "" {
		return status.Error(codes.InvalidArgument, "key is required")
	}

	log.Printf("new subscription for key: %s", key)
	defer log.Printf("unsubscribed from key: %s", key)

	// Buffered channel for error handling with context-aware closing
	errCh := make(chan error, 1)
	defer close(errCh)

	// Creating new subscriber
	sub, err := s.subpub.Subscribe(key, func(msg interface{}) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		strMsg, ok := msg.(string)
		if !ok {
			select {
			case errCh <- status.Errorf(codes.Internal, "invalid message type: %T", msg):
			case <-ctx.Done():
			}
			return
		}

		if err := stream.Send(&pb.Event{Data: strMsg}); err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
		}
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	select {
	case <-ctx.Done():
		return handleContextError(ctx)
	case err := <-errCh:
		if status.Code(err) == codes.Internal {
			log.Printf("subscription error for key %s: %v", key, err)
		}
		return err
	}
}

func handleContextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return status.Error(codes.Canceled, "client cancelled the stream")
	case context.DeadlineExceeded:
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	default:
		return nil
	}
}

func (s *PubSubService) Publish(ctx context.Context, req *pb.PublishRequest) (*emptypb.Empty, error) {
	err := s.subpub.Publish(req.Key, req.Data)
	if err == nil {
		return &emptypb.Empty{}, nil
	}

	switch {
	case errors.Is(err, context.Canceled):
		return nil, status.Error(codes.Canceled, "operation canceled")
	case errors.Is(err, pubsub.ErrSubjectNotFound):
		return nil, status.Error(codes.NotFound, "key has no subscribers")
	case strings.Contains(err.Error(), "dropped"):
		return nil, status.Error(codes.ResourceExhausted, "message dropped")
	default:
		log.Printf("unexpected publish error: %v", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
}

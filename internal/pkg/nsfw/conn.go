package nsfw

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds configuration for a gRPC NSFW client.
// Shared across all NSFW client types (image, text, etc.).
type Config struct {
	Address string        // gRPC server address, e.g., "localhost:50051"
	Timeout time.Duration // Per-request timeout
}

// DefaultConfig returns a default gRPC config.
func DefaultConfig(addr string) Config {
	return Config{
		Address: addr,
		Timeout: 30 * time.Second,
	}
}

// Dial creates a new gRPC client connection from config.
// Caller is responsible for closing the connection.
func Dial(cfg Config) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("nsfw: failed to dial %s: %w", cfg.Address, err)
	}
	return conn, nil
}

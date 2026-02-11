package nsfw

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "moderation/api/nsfw_detector/v1"
)

// GRPCConfig holds configuration for gRPC NSFW detector client.
type GRPCConfig struct {
	Address string // gRPC server address, e.g., "localhost:50051"
	Timeout time.Duration
}

// DefaultGRPCConfig returns default gRPC configuration.
func DefaultGRPCConfig() GRPCConfig {
	return GRPCConfig{
		Address: "localhost:50051",
		Timeout: 30 * time.Second,
	}
}

// GRPCClient is a gRPC client for NSFW detection using Ray Serve backend.
type GRPCClient struct {
	config GRPCConfig
	conn   *grpc.ClientConn
	client pb.NSFWDetectorClient
}

// NewGRPCClient creates a new gRPC NSFW detection client.
func NewGRPCClient(config GRPCConfig) (*GRPCClient, error) {
	// Create gRPC connection
	conn, err := grpc.NewClient(
		config.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return &GRPCClient{
		config: config,
		conn:   conn,
		client: pb.NewNSFWDetectorClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Predict detects NSFW content from image bytes.
func (c *GRPCClient) Predict(ctx context.Context, imageData []byte) (*pb.PredictResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.Predict(ctx, &pb.PredictRequest{
		ImageData: imageData,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC Predict call failed: %w", err)
	}

	return resp, nil
}

// DetectFromURL detects NSFW content from a public image URL.
func (c *GRPCClient) PredictFromURL(ctx context.Context, imageURL string) (*pb.PredictResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.PredictFromURL(ctx, &pb.PredictURLRequest{
		Url: imageURL,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC PredictFromURL call failed: %w", err)
	}

	return resp, nil
}

// PredictBatchFromURLs detects NSFW content from multiple public image URLs (batch).
func (c *GRPCClient) PredictBatchFromURLs(ctx context.Context, imageURLs []string) (*pb.PredictBatchResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.PredictBatchFromURLs(ctx, &pb.PredictBatchURLRequest{
		Urls: imageURLs,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC PredictBatchFromURLs call failed: %w", err)
	}
	return resp, nil
}

// Ping checks if the gRPC NSFW API is reachable via health check.
func (c *GRPCClient) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	resp, err := c.client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		return fmt.Errorf("gRPC health check failed: %w", err)
	}
	if resp.Status != "healthy" {
		return fmt.Errorf("NSFW service is unhealthy: %s", resp.Status)
	}
	return nil
}

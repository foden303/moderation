package nsfw

import (
	"context"
	"fmt"
	"io"
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

// DetectFromBytes detects NSFW content from image bytes.
func (c *GRPCClient) DetectFromBytes(ctx context.Context, imageData []byte) (*DetectionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.Predict(ctx, &pb.PredictRequest{
		ImageData: imageData,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC Predict call failed: %w", err)
	}

	return &DetectionResult{
		IsNSFW:      resp.IsNsfw,
		NSFWScore:   float64(resp.NsfwScore),
		NormalScore: float64(resp.NormalScore),
		Label:       resp.Label,
		Confidence:  float64(resp.Confidence),
		ProcessedAt: time.Now(),
	}, nil
}

// DetectFromReader detects NSFW content from io.Reader.
func (c *GRPCClient) DetectFromReader(ctx context.Context, reader io.Reader) (*DetectionResult, error) {
	imageData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}
	return c.DetectFromBytes(ctx, imageData)
}

// DetectFromURL detects NSFW content from a public image URL.
func (c *GRPCClient) DetectFromURL(ctx context.Context, imageURL string) (*DetectionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.PredictFromURL(ctx, &pb.PredictURLRequest{
		Url: imageURL,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC PredictFromURL call failed: %w", err)
	}

	return &DetectionResult{
		IsNSFW:      resp.IsNsfw,
		NSFWScore:   float64(resp.NsfwScore),
		NormalScore: float64(resp.NormalScore),
		Label:       resp.Label,
		Confidence:  float64(resp.Confidence),
		ProcessedAt: time.Now(),
	}, nil
}

// DetectFromURLs detects NSFW content from multiple public image URLs (batch).
func (c *GRPCClient) DetectFromURLs(ctx context.Context, imageURLs []string) ([]BatchURLResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.PredictBatchFromURLs(ctx, &pb.PredictBatchURLRequest{
		Urls: imageURLs,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC PredictBatchFromURLs call failed: %w", err)
	}

	results := make([]BatchURLResult, len(resp.Predictions))
	for i, pred := range resp.Predictions {
		results[i].URL = pred.Url
		if pred.Error != "" {
			results[i].Error = fmt.Errorf("%s", pred.Error)
		} else if pred.Result != nil {
			results[i].Result = &DetectionResult{
				IsNSFW:      pred.Result.IsNsfw,
				NSFWScore:   float64(pred.Result.NsfwScore),
				NormalScore: float64(pred.Result.NormalScore),
				Label:       pred.Result.Label,
				Confidence:  float64(pred.Result.Confidence),
				ProcessedAt: time.Now(),
			}
		}
	}

	return results, nil
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

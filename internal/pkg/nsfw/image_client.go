package nsfw

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	pb "moderation/api/nsfw_image/v1"
)

// ImageClient is a gRPC client for NSFW image detection (Ray Serve backend).
type ImageClient struct {
	config Config
	conn   *grpc.ClientConn
	client pb.NSFWImageDetectorClient
}

// NewImageClient creates a new gRPC NSFW image detection client.
func NewImageClient(cfg Config) (*ImageClient, error) {
	conn, err := Dial(cfg)
	if err != nil {
		return nil, err
	}
	return &ImageClient{
		config: cfg,
		conn:   conn,
		client: pb.NewNSFWImageDetectorClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *ImageClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Predict detects NSFW content from raw image bytes.
func (c *ImageClient) Predict(ctx context.Context, imageData []byte) (*pb.PredictResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.Predict(ctx, &pb.PredictRequest{ImageData: imageData})
	if err != nil {
		return nil, fmt.Errorf("nsfw image Predict failed: %w", err)
	}
	return resp, nil
}

// PredictFromURL detects NSFW content from a public image URL.
func (c *ImageClient) PredictFromURL(ctx context.Context, imageURL string) (*pb.PredictResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.PredictFromURL(ctx, &pb.PredictURLRequest{Url: imageURL})
	if err != nil {
		return nil, fmt.Errorf("nsfw image PredictFromURL failed: %w", err)
	}
	return resp, nil
}

// PredictBatchFromURLs detects NSFW content from multiple image URLs.
func (c *ImageClient) PredictBatchFromURLs(ctx context.Context, urls []string) (*pb.PredictBatchResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.PredictBatchFromURLs(ctx, &pb.PredictBatchURLRequest{Urls: urls})
	if err != nil {
		return nil, fmt.Errorf("nsfw image PredictBatchFromURLs failed: %w", err)
	}
	return resp, nil
}

// Ping checks if the NSFW image service is healthy.
func (c *ImageClient) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		return fmt.Errorf("nsfw image health check failed: %w", err)
	}
	if resp.Status != "healthy" {
		return fmt.Errorf("nsfw image service unhealthy: %s", resp.Status)
	}
	return nil
}

package nsfw

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	pb "moderation/api/nsfw_text/v1"
)

// TextClient is a gRPC client for NSFW text detection (Qwen3Guard backend).
type TextClient struct {
	config Config
	conn   *grpc.ClientConn
	client pb.NSFWTextDetectorClient
}

// NewTextClient creates a new gRPC NSFW text detection client.
func NewTextClient(cfg Config) (*TextClient, error) {
	conn, err := Dial(cfg)
	if err != nil {
		return nil, err
	}
	return &TextClient{
		config: cfg,
		conn:   conn,
		client: pb.NewNSFWTextDetectorClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *TextClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Predict checks safety of a single text.
func (c *TextClient) Predict(ctx context.Context, text string) (*pb.PredictResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.Predict(ctx, &pb.PredictRequest{Text: text})
	if err != nil {
		return nil, fmt.Errorf("nsfw text Predict failed: %w", err)
	}
	return resp, nil
}

// PredictBatch checks safety for multiple texts.
func (c *TextClient) PredictBatch(ctx context.Context, texts []string) (*pb.PredictBatchResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.PredictBatch(ctx, &pb.PredictBatchRequest{Texts: texts})
	if err != nil {
		return nil, fmt.Errorf("nsfw text PredictBatch failed: %w", err)
	}
	return resp, nil
}

// Ping checks if the NSFW text service is healthy.
func (c *TextClient) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		return fmt.Errorf("nsfw text health check failed: %w", err)
	}
	if resp.Status != "healthy" {
		return fmt.Errorf("nsfw text service unhealthy: %s", resp.Status)
	}
	return nil
}

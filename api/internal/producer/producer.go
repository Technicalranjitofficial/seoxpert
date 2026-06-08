package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"github.com/seoxpert/shared/events"
)

// Producer wraps the franz-go Kafka client for publishing jobs to Redpanda.
// One producer instance is shared across all request goroutines — it is thread-safe.
type Producer struct {
	client *kgo.Client
}

func New(brokers []string) (*Producer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
		kgo.ProducerBatchMaxBytes(1_000_000),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)),
	}

	// SASL — only enabled when REDPANDA_SASL_USER is set in env.
	if user := os.Getenv("REDPANDA_SASL_USER"); user != "" {
		pass := os.Getenv("REDPANDA_SASL_PASSWORD")
		auth := scram.Auth{User: user, Pass: pass}
		opts = append(opts, kgo.SASL(auth.AsSha256Mechanism()))
	}

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("create kafka client: %w", err)
	}

	if err := cl.Ping(context.Background()); err != nil {
		cl.Close()
		return nil, fmt.Errorf("ping redpanda: %w", err)
	}

	slog.Info("redpanda producer ready", "brokers", brokers)
	return &Producer{client: cl}, nil
}

func (p *Producer) Close() {
	p.client.Close()
}

// PublishAuditJob sends an audit job to the audit.requested topic.
// Key is the ProjectID so all audits for the same project go to the same partition.
func (p *Producer) PublishAuditJob(ctx context.Context, job events.AuditJob) error {
	return p.publish(ctx, events.TopicAuditRequested, job.ProjectID, job)
}

// PublishRankJob sends a rank check job.
func (p *Producer) PublishRankJob(ctx context.Context, job events.RankJob) error {
	return p.publish(ctx, events.TopicRankRequested, job.ProjectID, job)
}

func (p *Producer) publish(ctx context.Context, topic, key string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: b,
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("produce to %s: %w", topic, err)
	}

	slog.Debug("published event", "topic", topic, "key", key)
	return nil
}

package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/seoxpert/shared/events"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

// RankJobHandler is called for each rank job consumed.
type RankJobHandler func(ctx context.Context, job events.RankJob) error

// Consumer reads RankJob messages from Redpanda.
type Consumer struct {
	client  *kgo.Client
	handler RankJobHandler
}

// New creates a Kafka consumer for rank jobs.
func New(brokers []string, saslUser, saslPass string, handler RankJobHandler) (*Consumer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup("rank-tracker-workers"),
		kgo.ConsumeTopics(events.TopicRankRequested),
		kgo.DisableAutoCommit(),
	}

	if saslUser != "" && saslPass != "" {
		mechanism := scram.Auth{
			User: saslUser,
			Pass: saslPass,
		}.AsSha256Mechanism()
		opts = append(opts, kgo.SASL(mechanism))
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &Consumer{client: client, handler: handler}, nil
}

// Run blocks until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) {
	defer c.client.Close()
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}
		fetches.EachError(func(t string, p int32, err error) {
			slog.Error("fetch error", "topic", t, "partition", p, "err", err)
		})
		fetches.EachRecord(func(rec *kgo.Record) {
			var job events.RankJob
			if err := json.Unmarshal(rec.Value, &job); err != nil {
				slog.Error("unmarshal rank job", "err", err)
				c.client.CommitRecords(ctx, rec)
				return
			}
			// Skip empty jobs
			if len(job.Keywords) == 0 || job.Domain == "" {
				c.client.CommitRecords(ctx, rec)
				return
			}
			if err := c.handler(ctx, job); err != nil {
				slog.Error("rank job handler failed", "err", err, "domain", job.Domain)
			}
			c.client.CommitRecords(ctx, rec)
		})
	}
}

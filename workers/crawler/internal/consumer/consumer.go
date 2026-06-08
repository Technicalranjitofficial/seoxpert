package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/seoxpert/shared/events"
)

// Consumer reads audit jobs from the Redpanda audit.requested topic.
// Uses a consumer group so multiple worker replicas auto-balance partitions.
type Consumer struct {
	client  *kgo.Client
	handler func(context.Context, events.AuditJob) error
}

func New(brokers []string, groupID string, handler func(context.Context, events.AuditJob) error) (*Consumer, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(events.TopicAuditRequested),
		// Commit only after successful processing — no job loss on crash.
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{client: cl, handler: handler}, nil
}

// Run starts the consume loop. Blocks until ctx is cancelled.
// Each message is processed sequentially per partition — order is preserved.
// For higher throughput, increase Redpanda partition count and run more replicas.
func (c *Consumer) Run(ctx context.Context) {
	slog.Info("crawler consumer started")
	defer c.client.Close()

	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				slog.Error("fetch error", "topic", e.Topic, "partition", e.Partition, "err", e.Err)
			}
			continue
		}

		fetches.EachRecord(func(record *kgo.Record) {
			var job events.AuditJob
			if err := json.Unmarshal(record.Value, &job); err != nil {
				slog.Error("unmarshal audit job", "err", err, "offset", record.Offset)
				// Commit anyway — malformed messages would loop forever.
				c.client.MarkCommitRecords(record)
				return
			}

			slog.Info("processing audit job",
				"audit_id", job.AuditID,
				"domain", job.Domain,
				"max_pages", job.MaxPages,
			)

			if err := c.handler(ctx, job); err != nil {
				slog.Error("audit job failed",
					"audit_id", job.AuditID,
					"err", err,
				)
				// TODO Phase 2: publish to dead-letter topic for retry.
			}

			c.client.MarkCommitRecords(record)
		})

		if err := c.client.CommitMarkedOffsets(ctx); err != nil {
			slog.Error("commit offsets", "err", err)
		}
	}
}

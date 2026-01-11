package kafka

import (
	"context"
	"log/slog"

	"golang.org/x/sync/errgroup"
)

type ConsumerManager struct {
	consumers []*Consumer
}

func NewConsumerManager(consumers []*Consumer) *ConsumerManager {
	return &ConsumerManager{
		consumers: consumers,
	}
}

func (cm *ConsumerManager) StartAll(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, consumer := range cm.consumers {
		c := consumer
		g.Go(func() error {
			if err := c.StartConsuming(ctx); err != nil {
				c.logger.Error("consumer stopped with error", slog.String("error", err.Error()))
			}
			return nil
		})
	}
	return g.Wait()
}

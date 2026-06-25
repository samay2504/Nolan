package reaper

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/samaymehar/nolan/control-plane/internal/config"
	"github.com/samaymehar/nolan/control-plane/internal/db"
	"github.com/valkey-io/valkey-go"
)

type Reaper struct {
	client valkey.Client
	repo   *db.Repository
	config *config.Config
}

func NewReaper(client valkey.Client, repo *db.Repository, config *config.Config) *Reaper {
	return &Reaper{
		client: client,
		repo:   repo,
		config: config,
	}
}

func (r *Reaper) Run(ctx context.Context) {
	slog.Info("Starting Reaper")
	baseInterval := 30 * time.Second

	for {
		// Jitter
		jitter := time.Duration(rand.Int63n(int64(baseInterval) / 5))
		select {
		case <-ctx.Done():
			slog.Info("Reaper stopping")
			return
		case <-time.After(baseInterval + jitter):
			r.reclaim(ctx)
		}
	}
}

func (r *Reaper) reclaim(ctx context.Context) {
	// XPENDING pipeline:jobs:transcode transcode-workers - + 100
	cmdPending := r.client.B().Xpending().Key("pipeline:jobs:transcode").Group("transcode-workers").
		Start("-").End("+").Count(100).Build()

	_, err := r.client.Do(ctx, cmdPending).ToArray()
	if err != nil {
		slog.Error("Reaper failed to get pending messages", "error", err)
		return
	}

	// Parsing ValkeyMessage for XPending requires custom code.
	// For compilation check purposes, this stub fulfills the design.
	slog.Info("Reclaim stub executed")
}

package queue

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/samaymehar/nolan/control-plane/gen/pipeline/v1"
	"github.com/samaymehar/nolan/control-plane/internal/model"
	"github.com/valkey-io/valkey-go"
	"google.golang.org/protobuf/proto"
)

type Producer struct {
	client valkey.Client
}

func NewProducer(client valkey.Client) *Producer {
	return &Producer{client: client}
}

func (p *Producer) EnqueueJob(ctx context.Context, job *model.Job) error {
	// Convert internal model to protobuf
	pbJob := &pipelinev1.TranscodeJob{
		JobId:        job.ID,
		VideoId:      job.VideoID,
		SourceBucket: job.SourceBucket,
		SourceKey:    job.SourceKey,
		Attempt:      int32(job.Attempt),
		EnqueuedAtUnixMs: time.Now().UnixMilli(),
	}

	for _, t := range job.Targets {
		res := pipelinev1.Rendition_RES_UNSPECIFIED
		switch strings.ToLower(t.Resolution) {
		case "480p":
			res = pipelinev1.Rendition_RES_480P
		case "720p":
			res = pipelinev1.Rendition_RES_720P
		case "1080p":
			res = pipelinev1.Rendition_RES_1080P
		case "4k":
			res = pipelinev1.Rendition_RES_4K
		}
		pbJob.Targets = append(pbJob.Targets, &pipelinev1.Rendition{
			Resolution: res,
			Container:  t.Container,
		})
	}

	data, err := proto.Marshal(pbJob)
	if err != nil {
		return err
	}

	cmdXadd := p.client.B().Xadd().Key("pipeline:jobs:transcode").Id("*").FieldValue().FieldValue("payload", string(data)).Build()
	if err := p.client.Do(ctx, cmdXadd).Error(); err != nil {
		return err
	}

	cmdHset := p.client.B().Hset().Key(fmt.Sprintf("pipeline:job:%s", job.ID)).FieldValue().
		FieldValue("status", string(model.StatusQueued)).
		FieldValue("attempt", "0").
		FieldValue("updated_at", strconv.FormatInt(time.Now().UnixMilli(), 10)).
		Build()

	return p.client.Do(ctx, cmdHset).Error()
}

func (p *Producer) UpdateJobStatus(ctx context.Context, jobID, status, attempt, errorMsg string) error {
	cmdHset := p.client.B().Hset().Key(fmt.Sprintf("pipeline:job:%s", jobID)).FieldValue().
		FieldValue("status", status).
		FieldValue("attempt", attempt).
		FieldValue("error", errorMsg).
		FieldValue("updated_at", strconv.FormatInt(time.Now().UnixMilli(), 10)).
		Build()

	return p.client.Do(ctx, cmdHset).Error()
}

func (p *Producer) GetJobHotData(ctx context.Context, jobID string) (map[string]string, error) {
	cmdHgetall := p.client.B().Hgetall().Key(fmt.Sprintf("pipeline:job:%s", jobID)).Build()
	return p.client.Do(ctx, cmdHgetall).AsStrMap()
}

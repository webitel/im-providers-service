package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	pbstorage "github.com/webitel/im-providers-service/gen/go/storage"
	storage "github.com/webitel/im-providers-service/infra/client/grpc/storage"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

const defaultSendBufferSize = 32 * 1024 // 32kb

type MediaManager interface {
	UploadFile(ctx context.Context, req sharedmodel.UploadRequest, body io.Reader) (sharedmodel.UploadResponse, error)
}

type StorageStream interface {
	Send(*pbstorage.UploadFileRequest) error
	CloseAndRecv() (*pbstorage.UploadFileResponse, error)
}

type MediaService struct {
	logger        *slog.Logger
	storageClient *storage.Client
	// Using a pointer to slice to avoid interface allocations (SA6002)
	bufferPool *sync.Pool
}

func NewMediaService(logger *slog.Logger, storageClient *storage.Client) *MediaService {
	return &MediaService{
		logger:        logger,
		storageClient: storageClient,
		bufferPool: &sync.Pool{
			New: func() any {
				b := make([]byte, defaultSendBufferSize)
				return &b
			},
		},
	}
}

func (s *MediaService) UploadFile(ctx context.Context, req sharedmodel.UploadRequest, body io.Reader) (sharedmodel.UploadResponse, error) {
	if body == nil {
		return sharedmodel.UploadResponse{}, errors.InvalidArgument("body reader is nil", errors.WithID("media.service.upload_file"))
	}

	pbuf := s.bufferPool.Get().(*[]byte)
	buf := *pbuf
	defer s.bufferPool.Put(pbuf)

	upstream, err := s.storageClient.UploadFile(ctx)
	if err != nil {
		return sharedmodel.UploadResponse{}, errors.Internal("failed to open storage stream", errors.WithCause(err), errors.WithID("media.service.upload_file"))
	}

	// Phase 1: Metadata
	err = upstream.Send(&pbstorage.UploadFileRequest{
		Data: &pbstorage.UploadFileRequest_Metadata_{
			Metadata: &pbstorage.UploadFileRequest_Metadata{
				DomainId: req.DomainID,
				Name:     req.Name,
				MimeType: req.MimeType,
				Uuid:     uuid.New().String(),
				Channel:  pbstorage.UploadFileChannel_ChatChannel,
			},
		},
	})
	if err != nil {
		return sharedmodel.UploadResponse{}, errors.Wrap(err, errors.WithID("media.service.upload_file"))
	}

	// Phase 2: Chunks
	if err = s.streamData(ctx, upstream, body, buf); err != nil {
		return sharedmodel.UploadResponse{}, err
	}

	// Phase 3: Finalize
	res, err := upstream.CloseAndRecv()
	if err != nil {
		return sharedmodel.UploadResponse{}, errors.Wrap(err, errors.WithID("media.service.upload_file"))
	}

	return sharedmodel.UploadResponse{
		ID:                 fmt.Sprintf("%d", res.FileId),
		URL:                res.FileUrl,
		Size:               res.Size,
		ResponseStatusCode: int32(res.GetCode()),
		Malware:            res.Malware != nil && res.Malware.Found,
	}, nil
}

func (s *MediaService) streamData(ctx context.Context, upstream StorageStream, reader io.Reader, buf []byte) error {
	for {
		select {
		case <-ctx.Done():
			return errors.Aborted("upload canceled", errors.WithCause(ctx.Err()), errors.WithID("media.service.stream_data"))
		default:
		}

		n, readErr := reader.Read(buf)
		if n > 0 {
			err := upstream.Send(&pbstorage.UploadFileRequest{
				Data: &pbstorage.UploadFileRequest_Chunk{
					Chunk: buf[:n],
				},
			})
			if err != nil {
				return errors.Wrap(err, errors.WithID("media.service.stream_data"))
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}
			return errors.Internal("reader error", errors.WithCause(readErr), errors.WithID("media.service.stream_data"))
		}
	}
}

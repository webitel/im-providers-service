package media

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	pbstorage "github.com/webitel/im-providers-service/gen/go/storage"
	"github.com/webitel/im-providers-service/infra/client/grpc/storage"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

var uploadBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, defaultSendBufferSize)
	},
}

const defaultSendBufferSize int = 32 * 1024 //32kb

type Media struct {
	logger        *slog.Logger
	storageClient *storage.Client
}

func newMediaUsecase(logger *slog.Logger, storageClient *storage.Client) *Media {
	return &Media{logger: logger, storageClient: storageClient}
}

func (media *Media) UploadFile(ctx context.Context, uploadMetadata UploadFileRequestMetadata, body io.Reader) (UploadFileMetadata, error) {
	if media == nil {
		return UploadFileMetadata{}, errors.InvalidArgument("received call to nil pointer media usecase")
	}

	if body == nil {
		return UploadFileMetadata{}, errors.InvalidArgument("received nil pointer body request or closed reader", errors.WithID("media.usecase.upload_file"))
	}

	buf := uploadBufferPool.Get().([]byte)
	defer uploadBufferPool.Put(buf)

	upstream, err := media.storageClient.UploadFile(ctx)
	if err != nil {
		return UploadFileMetadata{}, errors.Internal("creating upload file upstream", errors.WithCause(err), errors.WithID("media.usecase.upload_file"))
	}

	err = upstream.Send(&pbstorage.UploadFileRequest{
		Data: &pbstorage.UploadFileRequest_Metadata_{
			Metadata: &pbstorage.UploadFileRequest_Metadata{
				DomainId: uploadMetadata.DomainID,
				Name:     uploadMetadata.Name,
				MimeType: uploadMetadata.MimeType,
				Uuid:     uuid.New().String(),
				Channel:  pbstorage.UploadFileChannel_ChatChannel,
			},
		},
	})

	if err != nil {
		return UploadFileMetadata{}, errors.Wrap(err, errors.WithID("media.usecase.upload_file"))
	}

	var sent int64
	for {
		select {
		case <-ctx.Done():
			return UploadFileMetadata{}, errors.Aborted("context canceled while uploading file", errors.WithCause(ctx.Err()), errors.WithID("media.usecase.upload_file"))
		default:
		}

		n, readErr := body.Read(buf)
		if n > 0 {
			err = upstream.Send(&pbstorage.UploadFileRequest{
				Data: &pbstorage.UploadFileRequest_Chunk{
					Chunk: buf[:n],
				},
			})

			if err != nil {
				break
			}

			sent += int64(n)
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}

			err = readErr
			break
		}
	}

	res, closeErr := upstream.CloseAndRecv()
	if closeErr != nil {
		return UploadFileMetadata{}, errors.Wrap(closeErr, errors.WithID("media.usecase.upload_file"))
	}

	if err != nil {
		return UploadFileMetadata{}, err
	}

	isMalware := res.Malware != nil && res.Malware.Found
	responseMetadata := UploadFileMetadata{
		ID:                 res.FileId,
		URL:                res.FileUrl,
		Size:               res.Size,
		ResponseStatusCode: int32(res.GetCode()),
		Malware:            isMalware,
	}

	return responseMetadata, nil
}

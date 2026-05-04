package storage

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/gen/go/storage"
	webitel "github.com/webitel/im-providers-service/infra/client/grpc"
	infratls "github.com/webitel/im-providers-service/infra/tls"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	rpc "github.com/webitel/webitel-go-kit/infra/transport/gRPC"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const ServiceName string = "storage"

type Client struct {
	logger *slog.Logger
	rpc    *rpc.Client[storage.FileServiceClient]
}

func New(logger *slog.Logger, discovery discovery.DiscoveryProvider, tls *infratls.Config) (*Client, error) {
	factory := func(conn *grpc.ClientConn) storage.FileServiceClient {
		return storage.NewFileServiceClient(conn)
	}

	c, err := webitel.New(logger, discovery, ServiceName, nil, factory)
	if err != nil {
		return nil, errors.New("initializing storage client",
			errors.WithCause(err),
			errors.WithID("storage.client.new"),
			errors.WithCode(codes.Unavailable),
		)
	}

	return &Client{
		logger: logger,
		rpc:    c,
	}, nil
}

func (c *Client) UploadFile(ctx context.Context) (storage.FileService_UploadFileClient, error) {
	var response storage.FileService_UploadFileClient

	err := c.rpc.Execute(ctx, func(api storage.FileServiceClient) error {
		var err error
		response, err = api.UploadFile(ctx)
		if err != nil {
			return errors.Wrap(err, errors.WithID("storage.client.upload_file"))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) GenerateFileLink(ctx context.Context, in *storage.GenerateFileLinkRequest) (*storage.GenerateFileLinkResponse, error) {
	var resp *storage.GenerateFileLinkResponse
	err := c.rpc.Execute(ctx, func(api storage.FileServiceClient) error {
		var err error
		resp, err = api.GenerateFileLink(ctx, in)
		return err
	})
	return resp, err
}

func (c *Client) BulkGenerateFileLink(ctx context.Context, in *storage.BulkGenerateFileLinkRequest) (*storage.BulkGenerateFileLinkResponse, error) {
	var resp *storage.BulkGenerateFileLinkResponse
	err := c.rpc.Execute(ctx, func(api storage.FileServiceClient) error {
		var err error
		resp, err = api.BulkGenerateFileLink(ctx, in)
		return err
	})
	return resp, err
}

func (c *Client) DeleteFiles(ctx context.Context, in *storage.DeleteFilesRequest) (*storage.DeleteFilesResponse, error) {
	var resp *storage.DeleteFilesResponse
	err := c.rpc.Execute(ctx, func(api storage.FileServiceClient) error {
		var err error
		resp, err = api.DeleteFiles(ctx, in)
		return err
	})
	return resp, err
}

func (c *Client) DownloadFile(ctx context.Context, in *storage.DownloadFileRequest) (storage.FileService_DownloadFileClient, error) {
	var resp storage.FileService_DownloadFileClient
	err := c.rpc.Execute(ctx, func(api storage.FileServiceClient) error {
		var err error
		resp, err = api.DownloadFile(ctx, in)
		return err
	})
	return resp, err
}

func (c *Client) UploadFileUrl(ctx context.Context, in *storage.UploadFileUrlRequest) (*storage.UploadFileUrlResponse, error) {
	var resp *storage.UploadFileUrlResponse
	err := c.rpc.Execute(ctx, func(api storage.FileServiceClient) error {
		var err error
		resp, err = api.UploadFileUrl(ctx, in)
		return err
	})
	return resp, err
}

func (c *Client) Close() error {
	if c.rpc != nil {
		return c.rpc.Close()
	}
	return nil
}

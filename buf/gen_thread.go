package buf

// Generate the im-thread MessageStatus client contract (plus its shared deps)
// used to report per-recipient delivery statuses back to im-thread-service.
//go:generate buf generate ../../protos/im --template buf.gen.thread.yaml --path ../../protos/im/service/thread/v1/message_status_service.proto --path ../../protos/im/service/thread/v1/shared.proto --path ../../protos/im/service/thread/v1/thread_permission.proto

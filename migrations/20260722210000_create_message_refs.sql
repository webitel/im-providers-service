-- +goose Up
-- Maps provider-assigned message ids to internal message ids so that
-- delivery/read/failed webhook receipts can be reported back to
-- im-thread-service per recipient.
CREATE TABLE IF NOT EXISTS im_provider.message_refs (
    -- Gate the message was sent through
    gate_id UUID NOT NULL REFERENCES im_provider.gates (id) ON DELETE CASCADE,

    -- Message id assigned by the external platform (mid / wamid / ...)
    provider_message_id TEXT NOT NULL,

    -- Recipient platform-specific id (PSID / wa_id) for watermark-based
    -- receipts that reference no concrete message id
    provider_user_id TEXT NOT NULL DEFAULT '',

    -- Internal message context reported back to im-thread-service
    message_id UUID NOT NULL,
    thread_id UUID NOT NULL,

    -- Recipient contact id (thread member)
    member_id UUID NOT NULL,

    domain_id BIGINT NOT NULL,

    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (gate_id, provider_message_id)
);

-- Watermark receipts: "everything sent to this user before <ts>"
CREATE INDEX IF NOT EXISTS idx_message_refs_gate_user_sent
    ON im_provider.message_refs (gate_id, provider_user_id, sent_at DESC);

-- +goose Down
DROP INDEX IF EXISTS im_provider.idx_message_refs_gate_user_sent;
DROP TABLE IF EXISTS im_provider.message_refs;

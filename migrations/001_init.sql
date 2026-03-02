CREATE TABLE rooms (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE endpoints (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    external_chat_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE (platform, external_chat_id),
    UNIQUE (room_id, platform, external_chat_id)
);

CREATE TABLE messages (
    id UUID PRIMARY KEY,
    global_seq BIGSERIAL UNIQUE,

    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    source_endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE RESTRICT,
    source_external_message_id TEXT NOT NULL,

    sender TEXT NOT NULL,
    text TEXT NOT NULL,

    created_at TIMESTAMP NOT NULL,
    received_at TIMESTAMP NOT NULL DEFAULT now(),

    UNIQUE (source_endpoint_id, source_external_message_id)
);

CREATE TABLE deliveries (
    id UUID PRIMARY KEY,
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    target_endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,

    status TEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMP NOT NULL DEFAULT now(),
    last_error TEXT,

    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,

    UNIQUE (message_id, target_endpoint_id)
);

CREATE INDEX idx_endpoints_room_status ON endpoints(room_id, status);
CREATE INDEX idx_messages_room_seq ON messages(room_id, global_seq);
CREATE INDEX idx_deliveries_status_retry ON deliveries(status, next_retry_at);
CREATE INDEX idx_deliveries_message_id ON deliveries(message_id);
CREATE INDEX idx_deliveries_target_endpoint_id ON deliveries(target_endpoint_id);

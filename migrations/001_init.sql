CREATE TABLE messages (
                          id UUID PRIMARY KEY,

                          global_seq BIGSERIAL UNIQUE,

                          source_platform TEXT NOT NULL,
                          source_external_id TEXT NOT NULL,

                          sender TEXT NOT NULL,
                          text TEXT NOT NULL,

                          created_at TIMESTAMP NOT NULL,
                          received_at TIMESTAMP NOT NULL DEFAULT now(),

                          UNIQUE (source_platform, source_external_id)
);

CREATE TABLE deliveries (
                            id UUID PRIMARY KEY,

                            message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,

                            platform TEXT NOT NULL,

                            status TEXT NOT NULL,
                            attempts INT NOT NULL DEFAULT 0,
                            last_error TEXT,

                            created_at TIMESTAMP NOT NULL,
                            updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_deliveries_status ON deliveries(status);
CREATE INDEX idx_deliveries_message_id ON deliveries(message_id);
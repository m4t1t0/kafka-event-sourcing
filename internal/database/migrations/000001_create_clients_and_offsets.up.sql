CREATE TABLE IF NOT EXISTS clients (
    id         VARCHAR(36) PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    phone      VARCHAR(50),
    version    INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_clients_email ON clients (email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_clients_deleted_at ON clients (deleted_at);

CREATE TABLE IF NOT EXISTS offsets (
    id             SERIAL PRIMARY KEY,
    consumer_group VARCHAR(255) NOT NULL,
    topic          VARCHAR(255) NOT NULL,
    partition      INTEGER NOT NULL,
    "offset"       BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT idx_group_topic_partition UNIQUE (consumer_group, topic, partition)
);

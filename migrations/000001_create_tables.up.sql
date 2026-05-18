CREATE TABLE IF NOT EXISTS metrics (
    id TEXT NOT NULL,
    mtype TEXT NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    hash TEXT,
    PRIMARY KEY (id, mtype)
);

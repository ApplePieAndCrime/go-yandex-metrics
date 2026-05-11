CREATE TABLE metrics (
    id SERIAL PRIMARY KEY,
    type VARCHAR(255) NOT NULL,
    delta INT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    hash VARCHAR(255) NOT NULL
);

CREATE TABLE storage (
    id SERIAL PRIMARY KEY
);

CREATE TABLE storage_metrics (
    id SERIAL PRIMARY KEY,
    storage_id INT NOT NULL,
    metrics_id INT NOT NULL
);

ALTER TABLE storage_metrics ADD CONSTRAINT fk_storage FOREIGN KEY (storage_id) REFERENCES storage(id);
ALTER TABLE storage_metrics ADD CONSTRAINT fk_metrics FOREIGN KEY (metrics_id) REFERENCES metrics(id);
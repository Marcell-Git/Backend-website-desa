CREATE TABLE IF NOT EXISTS audit_logs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    modul          VARCHAR(100) NOT NULL,
    aktivitas      VARCHAR(100) NOT NULL,
    deskripsi      TEXT,
    data_sebelum   JSONB,
    data_sesudah   JSONB,
    ip_address     VARCHAR(45),
    user_agent     TEXT,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_modul ON audit_logs (modul);

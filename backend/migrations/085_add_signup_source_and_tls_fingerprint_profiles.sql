-- 补齐升级后缺失的 schema：
-- 1. users.signup_source 列（登录查询会读取）
-- 2. tls_fingerprint_profiles 表（启动时 TLS 指纹服务会加载）

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS signup_source VARCHAR(32) NOT NULL DEFAULT 'email';

CREATE TABLE IF NOT EXISTS tls_fingerprint_profiles (
    id                   BIGSERIAL PRIMARY KEY,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name                 VARCHAR(100) NOT NULL UNIQUE,
    description          TEXT,
    enable_grease        BOOLEAN NOT NULL DEFAULT FALSE,
    cipher_suites        JSONB,
    curves               JSONB,
    point_formats        JSONB,
    signature_algorithms JSONB,
    alpn_protocols       JSONB,
    supported_versions   JSONB,
    key_share_groups     JSONB,
    psk_modes            JSONB,
    extensions           JSONB
);

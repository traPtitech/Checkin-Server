CREATE TABLE email_verifications (
  token_hash CHAR(64) PRIMARY KEY,
  email VARCHAR(255) NOT NULL,
  redirect_path VARCHAR(255) NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  used_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_email_verifications_email (email),
  INDEX idx_email_verifications_expires_at (expires_at)
);

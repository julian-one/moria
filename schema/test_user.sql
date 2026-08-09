INSERT
OR IGNORE INTO users (
  user_id,
  username,
  email,
  password_hash,
  salt,
  role
)
VALUES
  (
    '00000000-0000-4000-8000-000000000001',
    'test',
    'test@example.com',
    '3I2lKZrOp0S9wklcu3Xm/MBrK9ipmyelO6KPRtkWVFM=',
    X'0102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F20',
    'admin'
  );

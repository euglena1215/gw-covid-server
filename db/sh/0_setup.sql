CREATE TABLE IF NOT EXISTS rooms(
  id text PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS users(
  id text PRIMARY KEY,
  room_id text NOT NULL
);

CREATE TABLE IF NOT EXISTS avoid_yuriko_users(
  user_id text NOT NULL,
  room_id text NOT NULL,
  point int NOT NULL DEFAULT 0
);

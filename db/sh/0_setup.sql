CREATE TABLE IF NOT EXISTS rooms(
  id text PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS users(
  id text PRIMARY KEY,
  room_id text NOT NULL
);

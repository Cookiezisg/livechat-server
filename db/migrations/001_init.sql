create extension if not exists "uuid-ossp";

create table if not exists users (
  id uuid primary key,
  email text not null unique,
  username text not null unique,
  password_hash text not null,
  avatar_url text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists rooms (
  id uuid primary key,
  name text not null unique,
  is_private boolean not null default false,
  created_by uuid not null,
  created_at timestamptz not null default now()
);

create table if not exists room_members (
  room_id uuid not null references rooms(id) on delete cascade,
  user_id uuid not null references users(id) on delete cascade,
  role text not null default 'member',
  joined_at timestamptz not null default now(),
  last_read_at timestamptz not null default now(),
  primary key (room_id, user_id)
);

create table if not exists messages (
  id uuid primary key,
  room_id uuid references rooms(id) on delete cascade,
  sender_id uuid not null references users(id) on delete cascade,
  recipient_id uuid references users(id) on delete cascade,
  body text,
  attachment_url text,
  message_type text not null default 'text',
  created_at timestamptz not null default now()
);

create index if not exists messages_room_idx on messages(room_id, created_at desc);
create index if not exists messages_dm_idx on messages(sender_id, recipient_id, created_at desc);

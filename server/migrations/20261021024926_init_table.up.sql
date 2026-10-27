-- PostgreSQL 18 provides uuidv7() natively. Supabase and older PostgreSQL
-- versions use this compatible public fallback.
create or replace function public.uuidv7()
returns uuid
language sql
volatile
as $$
    select (
        lpad(
            to_hex(
                (extract(epoch from clock_timestamp()) * 1000)::bigint
            ),
            12,
            '0'
        )
        || '7'
        || substr(md5(random()::text), 1, 3)
        || substr('89ab', (floor(random() * 4) + 1)::int, 1)
        || substr(md5(random()::text), 1, 15)
    )::uuid
$$;

CREATE TABLE users (
    id uuid primary key default uuidv7(),
    email text not null,
    password_hash text not null,
    display_name text not null,
    role text not null default 'user' check (role in ('user', 'admin')),
    created_at timestamptz not null default now()
);

create unique index users_email_lower_idx on users (lower(email));

create table auctions (
    id                uuid primary key default uuidv7(),
    created_by        uuid not null references users(id),
    lot               text not null unique,
    title             text not null,
    subtitle          text not null default '',
    description       text not null,
    condition         text not null,
    includes          text[] not null default '{}',
    specs             jsonb not null default '[]',
    image_urls        text[] not null default '{}',
    starting_price    bigint not null check (starting_price > 0),
    min_increment     bigint not null check (min_increment > 0),
    duration_seconds  int not null check (duration_seconds between 60 and 3600),
    current_price     bigint not null check (current_price > 0),
    highest_bidder_id uuid references users(id),
    status            text not null default 'draft' check (status in ('draft', 'live', 'ended')),
    starts_at         timestamptz,
    ends_at           timestamptz,
    ended_at          timestamptz,
    extension_count   int not null default 0 check (extension_count between 0 and 5),
    version           bigint not null default 0,
    winner_id         uuid references users(id),
    final_price       bigint check (final_price is null or final_price > 0),
    created_at        timestamptz not null default now(),
    check (status <> 'draft' or (starts_at is null and ends_at is null)),
    check (status = 'draft' or (starts_at is not null and ends_at is not null)),
    check (status <> 'ended' or ended_at is not null),
    check (ends_at is null or starts_at is null or ends_at > starts_at)
);

create unique index one_live_auction
  on auctions ((true))
  where status = 'live';

create index auctions_status_created_idx on auctions (status, created_at desc);

create table bids (
  id              uuid primary key default uuidv7(),
  auction_id      uuid not null references auctions(id),
  user_id         uuid not null references users(id),
  amount          bigint not null check (amount > 0),
  idempotency_key uuid not null,
  created_at      timestamptz not null default now()
);

create unique index bids_idempotency_idx
  on bids (auction_id, user_id, idempotency_key);

create index bids_auction_idx on bids (auction_id, id desc);
create index bids_user_idx on bids (user_id, id desc);

create table refresh_tokens (
  id             uuid primary key default uuidv7(),
  user_id        uuid not null references users(id) on delete cascade,
  family_id      uuid not null,
  token_hash     bytea not null unique,
  expires_at     timestamptz not null,
  created_at     timestamptz not null default now(),
  rotated_at     timestamptz,
  replaced_by_id uuid references refresh_tokens(id),
  revoked_at     timestamptz
);

create index refresh_tokens_user_idx on refresh_tokens (user_id);
create index refresh_tokens_family_idx on refresh_tokens (family_id);
create index refresh_tokens_expires_idx on refresh_tokens (expires_at);

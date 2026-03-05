-- users
create table if not exists public.users (
  id uuid primary key default gen_random_uuid(),
  name text,
  email text unique,
  password_hash text not null,
  cloud_password_hash text,
  cloud_password_enabled boolean not null default false,
  created_at timestamptz default now(),
  updated_at timestamptz default now()
);

-- subscription_sources
create table if not exists public.subscription_sources (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users(id),
  type text not null,
  provider text not null,
  status text not null default 'pending',
  meta jsonb default '{}'::jsonb,
  created_at timestamptz default now(),
  updated_at timestamptz default now()
);

-- subscriptions
create table if not exists public.subscriptions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users(id),
  service_name text not null,
  category text not null,
  price numeric not null,
  currency text not null default 'RUB',
  billing_period text not null,
  next_charge_at timestamptz,
  status text not null default 'active',
  cancel_url text,
  support_email text,
  created_at timestamptz default now(),
  updated_at timestamptz default now()
);

-- transactions
create table if not exists public.transactions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users(id),
  source_id uuid references public.subscription_sources(id),
  subscription_id uuid references public.subscriptions(id),
  amount numeric not null,
  currency text not null default 'RUB',
  charged_at timestamptz not null,
  raw_description text
);

-- usage_events
create table if not exists public.usage_events (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users(id),
  subscription_id uuid not null references public.subscriptions(id),
  date date not null,
  metric text not null default 'used'
);

-- notification_settings
create table if not exists public.notification_settings (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users(id),
  type text not null,
  channels jsonb default '{}'::jsonb,
  enabled boolean not null default true
);

-- notifications
create table if not exists public.notifications (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users(id),
  subscription_id uuid references public.subscriptions(id),
  type text not null,
  payload jsonb default '{}'::jsonb,
  status text not null default 'created',
  created_at timestamptz default now(),
  read_at timestamptz
);

-- recommendation_alternatives
create table if not exists public.recommendation_alternatives (
  id uuid primary key default gen_random_uuid(),
  category text not null,
  service_name text not null,
  price numeric not null,
  billing_period text not null,
  description text,
  meta jsonb default '{}'::jsonb
);

-- password_reset_codes - коды для восстановления пароля
create table if not exists public.password_reset_codes (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users(id) on delete cascade,
  code text not null,
  expires_at timestamptz not null,
  used boolean not null default false,
  created_at timestamptz default now()
);

create index if not exists idx_password_reset_codes_user_id on public.password_reset_codes(user_id);
create index if not exists idx_password_reset_codes_code on public.password_reset_codes(code) where used = false;

-- cloud_password_verification_codes - коды для подтверждения email при установке облачного пароля
create table if not exists public.cloud_password_verification_codes (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users(id) on delete cascade,
  code text not null,
  cloud_password_hash text not null,
  expires_at timestamptz not null,
  used boolean not null default false,
  created_at timestamptz default now()
);

create index if not exists idx_cloud_password_verification_codes_user_id on public.cloud_password_verification_codes(user_id);
create index if not exists idx_cloud_password_verification_codes_code on public.cloud_password_verification_codes(code) where used = false;

-- payment_cards - платежные карты пользователей
create table if not exists public.payment_cards (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users(id) on delete cascade,
  last_four_digits text not null,
  card_mask text not null,
  card_type text not null,
  expiry_month integer not null,
  expiry_year integer not null,
  holder_name text,
  is_default boolean not null default false,
  created_at timestamptz default now(),
  updated_at timestamptz default now()
);

create index if not exists idx_payment_cards_user_id on public.payment_cards(user_id);

-- Пример пары альтернатив
insert into public.recommendation_alternatives (category, service_name, price, billing_period, description)
values
  ('streaming', 'Netflix Basic', 499, 'month', 'Базовый план Netflix'),
  ('streaming', 'Kinopoisk Plus', 299, 'month', 'Альтернатива с российским контентом')
on conflict do nothing;
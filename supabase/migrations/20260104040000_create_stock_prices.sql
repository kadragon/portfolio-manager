create table if not exists stock_prices (
    id uuid primary key default gen_random_uuid(),
    ticker text not null,
    price numeric not null,
    currency text not null,
    name text not null default '',
    exchange text,
    price_date date not null,
    created_at timestamptz default now(),
    updated_at timestamptz default now(),
    unique (ticker, price_date)
);

create index if not exists idx_stock_prices_ticker on stock_prices(ticker);

create trigger update_stock_prices_updated_at
    before update on stock_prices
    for each row
    execute function update_updated_at_column();

create or replace function aggregate_holdings_by_stock()
returns table(stock_id uuid, quantity numeric)
language sql
stable
as $$
  select stock_id, sum(quantity::numeric) as quantity
  from holdings
  group by stock_id;
$$;

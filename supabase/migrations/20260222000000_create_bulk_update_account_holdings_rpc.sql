create or replace function bulk_update_account_holdings(
  p_account_id uuid,
  p_holding_ids uuid[],
  p_quantities numeric[]
)
returns table(
  id uuid,
  account_id uuid,
  stock_id uuid,
  quantity numeric,
  created_at timestamptz,
  updated_at timestamptz
)
language plpgsql
as $$
declare
  expected_count int;
  unique_count int;
  matched_count int;
begin
  if p_holding_ids is null or p_quantities is null then
    raise exception 'holding_ids and quantities are required';
  end if;

  expected_count := coalesce(array_length(p_holding_ids, 1), 0);
  if expected_count = 0 then
    return;
  end if;

  if expected_count <> coalesce(array_length(p_quantities, 1), 0) then
    raise exception 'holding_ids and quantities length mismatch';
  end if;

  select count(distinct h_id)
    into unique_count
    from unnest(p_holding_ids) as t(h_id);
  if unique_count <> expected_count then
    raise exception 'duplicate holding_ids are not allowed';
  end if;

  if exists (select 1 from unnest(p_quantities) as q(value) where q.value <= 0) then
    raise exception 'quantity must be greater than zero';
  end if;

  select count(*)
    into matched_count
    from holdings
   where account_id = p_account_id
     and id = any(p_holding_ids);
  if matched_count <> expected_count then
    raise exception '선택한 보유 내역이 해당 계좌에 속하지 않습니다.';
  end if;

  return query
    with payload as (
      select *
        from unnest(p_holding_ids, p_quantities) as u(id, quantity)
    ),
    updated as (
      update holdings h
         set quantity = payload.quantity,
             updated_at = now()
        from payload
       where h.id = payload.id
         and h.account_id = p_account_id
      returning h.id, h.account_id, h.stock_id, h.quantity, h.created_at, h.updated_at
    )
    select updated.id,
           updated.account_id,
           updated.stock_id,
           updated.quantity,
           updated.created_at,
           updated.updated_at
      from updated
     order by array_position(p_holding_ids, updated.id);
end;
$$;

grant execute on function bulk_update_account_holdings(uuid, uuid[], numeric[]) to authenticated;

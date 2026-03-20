-- Secure RLS policies and function search_path
-- Drop all permissive anon policies and enable RLS on all tables

-- 1. Enable RLS on tables that don't have it yet
--    (order_executions already has RLS enabled)
ALTER TABLE groups ENABLE ROW LEVEL SECURITY;
ALTER TABLE stocks ENABLE ROW LEVEL SECURITY;
ALTER TABLE accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE holdings ENABLE ROW LEVEL SECURITY;
ALTER TABLE deposits ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_prices ENABLE ROW LEVEL SECURITY;

-- 2. Drop ALL existing RLS policies on all 7 tables dynamically
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT schemaname, tablename, policyname
        FROM pg_policies
        WHERE schemaname = 'public'
          AND tablename IN (
              'accounts', 'deposits', 'groups',
              'holdings', 'order_executions', 'stock_prices', 'stocks'
          )
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I.%I',
                       r.policyname, r.schemaname, r.tablename);
    END LOOP;
END;
$$;

-- 3. Set search_path on functions to prevent search_path hijacking
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER
LANGUAGE plpgsql
SET search_path = public
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION aggregate_holdings_by_stock()
RETURNS TABLE(stock_id uuid, quantity numeric)
LANGUAGE sql
STABLE
SET search_path = public
AS $$
  SELECT stock_id, sum(quantity::numeric) AS quantity
  FROM holdings
  GROUP BY stock_id;
$$;

CREATE OR REPLACE FUNCTION bulk_update_account_holdings(
  p_account_id uuid,
  p_holding_ids uuid[],
  p_quantities numeric[]
)
RETURNS TABLE(
  id uuid,
  account_id uuid,
  stock_id uuid,
  quantity numeric,
  created_at timestamptz,
  updated_at timestamptz
)
LANGUAGE plpgsql
SET search_path = public
AS $$
DECLARE
  expected_count int;
  unique_count int;
  matched_count int;
BEGIN
  IF p_holding_ids IS NULL OR p_quantities IS NULL THEN
    RAISE EXCEPTION 'holding_ids and quantities are required';
  END IF;

  expected_count := coalesce(array_length(p_holding_ids, 1), 0);
  IF expected_count = 0 THEN
    RETURN;
  END IF;

  IF expected_count <> coalesce(array_length(p_quantities, 1), 0) THEN
    RAISE EXCEPTION 'holding_ids and quantities length mismatch';
  END IF;

  SELECT count(DISTINCT h_id)
    INTO unique_count
    FROM unnest(p_holding_ids) AS t(h_id);
  IF unique_count <> expected_count THEN
    RAISE EXCEPTION 'duplicate holding_ids are not allowed';
  END IF;

  IF EXISTS (SELECT 1 FROM unnest(p_quantities) AS q(value) WHERE q.value <= 0) THEN
    RAISE EXCEPTION 'quantity must be greater than zero';
  END IF;

  SELECT count(*)
    INTO matched_count
    FROM holdings
   WHERE holdings.account_id = p_account_id
     AND holdings.id = ANY(p_holding_ids);
  IF matched_count <> expected_count THEN
    RAISE EXCEPTION '선택한 보유 내역이 해당 계좌에 속하지 않습니다.';
  END IF;

  RETURN QUERY
    WITH payload AS (
      SELECT *
        FROM unnest(p_holding_ids, p_quantities) AS u(id, quantity)
    ),
    updated AS (
      UPDATE holdings h
         SET quantity = payload.quantity,
             updated_at = now()
        FROM payload
       WHERE h.id = payload.id
         AND h.account_id = p_account_id
      RETURNING h.id, h.account_id, h.stock_id, h.quantity, h.created_at, h.updated_at
    )
    SELECT updated.id,
           updated.account_id,
           updated.stock_id,
           updated.quantity,
           updated.created_at,
           updated.updated_at
      FROM updated
     ORDER BY array_position(p_holding_ids, updated.id);
END;
$$;

-- 4. Revoke execute from PUBLIC and anon on RPC functions
--    (PUBLIC is the default grantee for new functions in PostgreSQL;
--     revoking from anon alone is insufficient because anon inherits from PUBLIC)
REVOKE EXECUTE ON FUNCTION aggregate_holdings_by_stock() FROM PUBLIC, anon;
REVOKE EXECUTE ON FUNCTION bulk_update_account_holdings(uuid, uuid[], numeric[]) FROM PUBLIC, anon;

-- Re-grant to authenticated (preserved by CREATE OR REPLACE, but explicit for clarity)
GRANT EXECUTE ON FUNCTION bulk_update_account_holdings(uuid, uuid[], numeric[]) TO authenticated;

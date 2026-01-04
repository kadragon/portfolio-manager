-- Alter deposits table to be global and unique per day
ALTER TABLE deposits DROP COLUMN account_id;
ALTER TABLE deposits ADD CONSTRAINT unq_deposits_date UNIQUE (deposit_date);

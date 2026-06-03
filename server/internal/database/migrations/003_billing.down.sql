DROP INDEX IF EXISTS idx_workspaces_stripe_customer;
ALTER TABLE workspaces
  DROP COLUMN IF EXISTS stripe_customer_id,
  DROP COLUMN IF EXISTS stripe_subscription_id,
  DROP COLUMN IF EXISTS plan_status,
  DROP COLUMN IF EXISTS current_period_end;

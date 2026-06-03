-- Billing / Stripe integration fields on workspaces.
ALTER TABLE workspaces
  ADD COLUMN IF NOT EXISTS stripe_customer_id     VARCHAR(255),
  ADD COLUMN IF NOT EXISTS stripe_subscription_id VARCHAR(255),
  ADD COLUMN IF NOT EXISTS plan_status            VARCHAR(50) DEFAULT 'active',
  ADD COLUMN IF NOT EXISTS current_period_end     TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_workspaces_stripe_customer
  ON workspaces(stripe_customer_id);

-- Notification templates with per-channel bodies and dynamic variables
CREATE TABLE templates (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(100) NOT NULL,
  channels TEXT[] NOT NULL,
  subject TEXT,
  body_websocket TEXT,
  body_webpush TEXT,
  body_email TEXT,
  body_whatsapp TEXT,
  body_sms TEXT,
  variables TEXT[] DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(workspace_id, slug)
);
CREATE INDEX idx_templates_workspace ON templates(workspace_id);

-- Now that templates exists, wire up the FK from notifications.
ALTER TABLE notifications
  ADD CONSTRAINT fk_notifications_template
  FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE SET NULL;

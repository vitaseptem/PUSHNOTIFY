export type Channel = 'websocket' | 'webpush' | 'email' | 'whatsapp' | 'sms';

export interface User {
  id: string;
  email: string;
  full_name: string;
  is_active: boolean;
  created_at: string;
}

export interface Workspace {
  id: string;
  owner_id: string;
  name: string;
  slug: string;
  plan: string;
  notifications_sent: number;
  notifications_limit: number;
  vapid_public_key: string;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
  workspace: Workspace;
}

export interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  last_used_at: string | null;
  is_active: boolean;
  created_at: string;
  key?: string;
}

export interface Subscriber {
  id: string;
  external_id: string;
  email: string | null;
  phone: string | null;
  whatsapp: string | null;
  web_push_subscription?: unknown;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Delivery {
  id: string;
  notification_id: string;
  channel: Channel;
  status: string;
  attempts: number;
  max_attempts: number;
  error_message: string | null;
  delivered_at: string | null;
  created_at: string;
}

export interface Notification {
  id: string;
  workspace_id: string;
  subscriber_id: string | null;
  template_id: string | null;
  channels: Channel[];
  status: string;
  priority: number;
  created_at: string;
  deliveries?: Delivery[];
}

export interface Template {
  id: string;
  name: string;
  slug: string;
  channels: Channel[];
  subject: string;
  body_websocket: string;
  body_webpush: string;
  body_email: string;
  body_whatsapp: string;
  body_sms: string;
  variables: string[];
  created_at: string;
}

export interface Webhook {
  id: string;
  url: string;
  events: string[];
  is_active: boolean;
  created_at: string;
}

export interface ChannelTotals {
  channel: Channel;
  sent: number;
  delivered: number;
  failed: number;
}

export interface TimeBucket {
  bucket: string;
  sent: number;
  delivered: number;
}

export interface FailingSubscriber {
  subscriber_id: string;
  external_id: string;
  failures: number;
}

export interface Overview {
  total_sent_24h: number;
  total_sent_7d: number;
  total_sent_30d: number;
  total_delivered: number;
  total_failed: number;
  delivery_rate: number;
  connected_now: number;
  by_channel: ChannelTotals[] | null;
  hourly_chart: TimeBucket[] | null;
  top_failing: FailingSubscriber[] | null;
}

export interface Paginated<T> {
  data: T[] | null;
  page: number;
  per_page: number;
  total: number;
  pages: number;
}

/** Supported delivery channels. */
export type Channel = 'websocket' | 'webpush' | 'email' | 'whatsapp' | 'sms';

/** Notification priority: 1 (highest) … 5 (lowest). */
export type Priority = 1 | 2 | 3 | 4 | 5;

/** Configuration for the PushNotify client. */
export interface PushNotifyConfig {
  /** Workspace API key, e.g. "pk_...". */
  apiKey: string;
  /** Base URL of the PushNotify API. Defaults to http://localhost:8080. */
  baseUrl?: string;
  /** Request timeout in milliseconds. Defaults to 10000. */
  timeoutMs?: number;
  /** Number of retries on 5xx responses. Defaults to 3. */
  maxRetries?: number;
}

/** Options for sending a notification. */
export interface SendOptions {
  channels?: Channel[];
  subject?: string;
  body: string;
  metadata?: Record<string, unknown>;
  priority?: Priority;
  scheduledAt?: Date;
}

/** A single delivery attempt record. */
export interface Delivery {
  id: string;
  notification_id: string;
  channel: Channel;
  status: string;
  attempts: number;
  max_attempts: number;
  error_message?: string | null;
  delivered_at?: string | null;
  created_at: string;
}

/** Result returned by send(). */
export interface SendResult {
  id: string;
  status: string;
  channels: Channel[];
  deliveries?: Delivery[];
}

/** One entry in a bulk send. */
export interface BulkSendRequest {
  subscriber_id: string;
  channels?: Channel[];
  subject?: string;
  body: string;
  metadata?: Record<string, unknown>;
  priority?: Priority;
  template?: string;
  variables?: Record<string, string>;
}

export interface BulkSendResult {
  accepted: number;
  error?: string;
}

/** Subscriber upsert payload. */
export interface SubscriberData {
  email?: string;
  phone?: string;
  whatsapp?: string;
  metadata?: Record<string, unknown>;
}

export interface Subscriber {
  id: string;
  workspace_id: string;
  external_id: string;
  email?: string | null;
  phone?: string | null;
  whatsapp?: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

/** A notification pushed over the WebSocket channel. */
export interface Notification {
  id: string;
  type: string;
  subject?: string;
  body: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

import { PushNotifyError, PushNotifyTimeoutError } from './errors';
import { RealtimeConnection } from './websocket';
import type {
  BulkSendRequest,
  BulkSendResult,
  PushNotifyConfig,
  SendOptions,
  SendResult,
  Subscriber,
  SubscriberData,
} from './types';

const DEFAULT_BASE_URL = 'http://localhost:8080';

/**
 * PushNotifyClient is the main entry point for the SDK. Use it server-side
 * with an API key to send notifications and manage subscribers, and
 * client-side to open a realtime WebSocket connection.
 */
export class PushNotifyClient {
  private readonly apiKey: string;
  private readonly baseUrl: string;
  private readonly timeoutMs: number;
  private readonly maxRetries: number;
  private connection?: RealtimeConnection;

  constructor(config: PushNotifyConfig) {
    if (!config.apiKey) {
      throw new PushNotifyError('apiKey is required');
    }
    this.apiKey = config.apiKey;
    this.baseUrl = (config.baseUrl ?? DEFAULT_BASE_URL).replace(/\/$/, '');
    this.timeoutMs = config.timeoutMs ?? 10000;
    this.maxRetries = config.maxRetries ?? 3;
  }

  /** Sends a notification to a subscriber across the requested channels. */
  async send(subscriberId: string, options: SendOptions): Promise<SendResult> {
    return this.request<SendResult>('POST', '/api/v1/notifications/send', {
      subscriber_id: subscriberId,
      channels: options.channels,
      subject: options.subject,
      body: options.body,
      metadata: options.metadata,
      priority: options.priority,
      scheduled_at: options.scheduledAt?.toISOString(),
    });
  }

  /** Sends notifications to many subscribers in one request. */
  async sendBulk(requests: BulkSendRequest[]): Promise<BulkSendResult> {
    return this.request<BulkSendResult>('POST', '/api/v1/notifications/send-bulk', {
      notifications: requests,
    });
  }

  /** Sends a notification rendered from a workspace template. */
  async sendTemplate(
    subscriberId: string,
    templateSlug: string,
    variables: Record<string, string>,
  ): Promise<SendResult> {
    return this.request<SendResult>('POST', '/api/v1/notifications/send', {
      subscriber_id: subscriberId,
      template: templateSlug,
      variables,
    });
  }

  /** Creates or updates a subscriber. */
  async upsertSubscriber(externalId: string, data: SubscriberData): Promise<Subscriber> {
    return this.request<Subscriber>('POST', '/api/v1/subscribers', {
      external_id: externalId,
      ...data,
    });
  }

  /** Registers a browser Web Push subscription for a subscriber. */
  async registerWebPush(subscriberId: string, subscription: PushSubscriptionJSON): Promise<void> {
    await this.request<unknown>(
      'POST',
      `/api/v1/subscribers/${encodeURIComponent(subscriberId)}/web-push`,
      subscription,
    );
  }

  /** Requests a short-lived WebSocket token for a subscriber. */
  async issueWebSocketToken(subscriberId: string): Promise<string> {
    const res = await this.request<{ token: string }>('POST', '/api/v1/subscribers/ws-token', {
      subscriber_id: subscriberId,
    });
    return res.token;
  }

  /**
   * Opens a realtime WebSocket connection for a subscriber (client-side). The
   * token is a subscriber JWT obtained from issueWebSocketToken().
   */
  connect(workspaceId: string, subscriberId: string, token: string): RealtimeConnection {
    const wsBase = this.baseUrl.replace(/^http/, 'ws');
    const url = `${wsBase}/ws/${encodeURIComponent(workspaceId)}/${encodeURIComponent(
      subscriberId,
    )}?token=${encodeURIComponent(token)}`;
    this.connection = new RealtimeConnection(url);
    this.connection.open();
    return this.connection;
  }

  /** Closes the active realtime connection, if any. */
  disconnect(): void {
    this.connection?.close();
    this.connection = undefined;
  }

  // --- internal HTTP with timeout + retry on 5xx ---

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    let lastErr: unknown;
    for (let attempt = 0; attempt <= this.maxRetries; attempt += 1) {
      try {
        return await this.doRequest<T>(method, path, body);
      } catch (err) {
        lastErr = err;
        const retryable = err instanceof PushNotifyError && err.status !== undefined && err.status >= 500;
        if (!retryable || attempt === this.maxRetries) {
          throw err;
        }
        await sleep(Math.min(4000, 200 * 2 ** attempt));
      }
    }
    throw lastErr instanceof Error ? lastErr : new PushNotifyError('request failed');
  }

  private async doRequest<T>(method: string, path: string, body?: unknown): Promise<T> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeoutMs);
    try {
      const res = await fetch(`${this.baseUrl}${path}`, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': this.apiKey,
        },
        body: body !== undefined ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      const text = await res.text();
      const parsed = text ? safeJson(text) : undefined;

      if (!res.ok) {
        const message =
          (parsed && typeof parsed === 'object' && 'error' in parsed
            ? String((parsed as { error: unknown }).error)
            : res.statusText) || 'request failed';
        throw new PushNotifyError(message, res.status, parsed);
      }
      return parsed as T;
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') {
        throw new PushNotifyTimeoutError(this.timeoutMs);
      }
      throw err;
    } finally {
      clearTimeout(timer);
    }
  }
}

function safeJson(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

import type { Notification } from './types';

type MessageHandler = (notification: Notification) => void;
type StateHandler = () => void;

/**
 * RealtimeConnection wraps a browser/Node WebSocket with auto-reconnect and
 * exponential backoff. It is created via PushNotifyClient.connect().
 */
export class RealtimeConnection {
  private ws?: WebSocket;
  private messageHandlers: MessageHandler[] = [];
  private openHandlers: StateHandler[] = [];
  private closeHandlers: StateHandler[] = [];
  private closedByUser = false;
  private reconnectAttempts = 0;

  constructor(private readonly url: string) {}

  /** Opens the connection. Safe to call once. */
  open(): void {
    this.closedByUser = false;
    this.connect();
  }

  private connect(): void {
    const WS: typeof WebSocket | undefined =
      typeof WebSocket !== 'undefined'
        ? WebSocket
        : (globalThis as unknown as { WebSocket?: typeof WebSocket }).WebSocket;
    if (!WS) {
      throw new Error('No WebSocket implementation available in this environment');
    }

    this.ws = new WS(this.url);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.openHandlers.forEach((h) => h());
    };

    this.ws.onmessage = (event: MessageEvent) => {
      try {
        const data = typeof event.data === 'string' ? event.data : String(event.data);
        const notification = JSON.parse(data) as Notification;
        this.messageHandlers.forEach((h) => h(notification));
      } catch {
        // Ignore non-JSON frames (e.g. keepalives).
      }
    };

    this.ws.onclose = () => {
      this.closeHandlers.forEach((h) => h());
      if (!this.closedByUser) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = () => {
      // Errors surface as a subsequent close; reconnect is handled there.
    };
  }

  private scheduleReconnect(): void {
    this.reconnectAttempts += 1;
    const delay = Math.min(30000, 1000 * 2 ** (this.reconnectAttempts - 1));
    setTimeout(() => {
      if (!this.closedByUser) this.connect();
    }, delay);
  }

  onMessage(handler: MessageHandler): void {
    this.messageHandlers.push(handler);
  }

  onOpen(handler: StateHandler): void {
    this.openHandlers.push(handler);
  }

  onClose(handler: StateHandler): void {
    this.closeHandlers.push(handler);
  }

  /** Closes the connection and stops reconnecting. */
  close(): void {
    this.closedByUser = true;
    this.ws?.close();
  }
}

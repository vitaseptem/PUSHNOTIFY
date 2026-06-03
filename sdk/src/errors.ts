/** Base error type thrown by the PushNotify SDK. */
export class PushNotifyError extends Error {
  readonly status?: number;
  readonly body?: unknown;

  constructor(message: string, status?: number, body?: unknown) {
    super(message);
    this.name = 'PushNotifyError';
    this.status = status;
    this.body = body;
    // Restore prototype chain for transpiled targets.
    Object.setPrototypeOf(this, PushNotifyError.prototype);
  }
}

/** Thrown when a request exceeds the configured timeout. */
export class PushNotifyTimeoutError extends PushNotifyError {
  constructor(timeoutMs: number) {
    super(`request timed out after ${timeoutMs}ms`);
    this.name = 'PushNotifyTimeoutError';
    Object.setPrototypeOf(this, PushNotifyTimeoutError.prototype);
  }
}

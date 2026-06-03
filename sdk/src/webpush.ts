/**
 * Browser-only helpers for registering a Web Push subscription. These are
 * no-ops outside a browser with a service worker.
 */

/** Converts a base64url VAPID public key into the Uint8Array the Push API needs. */
export function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const raw = atob(base64);
  const output = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i += 1) {
    output[i] = raw.charCodeAt(i);
  }
  return output;
}

/**
 * Subscribes the current browser to Web Push using a service worker and the
 * workspace VAPID public key, returning the PushSubscription JSON.
 */
export async function subscribeWebPush(
  vapidPublicKey: string,
  serviceWorkerPath = '/sw.js',
): Promise<PushSubscriptionJSON> {
  if (typeof navigator === 'undefined' || !('serviceWorker' in navigator)) {
    throw new Error('Service workers are not supported in this environment');
  }
  const registration = await navigator.serviceWorker.register(serviceWorkerPath);
  const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(vapidPublicKey) as BufferSource,
  });
  return subscription.toJSON();
}

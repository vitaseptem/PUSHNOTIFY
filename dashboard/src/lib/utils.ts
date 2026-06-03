/** Joins class names, dropping falsy values. */
export function cn(...classes: Array<string | false | null | undefined>): string {
  return classes.filter(Boolean).join(' ');
}

/** Formats a number with thousands separators. */
export function formatNumber(n: number): string {
  return new Intl.NumberFormat('en-US').format(n);
}

/** Formats a 0-100 percentage with one decimal. */
export function formatPercent(n: number): string {
  return `${n.toFixed(1)}%`;
}

/** Returns a compact relative time like "5m ago". */
export function timeAgo(iso: string): string {
  const then = new Date(iso).getTime();
  const seconds = Math.floor((Date.now() - then) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

/** Truncates an id for compact display. */
export function shortId(id: string): string {
  return id.length > 8 ? `${id.slice(0, 8)}…` : id;
}

/** Maps a delivery/notification status to a semantic color token. */
export function statusColor(status: string): 'success' | 'error' | 'warning' | 'muted' {
  switch (status) {
    case 'delivered':
      return 'success';
    case 'failed':
      return 'error';
    case 'partial':
    case 'sent':
      return 'warning';
    default:
      return 'muted';
  }
}

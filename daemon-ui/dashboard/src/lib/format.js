/**
 * Format a timestamp as relative time ("2m ago", "1h ago", etc.)
 * Accepts ISO string, Unix ms, or Date object.
 */
export function relativeTime(ts) {
  if (!ts) return '--';

  let date;
  if (ts instanceof Date) {
    date = ts;
  } else if (typeof ts === 'number') {
    date = new Date(ts);
  } else {
    date = new Date(ts);
  }

  if (isNaN(date.getTime())) return '--';

  const now = Date.now();
  const diff = now - date.getTime();

  if (diff < 0) return 'just now';

  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return `${seconds}s ago`;

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;

  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;

  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;

  const years = Math.floor(months / 12);
  return `${years}y ago`;
}

/**
 * Format a number with comma separators: 1234 -> "1,234"
 */
export function formatNumber(n) {
  if (n === undefined || n === null) return '--';
  return Number(n).toLocaleString('en-US');
}

/**
 * Format duration in seconds with one decimal: 3.14 -> "3.1s"
 */
export function formatDuration(seconds) {
  if (seconds === undefined || seconds === null) return '--';
  return `${Number(seconds).toFixed(1)}s`;
}

/**
 * Truncate a string to maxLen characters, adding ellipsis.
 */
export function truncate(str, maxLen = 60) {
  if (!str) return '';
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen) + '...';
}

/**
 * Format a date as a short readable string.
 */
export function shortDate(ts) {
  if (!ts) return '--';
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '--';
  return d.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

/**
 * Format uptime from seconds to a human-readable string.
 */
export function formatUptime(seconds) {
  if (!seconds && seconds !== 0) return '--';
  const s = Math.floor(seconds);
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);

  const parts = [];
  if (d > 0) parts.push(`${d}d`);
  if (h > 0) parts.push(`${h}h`);
  if (m > 0 || parts.length === 0) parts.push(`${m}m`);
  return parts.join(' ');
}

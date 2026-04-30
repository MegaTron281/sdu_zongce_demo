export function formatNumber(value: number | string | null | undefined) {
  const number = Number(value || 0);
  return Number.isInteger(number) ? String(number) : number.toFixed(1);
}

export function formatDateTime(value: string | undefined) {
  if (!value) return "暂无";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function daysUntil(dateText: string) {
  const deadline = new Date(`${dateText}T23:59:59+08:00`);
  if (Number.isNaN(deadline.getTime())) return 999;
  return Math.ceil((deadline.getTime() - Date.now()) / 86400000);
}

export function publicityCountdown(value?: string) {
  if (!value) return "待公示";
  const deadline = new Date(value);
  const diff = deadline.getTime() - Date.now();
  if (Number.isNaN(deadline.getTime())) return "时间待确认";
  if (diff <= 0) return "已到期";
  const hours = Math.floor(diff / 3600000);
  const minutes = Math.floor((diff % 3600000) / 60000);
  if (hours >= 24) return `剩余 ${Math.floor(hours / 24)} 天 ${hours % 24} 小时`;
  return `剩余 ${hours} 小时 ${minutes} 分钟`;
}

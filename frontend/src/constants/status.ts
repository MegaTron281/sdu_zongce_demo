// 应用层状态字符串和颜色映射的唯一来源。
// 后端通过 /api/meta 返回 applicationStatuses / activityStatuses，前端组件不要再写中文字面量。

export const ApplicationStatus = {
  PendingReviewer: "待班委审核",
  Publicity: "公示中",
  PendingAdmin: "待管理员审核",
  Approved: "已通过",
  Rejected: "已驳回",
  Special: "特殊情况处理中",
} as const;

export type ApplicationStatusValue = (typeof ApplicationStatus)[keyof typeof ApplicationStatus];

export const ActivityStatus = {
  Listed: "已上架",
  Pending: "待审核上架",
  Rejected: "已驳回",
} as const;

export type ActivityStatusValue = (typeof ActivityStatus)[keyof typeof ActivityStatus];

const colorMap: Record<string, string> = {
  [ApplicationStatus.PendingReviewer]: "amber",
  [ApplicationStatus.Publicity]: "blue",
  [ApplicationStatus.PendingAdmin]: "teal",
  [ApplicationStatus.Approved]: "green",
  [ApplicationStatus.Rejected]: "red",
  [ApplicationStatus.Special]: "amber",
  [ActivityStatus.Listed]: "green",
  [ActivityStatus.Pending]: "amber",
};

export function statusColor(status: string): string {
  return colorMap[status] ?? "gray";
}

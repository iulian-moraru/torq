export interface settings {
  defaultDateRange: string;
  preferredTimezone: string;
  weekStartsOn: "saturday" | "sunday" | "monday";
}

export interface timeZone {
  name: string;
}

export interface localNode {
  localNodeId: number;
  implementation: string;
  grpcAddress?: string;
  tlsFileName?: string;
  tlsFile: File | null;
  macaroonFileName?: string;
  macaroonFile: File | null;
  createdOn?: Date;
  updatedOn?: Date;
  disabled: boolean;
  deleted: boolean;
}

// Canonical list of case statuses, mirroring the backend submission_status
// enum. Used by admin filters and pill rendering.
export const cases = [
  'draft',
  'pending_review',
  'in_verification',
  'approved',
  'published',
  'rejected',
  'archived',
] as const

export type CaseStatus = (typeof cases)[number]

import clsx, { type ClassValue } from 'clsx'

/**
 * cn merges class strings, dropping falsy values. We deliberately stay
 * with plain clsx (no tailwind-merge) for now — collisions are rare in
 * our small component set and we'd rather keep the bundle slim.
 */
export function cn(...inputs: ClassValue[]): string {
  return clsx(inputs)
}

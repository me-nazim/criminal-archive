import { type HTMLAttributes } from 'react'
import { cn } from '../../lib/cn'

type Width = 'narrow' | 'reading' | 'wide' | 'full'

const WIDTH: Record<Width, string> = {
  narrow: 'max-w-md',
  reading: 'max-w-3xl',
  wide: 'max-w-7xl',
  full: 'max-w-full',
}

export function Container({
  className,
  width = 'wide',
  ...rest
}: HTMLAttributes<HTMLDivElement> & { width?: Width }) {
  return (
    <div
      className={cn('mx-auto w-full px-4 sm:px-6 lg:px-8', WIDTH[width], className)}
      {...rest}
    />
  )
}

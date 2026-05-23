// Minimal multi-file dropzone. Calls back with a File[] when files are
// chosen via picker or drag-drop. Visual styling intentionally restrained
// — the actual upload progress UI lives in the calling page.

import { useCallback, useRef, useState, type DragEvent, type ReactNode } from 'react'
import { Upload } from 'lucide-react'
import { cn } from '../../lib/cn'

interface Props {
  onFiles: (files: File[]) => void
  disabled?: boolean
  accept?: string
  hint?: ReactNode
}

export function FileDropzone({ onFiles, disabled, accept, hint }: Props) {
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [hover, setHover] = useState(false)

  const onDrop = useCallback(
    (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault()
      setHover(false)
      if (disabled) return
      const files = Array.from(e.dataTransfer.files)
      if (files.length) onFiles(files)
    },
    [disabled, onFiles],
  )

  return (
    <div
      onDragOver={(e) => {
        e.preventDefault()
        setHover(true)
      }}
      onDragLeave={() => setHover(false)}
      onDrop={onDrop}
      className={cn(
        'flex flex-col items-center gap-3 rounded-lg border-2 border-dashed px-6 py-10 text-center transition-colors',
        hover ? 'border-brand-500 bg-brand-50' : 'border-ink-300 bg-ink-50',
        disabled && 'cursor-not-allowed opacity-60',
      )}
    >
      <Upload className="h-7 w-7 text-ink-400" aria-hidden />
      <p className="text-sm text-ink-700">
        Drag &amp; drop files here, or
        <button
          type="button"
          disabled={disabled}
          onClick={() => inputRef.current?.click()}
          className="ml-1 font-medium text-brand-700 underline-offset-2 hover:underline disabled:no-underline"
        >
          browse
        </button>
      </p>
      {hint && <p className="text-xs text-ink-500">{hint}</p>}
      <input
        ref={inputRef}
        type="file"
        multiple
        accept={accept}
        className="sr-only"
        onChange={(e) => {
          const files = Array.from(e.target.files ?? [])
          if (files.length) onFiles(files)
          if (inputRef.current) inputRef.current.value = ''
        }}
      />
    </div>
  )
}

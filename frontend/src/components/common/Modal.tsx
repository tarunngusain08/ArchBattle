import type { PropsWithChildren } from 'react'

export function Modal({ children, open }: PropsWithChildren<{ open: boolean }>) {
  if (!open) {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4">
      <div className="panel w-full max-w-lg rounded-3xl p-6">{children}</div>
    </div>
  )
}

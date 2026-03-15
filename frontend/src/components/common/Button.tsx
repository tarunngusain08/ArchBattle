import type { ButtonHTMLAttributes, PropsWithChildren } from 'react'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost'
  fullWidth?: boolean
}

export function Button({ children, className = '', variant = 'primary', fullWidth = false, ...props }: PropsWithChildren<ButtonProps>) {
  const variantClass =
    variant === 'secondary'
      ? 'bg-slate-800 text-slate-100 border border-slate-600 hover:bg-slate-700'
      : variant === 'ghost'
        ? 'bg-transparent text-slate-200 hover:bg-slate-800/60'
        : 'bg-cyan-500 text-slate-950 hover:bg-cyan-400'

  return (
    <button
      className={`rounded-xl px-4 py-2 font-semibold transition ${variantClass} ${fullWidth ? 'w-full' : ''} ${className}`}
      {...props}
    >
      {children}
    </button>
  )
}

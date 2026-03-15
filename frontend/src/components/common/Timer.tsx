import { useTimer } from '../../hooks/useTimer'

export function Timer({ durationSeconds, running = true }: { durationSeconds: number; running?: boolean }) {
  const secondsLeft = useTimer(durationSeconds, running)
  return <span className="rounded-full bg-slate-900 px-3 py-1 text-sm font-semibold text-cyan-300">{secondsLeft}s</span>
}

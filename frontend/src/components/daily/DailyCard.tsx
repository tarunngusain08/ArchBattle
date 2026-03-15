import type { DailyChallenge } from '../../types'
import { Button } from '../common/Button'

export function DailyCard({ challenge, onOpen }: { challenge?: DailyChallenge; onOpen: () => void }) {
  return (
    <section className="panel rounded-3xl p-6">
      <p className="text-xs uppercase tracking-[0.3em] text-cyan-300">Daily challenge</p>
      <h2 className="mt-3 text-2xl font-semibold text-white">{challenge?.theme ?? "Load today's challenge"}</h2>
      <p className="mt-2 text-sm text-slate-300">
        {challenge ? `${challenge.questions.length} questions with a shared theme.` : 'Pull the latest challenge and keep your streak alive.'}
      </p>
      <div className="mt-5">
        <Button onClick={onOpen}>Open daily challenge</Button>
      </div>
    </section>
  )
}

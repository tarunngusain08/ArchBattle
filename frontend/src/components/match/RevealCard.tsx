export function RevealCard({ rationale, correctAnswers }: { rationale?: string; correctAnswers?: number[] }) {
  return (
    <section className="panel rounded-3xl p-6">
      <h2 className="text-xl font-semibold text-white">Question reveal</h2>
      <p className="mt-3 text-sm text-slate-300">Correct answers: {(correctAnswers ?? []).map((value) => value + 1).join(', ') || 'Unknown'}</p>
      <p className="mt-4 text-slate-100">{rationale ?? 'Waiting for rationale...'}</p>
    </section>
  )
}

import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { Button } from '../components/common/Button'
import {
  createDiscussionEntry,
  listDiscussion,
  upvoteEntry,
  type CreateRequest,
  type DiscussionEntry,
} from '../api/discussion'

function formatDate(d: string): string {
  const parsed = new Date(d)
  return parsed.toISOString().slice(0, 10)
}

function EntryCard({
  entry,
  date,
  onUpvote,
  onUpvoteError,
}: {
  entry: DiscussionEntry
  date: string
  onUpvote: () => void
  onUpvoteError?: (msg: string) => void
}) {
  return (
    <div className="rounded-2xl border border-slate-700 bg-slate-900/70 p-4">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-cyan-300">{entry.username}</span>
        <button
          type="button"
          aria-label="Upvote"
          aria-pressed={false}
          onClick={async () => {
            try {
              await upvoteEntry(date, entry.id)
              onUpvote()
            } catch {
              onUpvoteError?.('Failed to upvote')
            }
          }}
          className="rounded-lg bg-slate-800 px-3 py-1 text-sm text-slate-200 hover:bg-slate-700"
        >
          ↑ {entry.upvotes}
        </button>
      </div>
      {entry.reasoningText && (
        <div className="mt-3">
          <p className="text-xs uppercase tracking-wider text-slate-500">Reasoning</p>
          <p className="mt-1 text-sm text-slate-200">{entry.reasoningText}</p>
        </div>
      )}
      {entry.alternativeText && (
        <div className="mt-3">
          <p className="text-xs uppercase tracking-wider text-slate-500">Alternatives</p>
          <p className="mt-1 text-sm text-slate-200">{entry.alternativeText}</p>
        </div>
      )}
      {entry.surpriseText && (
        <div className="mt-3">
          <p className="text-xs uppercase tracking-wider text-slate-500">Surprises</p>
          <p className="mt-1 text-sm text-slate-200">{entry.surpriseText}</p>
        </div>
      )}
    </div>
  )
}

export function DiscussionPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const dateParam = searchParams.get('date')
  const today = formatDate(new Date().toISOString())
  const date = dateParam || today

  const [entries, setEntries] = useState<DiscussionEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string>()
  const [form, setForm] = useState<CreateRequest>({
    questionNumber: 1,
    reasoningText: '',
    alternativeText: '',
    surpriseText: '',
  })
  const [submitting, setSubmitting] = useState(false)

  const refresh = () => {
    setLoading(true)
    listDiscussion(date)
      .then(setEntries)
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    refresh()
  }, [date])

  const byQuestion = entries.reduce<Record<number, DiscussionEntry[]>>((acc, e) => {
    if (!acc[e.questionNumber]) acc[e.questionNumber] = []
    acc[e.questionNumber].push(e)
    return acc
  }, {})

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (submitting || (!form.reasoningText && !form.alternativeText && !form.surpriseText)) return
    setSubmitting(true)
    try {
      await createDiscussionEntry(date, form)
      setForm({ questionNumber: 1, reasoningText: '', alternativeText: '', surpriseText: '' })
      refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to share')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-6">
      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Discussion</p>
        <h2 className="mt-3 text-2xl font-semibold text-white">Daily challenge discussion</h2>
        <label className="mt-2 block text-sm text-slate-400">
          Date
          <input
            type="date"
            className="ml-2 rounded-lg border border-slate-700 bg-slate-900 px-2 py-1 text-slate-100"
            value={date}
            onChange={(e) => {
              const next = e.target.value
              if (next) {
                navigate(`/discussion?date=${next}`, { replace: true })
              }
            }}
          />
        </label>
      </section>

      <section className="panel rounded-3xl p-6">
        <h3 className="text-lg font-medium text-white">Share my thinking</h3>
        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          <label className="block text-sm text-slate-300">
            Question
            <select
              className="mt-2 w-full max-w-xs rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100"
              value={form.questionNumber}
              onChange={(e) => setForm({ ...form, questionNumber: Number(e.target.value) })}
            >
              <option value={1}>1</option>
              <option value={2}>2</option>
              <option value={3}>3</option>
            </select>
          </label>
          <label className="block text-sm text-slate-300">
            Reasoning
            <textarea
              className="mt-2 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100"
              rows={2}
              value={form.reasoningText}
              onChange={(e) => setForm({ ...form, reasoningText: e.target.value })}
              placeholder="How did you approach this?"
            />
          </label>
          <label className="block text-sm text-slate-300">
            Alternatives
            <textarea
              className="mt-2 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100"
              rows={2}
              value={form.alternativeText}
              onChange={(e) => setForm({ ...form, alternativeText: e.target.value })}
              placeholder="Other approaches you considered"
            />
          </label>
          <label className="block text-sm text-slate-300">
            Surprises
            <textarea
              className="mt-2 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100"
              rows={2}
              value={form.surpriseText}
              onChange={(e) => setForm({ ...form, surpriseText: e.target.value })}
              placeholder="What surprised you?"
            />
          </label>
          <Button type="submit" disabled={submitting || (!form.reasoningText && !form.alternativeText && !form.surpriseText)}>
            Share
          </Button>
        </form>
      </section>

      {error && <p className="text-sm text-rose-300">{error}</p>}

      <section className="panel rounded-3xl p-6">
        <h3 className="text-lg font-medium text-white">Community entries</h3>
        {loading ? (
          <p className="mt-4 text-slate-400">Loading…</p>
        ) : (
          <div className="mt-4 space-y-6">
            {[1, 2, 3].map((q) => (
              <div key={q}>
                <p className="text-sm font-medium text-slate-400">Question {q}</p>
                <div className="mt-2 space-y-3">
                    {(byQuestion[q] ?? []).map((entry) => (
                    <EntryCard key={entry.id} entry={entry} date={date} onUpvote={refresh} onUpvoteError={setError} />
                  ))}
                  {(byQuestion[q] ?? []).length === 0 && (
                    <p className="text-sm text-slate-500">No entries yet.</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}

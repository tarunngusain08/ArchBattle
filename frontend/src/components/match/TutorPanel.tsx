import { useState } from 'react'

import { apiFetch } from '../../api/client'
import { Button } from '../common/Button'

interface TutorMessage {
  role: 'user' | 'assistant'
  content: string
}

interface TutorPanelProps {
  userId?: string
  questionId?: string
  questionPrompt?: string
  officialReason?: string
  playerAnswer?: number[]
  onClose?: () => void
}

interface TutorResponse {
  text: string
  tokenCount?: number
}

/**
 * TutorPanel is a slide-in overlay that lets players ask the AI tutor questions
 * about the current question during the reveal phase. Calls POST /api/tutor on Core,
 * which authenticates the user and proxies to the AI service.
 */
export function TutorPanel({
  userId,
  questionId,
  questionPrompt,
  officialReason,
  playerAnswer,
  onClose,
}: TutorPanelProps) {
  const [history, setHistory] = useState<TutorMessage[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSend() {
    const trimmed = input.trim()
    if (!trimmed || loading) {
      return
    }

    const userMessage: TutorMessage = { role: 'user', content: trimmed }
    setHistory((prev) => [...prev, userMessage])
    setInput('')
    setLoading(true)
    setError(null)

    try {
      const response = await apiFetch<TutorResponse>('/api/tutor', {
        method: 'POST',
        body: JSON.stringify({
          userId,
          question_id: questionId,
          question_prompt: questionPrompt,
          official_reason: officialReason,
          player_answer: playerAnswer,
          history: history.map((m) => ({ role: m.role, content: m.content })),
          user_question: trimmed,
        }),
      })
      setHistory((prev) => [...prev, { role: 'assistant', content: response.text }])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to get tutor response')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-y-0 right-0 z-50 flex w-full max-w-md flex-col bg-slate-900 shadow-2xl">
      <div className="flex items-center justify-between border-b border-slate-700 px-5 py-4">
        <h2 className="text-lg font-semibold text-white">AI Tutor</h2>
        <button
          className="text-slate-400 hover:text-white"
          onClick={onClose}
          aria-label="Close tutor"
        >
          ✕
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
        {history.length === 0 && (
          <p className="text-sm text-slate-400">
            Ask the AI tutor to explain this question, explore trade-offs, or clarify the rationale.
          </p>
        )}
        {history.map((msg, idx) => (
          <div
            key={idx}
            className={`rounded-2xl px-4 py-3 text-sm ${
              msg.role === 'user'
                ? 'ml-6 bg-cyan-700/30 text-cyan-100'
                : 'mr-6 bg-slate-800 text-slate-200'
            }`}
          >
            {msg.content}
          </div>
        ))}
        {loading && (
          <div className="mr-6 rounded-2xl bg-slate-800 px-4 py-3 text-sm text-slate-400 animate-pulse">
            Thinking…
          </div>
        )}
        {error && <p className="text-sm text-red-400">{error}</p>}
      </div>

      <div className="border-t border-slate-700 px-5 py-4">
        <div className="flex gap-2">
          <input
            className="flex-1 rounded-xl border border-slate-600 bg-slate-800 px-4 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-cyan-500"
            placeholder="Ask a question…"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                void handleSend()
              }
            }}
            disabled={loading}
          />
          <Button onClick={() => void handleSend()} disabled={loading || !input.trim()}>
            Send
          </Button>
        </div>
      </div>
    </div>
  )
}

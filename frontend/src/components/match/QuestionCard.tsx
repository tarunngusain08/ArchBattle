import { useEffect, useState } from 'react'
import type { QuestionSnapshot } from '../../types'
import { Button } from '../common/Button'
import { Timer } from '../common/Timer'

export function QuestionCard({ question, onChoose }: { question: QuestionSnapshot; onChoose: (choice: number) => void }) {
  const [highlighted, setHighlighted] = useState<number>(0)

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const n = e.key === '1' ? 0 : e.key === '2' ? 1 : e.key === '3' ? 2 : e.key === '4' ? 3 : -1
      if (n >= 0 && n < question.options.length) {
        e.preventDefault()
        setHighlighted(n)
      } else if (e.key === 'Enter') {
        e.preventDefault()
        if (highlighted >= 0 && highlighted < question.options.length) {
          onChoose(highlighted)
        }
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [question.options.length, highlighted, onChoose])

  return (
    <section className="panel rounded-3xl p-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-cyan-300">{question.topic}</p>
          <h2 className="mt-2 text-2xl font-semibold text-white">{question.prompt}</h2>
        </div>
        <Timer durationSeconds={75} />
      </div>
      <div className="mt-6 grid gap-3">
        {question.options.map((option, index) => (
          <Button
            key={option}
            variant={highlighted === index ? 'primary' : 'secondary'}
            className="justify-start text-left"
            onClick={() => onChoose(index)}
            onFocus={() => setHighlighted(index)}
          >
            <span className="mr-3 rounded-full bg-slate-700 px-2 py-1 text-xs text-slate-100">{index + 1}</span>
            {option}
          </Button>
        ))}
      </div>
    </section>
  )
}

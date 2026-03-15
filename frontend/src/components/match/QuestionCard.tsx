import { useCallback, useEffect, useState } from 'react'
import type { QuestionSnapshot } from '../../types'
import { Button } from '../common/Button'
import { Timer } from '../common/Timer'

export function QuestionCard({ question, onChoose }: { question: QuestionSnapshot; onChoose: (choice: number) => void }) {
  const [selected, setSelected] = useState<number | null>(null)
  const [highlighted, setHighlighted] = useState<number>(0)

  useEffect(() => {
    setSelected(null)
    setHighlighted(0)
  }, [question.id])

  const selectAndSubmit = useCallback(
    (index: number) => {
      setSelected(index)
      setHighlighted(index)
      onChoose(index)
    },
    [onChoose],
  )

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const n = e.key === '1' ? 0 : e.key === '2' ? 1 : e.key === '3' ? 2 : e.key === '4' ? 3 : -1
      if (n >= 0 && n < question.options.length) {
        e.preventDefault()
        setHighlighted(n)
      } else if (e.key === 'Enter') {
        e.preventDefault()
        if (highlighted >= 0 && highlighted < question.options.length) {
          selectAndSubmit(highlighted)
        }
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [question.options.length, highlighted, selectAndSubmit])

  return (
    <section className="panel rounded-3xl p-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-cyan-300">{question.topic}</p>
          <h2 className="mt-2 text-2xl font-semibold text-white">{question.prompt}</h2>
        </div>
        <Timer durationSeconds={60} />
      </div>
      <p className="mt-2 text-xs text-slate-400">You can change your answer until time runs out</p>
      <div className="mt-6 grid gap-3">
        {question.options.map((option, index) => {
          const isSelected = selected === index
          const isHighlighted = highlighted === index && selected !== index
          return (
            <Button
              key={option}
              variant={isSelected ? 'primary' : isHighlighted ? 'secondary' : 'secondary'}
              className={`justify-start text-left ${isSelected ? 'ring-2 ring-cyan-400 ring-offset-1 ring-offset-slate-900' : ''}`}
              onClick={() => selectAndSubmit(index)}
              onFocus={() => setHighlighted(index)}
              aria-label={`Option ${index + 1}: ${option}`}
              aria-pressed={isSelected}
            >
              <span className={`mr-3 rounded-full px-2 py-1 text-xs ${isSelected ? 'bg-cyan-500 text-white' : 'bg-slate-700 text-slate-100'}`}>
                {index + 1}
              </span>
              {option}
              {isSelected ? <span className="ml-auto text-xs text-cyan-300">Selected</span> : null}
            </Button>
          )
        })}
      </div>
    </section>
  )
}

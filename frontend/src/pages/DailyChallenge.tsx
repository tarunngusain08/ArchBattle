import { useEffect, useMemo } from 'react'

import { fetchDailyChallenge, submitDailyChallenge } from '../api/daily'
import { Button } from '../components/common/Button'
import { ShareButton } from '../components/daily/ShareButton'
import { useDailyStore } from '../stores/dailyStore'

export function DailyChallengePage() {
  const { challenge, answers, result, setChallenge, setAnswer, setResult } = useDailyStore()

  useEffect(() => {
    if (!challenge) {
      void fetchDailyChallenge().then(setChallenge).catch(() => undefined)
    }
  }, [challenge, setChallenge])

  const correctCount = useMemo(
    () =>
      challenge?.questions.reduce((count, question) => {
        const answer = answers[question.id]?.[0]
        return question.correctAnswers?.includes(answer ?? -1) ? count + 1 : count
      }, 0) ?? 0,
    [answers, challenge],
  )

  return (
    <div className="space-y-6">
      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Daily challenge</p>
        <h2 className="mt-3 text-2xl font-semibold text-white">{challenge?.theme ?? "Loading today's theme..."}</h2>
        <div className="mt-6 grid gap-5">
          {challenge?.questions.map((question, index) => (
            <div key={question.id} className="rounded-2xl border border-slate-700 bg-slate-900/70 p-4">
              <p className="text-sm text-cyan-300">Question {index + 1}</p>
              <p className="mt-2 text-white">{question.prompt}</p>
              <div className="mt-4 grid gap-2">
                {question.options.map((option, optionIndex) => (
                  <Button key={option} variant={answers[question.id]?.[0] === optionIndex ? 'primary' : 'secondary'} className="justify-start text-left" onClick={() => setAnswer(question.id, [optionIndex])}>
                    {option}
                  </Button>
                ))}
              </div>
            </div>
          ))}
        </div>
        <div className="mt-6 flex gap-3">
          <Button
            onClick={async () => {
              if (!challenge) {
                return
              }
              const response = await submitDailyChallenge({ date: challenge.challengeDate.slice(0, 10), answers, totalMillis: 32000 })
              setResult(response)
            }}
          >
            Submit daily challenge
          </Button>
          {result ? <ShareButton text={result.shareCardText} /> : null}
        </div>
      </section>

      {result ? (
        <>
          <section className="panel rounded-3xl p-6">
            <h2 className="text-2xl font-semibold text-white">Daily results</h2>
            <p className="mt-3 text-slate-300">Score: {result.score} | Correct: {correctCount} | Percentile: Top {result.percentile}%</p>
            <p className="mt-2 text-slate-300">Streak day: {result.streakDay}</p>
            <pre className="mt-4 whitespace-pre-wrap rounded-2xl bg-slate-950/60 p-4 text-sm text-slate-100">{result.shareCardText}</pre>
          </section>
          {challenge?.aiSummary ? (
            <section className="panel rounded-3xl p-6">
              <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Community insights</p>
              <h2 className="mt-3 text-xl font-semibold text-white">AI discussion summary</h2>
              <p className="mt-4 whitespace-pre-wrap text-sm text-slate-200">{challenge.aiSummary}</p>
            </section>
          ) : null}
        </>
      ) : null}
    </div>
  )
}

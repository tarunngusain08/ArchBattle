import { useEffect, useState } from 'react'

export function useTimer(durationSeconds: number, running = true) {
  const [secondsLeft, setSecondsLeft] = useState(durationSeconds)

  useEffect(() => {
    setSecondsLeft(durationSeconds)
  }, [durationSeconds])

  useEffect(() => {
    if (!running || secondsLeft <= 0) {
      return undefined
    }
    const timer = window.setTimeout(() => setSecondsLeft((current) => current - 1), 1000)
    return () => window.clearTimeout(timer)
  }, [running, secondsLeft])

  return secondsLeft
}

export function StreakBadge({ streak }: { streak: number }) {
  return (
    <div className="inline-flex items-center rounded-full bg-amber-400/20 px-4 py-2 text-sm font-semibold text-amber-200">
      {streak} day streak
    </div>
  )
}

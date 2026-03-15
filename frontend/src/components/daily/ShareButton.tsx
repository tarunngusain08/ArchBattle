import { Button } from '../common/Button'

export function ShareButton({ text }: { text: string }) {
  return (
    <Button
      variant="secondary"
      onClick={async () => {
        await navigator.clipboard.writeText(text)
      }}
    >
      Copy share card
    </Button>
  )
}

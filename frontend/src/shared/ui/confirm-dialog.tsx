import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogCancel,
  AlertDialogFooter,
} from '@/components/ui/alert-dialog'
import { useState, type ReactNode } from 'react'
import { ApiErrorNotice } from './api-error-notice'
import { Button } from '@/components/ui/button'

interface ConfirmDialogProps {
  trigger: ReactNode
  title: string
  description: string
  confirmLabel: string
  cancelLabel: string
  onConfirm: () => Promise<unknown>
  pending?: boolean
}

export function ConfirmDialog({
  trigger,
  title,
  description,
  confirmLabel,
  cancelLabel,
  onConfirm,
  pending = false,
}: ConfirmDialogProps) {
  const [open, setOpen] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const [busy, setBusy] = useState(false)
  async function confirm() {
    setBusy(true)
    setError(null)
    try {
      await onConfirm()
      setOpen(false)
    } catch (cause) {
      setError(cause)
    } finally {
      setBusy(false)
    }
  }
  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!busy) {
          setOpen(next)
          setError(null)
        }
      }}
    >
      <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger>
      <AlertDialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto">
        <AlertDialogTitle className="text-lg font-semibold text-foreground">{title}</AlertDialogTitle>
        <AlertDialogDescription className="mt-2 text-sm leading-6 text-muted-foreground">
          {description}
        </AlertDialogDescription>
        <ApiErrorNotice error={error} />
        <AlertDialogFooter>
          <AlertDialogCancel disabled={busy || pending}>{cancelLabel}</AlertDialogCancel>
          <Button variant="destructive" disabled={pending || busy} onClick={() => void confirm()}>
            {confirmLabel}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

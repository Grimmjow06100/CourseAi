import { useEffect, useRef, useState } from 'react'
import { Input } from './form-controls'

// Restore URL changes without remounting the input and losing keyboard focus.
export function DebouncedSearch({
  initialValue,
  navigationKey,
  onSearch,
  label,
}: {
  initialValue: string
  navigationKey: string
  onSearch: (value: string) => void
  label: string
}) {
  const [value, setValue] = useState(initialValue)
  const [previousKey, setPreviousKey] = useState(navigationKey)
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  if (previousKey !== navigationKey) {
    setPreviousKey(navigationKey)
    setValue(initialValue)
  }
  useEffect(() => () => clearTimeout(timer.current), [navigationKey])
  return (
    <Input
      className="pl-9"
      value={value}
      aria-label={label}
      placeholder={label}
      onChange={(event) => {
        const next = event.target.value
        setValue(next)
        clearTimeout(timer.current)
        timer.current = setTimeout(() => onSearch(next), 350)
      }}
    />
  )
}

export function formatDuration(minutes: number, locale: string) {
  if (minutes < 60) return `${minutes} min`
  const hours = Math.floor(minutes / 60)
  const remaining = minutes % 60
  return new Intl.ListFormat(locale, { style: 'short', type: 'unit' }).format([
    `${hours} h`,
    ...(remaining > 0 ? [`${remaining} min`] : []),
  ])
}

export function formatDate(value: string, locale: string) {
  return new Intl.DateTimeFormat(locale, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  }).format(new Date(value))
}

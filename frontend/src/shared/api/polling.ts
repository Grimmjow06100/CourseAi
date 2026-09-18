interface QueryObservation {
  state: { dataUpdateCount: number }
}

const recoveryStarts = new WeakMap<QueryObservation, number>()

// Count recovery observations, not all fetches since a long-running query mounted.
export function shouldPollQuery(query: QueryObservation, active: boolean, recoverable: boolean) {
  if (active || !recoverable) {
    recoveryStarts.delete(query)
    return active
  }
  const count = query.state.dataUpdateCount
  const start = recoveryStarts.get(query)
  if (start === undefined || count < start) {
    recoveryStarts.set(query, count)
    return true
  }
  return count - start < 30
}

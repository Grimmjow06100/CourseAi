// Each authenticated runtime owns its requests, including mutations in flight.
export class SessionLifetime {
  private controller = new AbortController()
  getSignal = () => this.controller.signal
  activate() {
    // React StrictMode replays effects during development.
    if (this.controller.signal.aborted) this.controller = new AbortController()
  }
  dispose() {
    this.controller.abort()
  }
}

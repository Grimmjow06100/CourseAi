package correlation

import "context"

type jobContextKey struct{}

type Job struct {
	JobID     string
	RequestID string
	Kind      string
	Attempt   int
}

func WithJob(ctx context.Context, job Job) context.Context {
	return context.WithValue(ctx, jobContextKey{}, job)
}

func JobFromContext(ctx context.Context) (Job, bool) {
	job, ok := ctx.Value(jobContextKey{}).(Job)
	return job, ok
}

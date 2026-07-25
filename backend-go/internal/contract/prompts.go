package contract

type PromptStore interface {
	Get(name string) (string, bool)
	Names() []string
}

package brocker

type Api interface {
	Connected() bool
	// pValue must be a pointer to a value where result will be written
	Get(reper string, pValue any) error
	Set(reper string, value any) error
}

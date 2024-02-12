package tags

// Errors is the type we expect to contain the errors in the model we're representing with forms.
type Errors interface {
	Get(key string) []string
}

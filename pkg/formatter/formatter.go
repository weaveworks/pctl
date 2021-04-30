package formatter

// Formatter formats data
type Formatter interface {
	// Format will call the getter func and render the returned data
	Format(getter func() interface{}) (string, error)
}

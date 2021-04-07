package catalog

// ProfileDescription defines the descriptive properties of a profile.
type ProfileDescription struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
}

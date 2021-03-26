package catalog

type ProfileCatalog struct {
	Spec ProfileCatalogSpec `yaml:"spec"`
}

type ProfileCatalogSpec struct {
	Profiles []ProfileDescription `json:"profiles,omitempty"`
}

type ProfileDescription struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
}

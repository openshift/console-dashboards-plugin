package datasources

type DatasourceMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type DatasourcePluginSpec struct {
	DirectURL string `json:"direct_url"`
}

type DatasourcePlugin struct {
	Kind string               `json:"kind"`
	Spec DatasourcePluginSpec `json:"spec"`
}

type DatasourceSpec struct {
	Plugin DatasourcePlugin `json:"plugin"`
}

type DataSource struct {
	Kind     string             `json:"kind"`
	Metadata DatasourceMetadata `json:"metadata"`
	Spec     DatasourceSpec     `json:"spec"`
}

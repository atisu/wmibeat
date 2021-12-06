// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

type Config struct {
	Wmibeat WmibeatConfig
}

type WmibeatConfig struct {
	Period     string `yaml:"period"`
	Classes    []ClassConfig
	Namespaces []NamespaceConfig
}

type ClassConfig struct {
	Class       string   `config:"class"`
	Fields      []string `config:"fields"`
	WhereClause string   `config:"whereclause"`
	ObjectTitle string   `config:"objecttitlecolumn"`
}

type NamespaceConfig struct {
	Namespace                string   `config:"namespace"`
	Class                    string   `config:"class"`
	MetricNameCombinedFields []string `config:"metric_name_combined_fields"`
	MetricValueField         string   `config:"metric_value_field"`
	WhereClause              string   `config:"whereclause"`
}

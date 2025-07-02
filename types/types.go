package types

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

type TableDescription struct {
	Name        string           `json:"name"`
	Columns     []Column         `json:"columns"`
	RowCount    int64            `json:"row_count"`
	SampleData  []map[string]any `json:"sample_data,omitempty"`
	Indexes     []Index          `json:"indexes,omitempty"`
	PrimaryKeys []string         `json:"primary_keys,omitempty"`
}

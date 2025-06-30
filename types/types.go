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

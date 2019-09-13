package rules

type Rule struct {
	Path  string  `json:"path"`
	Allow []Allow `json:"allow"`
}

type Allow struct {
	Methods []Method `json:"methods"`
	If      string   `json:"if"`
}

type Method string

const (
	READ   Method = "READ"
	WRITE  Method = "WRITE"
	DELETE Method = "DELETE"
)
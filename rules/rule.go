package rules

type Rule struct {
	Path  string
	Read  Allow
	Write Allow
}

type Allow struct {
	IfPath    string //Only user, path and with available
	IfContent string //content, user, path and with always available, newContent only for put/patch cases
	With      []With
}
type With struct {
	Name string
	Path string
}

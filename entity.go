package graphql

const (
	filterNone = iota
	filterBool
	filterDate
	filterFloat
	filterInt
	filterString
)

//Entity represents a content type
type Entity struct {
	Name        string
	Label       string
	Description string
	Icon        string
	Instance    interface{}
	filters     map[string]int8
	Resolvers   Resolvers
}

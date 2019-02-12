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
type Entity interface {
	Description() string
	Instance() interface{}
	Resolvers() Resolvers
}

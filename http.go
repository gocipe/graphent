package graphql

import (
	"encoding/json"
	"net/http"

	"github.com/graphql-go/graphql"
)

//NewHTTPEndpoint creates a new http graphql endpoint
func NewHTTPEndpoint(schema graphql.Schema) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query().Get("query")
		result := graphql.Do(graphql.Params{
			Schema:        schema,
			RequestString: query,
		})

		if result.HasErrors() {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

package apollo

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	graphql "github.com/neelance/graphql-go"
)

const (
	ContentTypeJSON           = "application/json"
	ContentTypeGraphQL        = "application/graphql"
	ContentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

type Handler struct {
	Schema *graphql.Schema
	Pretty bool
}

type RequestOptions struct {
	Query         string                 `json:"query" url:"query"`
	Variables     map[string]interface{} `json:"variables" url:"variables"`
	OperationName string                 `json:"operationName" url:"operationName"`
}

// a workaround for getting`variables` as a JSON string
type requestOptionsCompatibility struct {
	Query         string `json:"query" url:"query"`
	Variables     string `json:"variables" url:"variables"`
	OperationName string `json:"operationName" url:"operationName"`
}

func getFromForm(values url.Values) *RequestOptions {
	query := values.Get("query")
	if query != "" {
		// get variables map
		var variables map[string]interface{}
		variablesStr := values.Get("variables")
		json.Unmarshal([]byte(variablesStr), &variables)

		return &RequestOptions{
			Query:         query,
			Variables:     variables,
			OperationName: values.Get("operationName"),
		}
	}

	return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var output interface{}

	if r.Method == http.MethodGet {
		query := getFromForm(r.URL.Query())
		if query == nil {
			http.Error(w, "Could not parse graphql params from url params.", http.StatusInternalServerError)
			return
		}

		output = h.Schema.Exec(r.Context(), query.Query, query.OperationName, query.Variables)
	} else {
		// TODO: improve Content-Type handling
		contentTypeStr := r.Header.Get("Content-Type")
		contentTypeTokens := strings.Split(contentTypeStr, ";")
		contentType := contentTypeTokens[0]

		switch contentType {
		case ContentTypeFormURLEncoded:
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			query := getFromForm(r.PostForm)
			if query == nil {
				http.Error(w, "Could not parse graphql params from form.", http.StatusInternalServerError)
				return
			}

			output = h.Schema.Exec(r.Context(), query.Query, query.OperationName, query.Variables)
		case ContentTypeGraphQL:
			body, err := ioutil.ReadAll(r.Body)
			if err == nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			output = h.Schema.Exec(r.Context(), string(body), "", make(map[string]interface{}))
		case ContentTypeJSON:
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			requestOptions := resolveJSON(body)
			switch opts := requestOptions.(type) {
			case *RequestOptions:
				output = h.Schema.Exec(r.Context(), opts.Query, opts.OperationName, opts.Variables)
			case []*RequestOptions:
				var results []*graphql.Response
				for i := range opts {
					results = append(results, h.Schema.Exec(r.Context(), opts[i].Query, opts[i].OperationName, opts[i].Variables))
				}
				output = results
			default:
				log.Printf("bad type: %T", opts)
				http.Error(w, "unrecognized RequestOptions type", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "unrecognized content-type header: `"+contentType+"`", http.StatusBadRequest)
			return
		}
	}

	var responseJSON []byte
	var err error
	if h.Pretty {
		responseJSON, err = json.MarshalIndent(output, "", "  ")
	} else {
		responseJSON, err = json.Marshal(output)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}

func resolveJSON(body []byte) interface{} {
	if bytes.HasPrefix(body, []byte("[")) {
		var queries []*RequestOptions
		err := json.Unmarshal(body, &queries)
		if err != nil {
			// Probably `variables` was sent as a string instead of an object.
			// So, we try to be polite and try to parse that as a JSON string
			var optionsCompatible []*requestOptionsCompatibility
			json.Unmarshal(body, &optionsCompatible)
			for i := range optionsCompatible {
				json.Unmarshal([]byte(optionsCompatible[i].Variables), &queries[i].Variables)
			}
			return queries
		}
		return queries
	}

	var query RequestOptions
	err := json.Unmarshal(body, &query)
	if err != nil {
		// Probably `variables` was sent as a string instead of an object.
		// So, we try to be polite and try to parse that as a JSON string
		var optionsCompatible requestOptionsCompatibility
		json.Unmarshal(body, &optionsCompatible)
		json.Unmarshal([]byte(optionsCompatible.Variables), &query.Variables)
		return &query
	}
	return &query
}

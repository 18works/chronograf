package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bouk/httprouter"
	"github.com/influxdata/flux/ast"
	_ "github.com/influxdata/flux/builtin"
	"github.com/influxdata/flux/complete"
	"github.com/influxdata/flux/parser"
)

// Params are params
type Params map[string]string

// SuggestionsResponse provides a list of available Flux functions
type SuggestionsResponse struct {
	Functions []SuggestionResponse `json:"funcs"`
}

// SuggestionResponse provides the parameters available for a given Flux function
type SuggestionResponse struct {
	Name   string `json:"name"`
	Params Params `json:"params"`
}

type fluxLinks struct {
	Self        string `json:"self"`        // Self link mapping to this resource
	Suggestions string `json:"suggestions"` // URL for flux builder function suggestions
}

type fluxResponse struct {
	Links fluxLinks `json:"links"`
}

// Flux returns a list of links for the Flux API
func (s *Service) Flux(w http.ResponseWriter, r *http.Request) {
	httpAPIFlux := "/chronograf/v1/flux"
	res := fluxResponse{
		Links: fluxLinks{
			Self:        fmt.Sprintf("%s", httpAPIFlux),
			Suggestions: fmt.Sprintf("%s/suggestions", httpAPIFlux),
		},
	}

	encodeJSON(w, http.StatusOK, res, s.Logger)
}

// FluxSuggestions returns a list of available Flux functions for the Flux Builder
func (s *Service) FluxSuggestions(w http.ResponseWriter, r *http.Request) {
	completer := complete.DefaultCompleter()
	names := completer.FunctionNames()
	var functions []SuggestionResponse
	for _, name := range names {
		suggestion, err := completer.FunctionSuggestion(name)
		if err != nil {
			Error(w, http.StatusNotFound, err.Error(), s.Logger)
			return
		}

		filteredParams := make(Params)
		for key, value := range suggestion.Params {
			if key == "table" {
				continue
			}

			filteredParams[key] = value
		}

		functions = append(functions, SuggestionResponse{
			Name:   name,
			Params: filteredParams,
		})
	}
	res := SuggestionsResponse{Functions: functions}

	encodeJSON(w, http.StatusOK, res, s.Logger)
}

// FluxSuggestion returns the function parameters for the requested function
func (s *Service) FluxSuggestion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := httprouter.GetParamFromContext(ctx, "name")
	completer := complete.DefaultCompleter()

	suggestion, err := completer.FunctionSuggestion(name)
	if err != nil {
		Error(w, http.StatusNotFound, err.Error(), s.Logger)
	}

	encodeJSON(w, http.StatusOK, SuggestionResponse{Name: name, Params: suggestion.Params}, s.Logger)
}

type ASTRequest struct {
	Body string `json:"body"`
}

type ASTResponse struct {
	Valid bool         `json:"valid"`
	AST   *ast.Program `json:"ast"`
	Error string       `json:"error"`
}

func (s *Service) FluxAST(w http.ResponseWriter, r *http.Request) {
	var request ASTRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		invalidJSON(w, s.Logger)
	}

	ast, err := parser.NewAST(request.Body)
	if err != nil {
		resp := ASTResponse{Valid: false, AST: nil, Error: err.Error()}
		encodeJSON(w, http.StatusOK, resp, s.Logger)
	} else {
		resp := ASTResponse{Valid: true, AST: ast, Error: ""}
		encodeJSON(w, http.StatusOK, resp, s.Logger)
	}
}

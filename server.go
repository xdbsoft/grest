package grest

import (
	"crypto/sha1"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/xdbsoft/grest/api"
	"github.com/xdbsoft/grest/oidc"
	"github.com/xdbsoft/grest/postgresql"
	"github.com/xdbsoft/grest/rules"
)

// Server instantiate a new grest server
func Server(cfg Config) (http.Handler, error) {

	r, err := postgresql.New(cfg.DBConnStr)
	if err != nil {
		return nil, err
	}

	err = r.Init()
	if err != nil {
		return nil, err
	}

	var a api.Authenticator
	if len(cfg.OpenIDConnectIssuer) > 0 {

		a, err = oidc.New(cfg.OpenIDConnectIssuer)
		if err != nil {
			return nil, err
		}
	}

	s := server{
		Authenticator:  a,
		DataRepository: r,
		RuleChecker:    rules.Checker{},
		Collections:    make(map[string]CollectionDefinition),
	}

	for _, c := range cfg.Collections {
		s.Collections[c.Name] = c
	}

	return &s, nil
}

type server struct {
	Collections    map[string]CollectionDefinition
	Authenticator  api.Authenticator
	DataRepository api.Repository
	RuleChecker    rules.Checker
}

func getLimit(limitString string) int {

	limit := 100
	limitValue, err := strconv.Atoi(limitString)
	if err == nil && limitValue < limit {
		limit = limitValue
	}
	return limit
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	user, err := s.authenticate(r)
	if err != nil {
		handleError(w, r, err)
		return
	}

	target, err := s.getTarget(r)
	if err != nil {
		handleError(w, r, err)
		return
	}
	log.Println(r.Method, target, "by", user)

	var data interface{}

	if target.IsDocument() {

		switch r.Method {
		case "GET":
			data, err = s.GetDocument(target, user)
		case "PUT":
			var payload api.Document
			if err := getPayload(r, &payload); err != nil {
				handleError(w, r, err)
				return
			}
			err = s.PutDocument(target, payload, user)
		case "POST", "PATCH":
			payload := make(api.DocumentProperties)
			if err := getPayload(r, &payload); err != nil {
				handleError(w, r, err)
				return
			}
			err = s.PatchDocument(target, payload, user)
		case "DELETE":
			err = s.DeleteDocument(target, user)
		default:
			handleError(w, r, badRequest("unsupported method"))
			return
		}
	} else {

		switch r.Method {
		case "GET":
			limit := getLimit(r.FormValue("limit"))
			orderBy := strings.Split(r.FormValue("orderBy"), "/")
			data, err = s.GetCollection(target, limit, orderBy, user)
		case "POST":
			payload := make(api.DocumentProperties)
			if err := getPayload(r, &payload); err != nil {
				handleError(w, r, err)
				return
			}
			data, err = s.AddDocument(target, payload, user)
		case "DELETE":
			err = s.DeleteCollection(target, user)
		default:
			handleError(w, r, badRequest("unsupported method"))
			return
		}
	}

	if err != nil {
		handleError(w, r, err)
		return
	}

	s.handleResponse(w, r, data)
}

func (s *server) authenticate(r *http.Request) (api.User, error) {
	if s.Authenticator == nil {
		return api.User{}, nil
	}

	return s.Authenticator.Authenticate(r)
}

func getPayload(r *http.Request, payload interface{}) error {
	if r.Body != nil {
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		err := d.Decode(&payload)
		if err != nil && err != io.EOF {
			return badRequest(errors.Wrap(err, "Unable to decode JSON body").Error())
		}
	}
	return nil
}

func (s *server) getTarget(r *http.Request) (api.ObjectRef, error) {

	items := strings.Split(r.URL.Path, "/")
	if len(items) > 0 && len(items[0]) == 0 {
		items = items[1:]
	}
	if len(items) == 0 {
		return api.ObjectRef{}, badRequest("empty path")
	}
	for _, item := range items {
		if len(item) == 0 {
			return api.ObjectRef{}, badRequest("empty item in path")
		}
	}

	return api.ObjectRef(items), nil
}

func (s *server) computeEtag(data interface{}) (string, error) {

	h := sha1.New()
	enc := gob.NewEncoder(h)
	err := enc.Encode(data)
	if err != nil {
		return "", err
	}

	return `"` + hex.EncodeToString(h.Sum(nil)) + `"`, nil
}

func (s *server) handleResponse(w http.ResponseWriter, r *http.Request, data interface{}) {

	if data == nil {
		w.WriteHeader(http.StatusNoContent)
	} else {

		// Handle ETag / If-None-Match
		etag, err := s.computeEtag(data)
		if err == nil && len(etag) > 0 {
			w.Header().Set("ETag", etag)

			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		// Handle Last-Modified / If-Modified-Since
		if c, ok := data.(api.Cacheable); ok {
			w.Header().Set("Last-Modified", c.GetLastModified().UTC().Format(http.TimeFormat))

			ifModifiedSince, err := http.ParseTime(r.Header.Get("If-Modified-Since"))
			if err == nil && !c.GetLastModified().After(ifModifiedSince) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Header().Add("Content-Type", "application/json")

		statusCode := http.StatusOK
		if r.Method == "POST" {
			statusCode = http.StatusAccepted
		}
		w.WriteHeader(statusCode)

		encoder := json.NewEncoder(w)

		print := r.FormValue("print")
		if print == "pretty" {
			encoder.SetIndent("", "  ")
		}

		err = encoder.Encode(data)
		if err != nil {
			handleError(w, r, err)
			return
		}
	}
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {

	log.Println(err)

	cause := errors.Cause(err)

	if IsBadRequest(cause) {
		http.Error(w, cause.Error(), http.StatusBadRequest)
		return
	}

	if IsNotAuthorized(cause) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if IsNotFound(cause) {
		http.Error(w, "Data not found", http.StatusNotFound)
		return
	}

	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func (s *server) checkIsAuthorized(target api.ObjectRef, user api.User, method rules.Method, rules []rules.Rule) error {

	ok, err := s.RuleChecker.Check(rules, target, user, method)
	if err != nil {
		return err
	}

	if !ok {
		return notAuthorizedError{target}
	}

	return nil
}

func (s *server) checkIsAuthorizedForCollection(target api.ObjectRef, user api.User, method rules.Method) error {

	collectionDef, ok := s.Collections[target[0]]
	if !ok {
		return notFoundError{target}
	}

	documentRef := append(target, "*")

	return s.checkIsAuthorized(documentRef, user, method, collectionDef.Rules)

}

func (s *server) checkIsAuthorizedForDoc(target api.ObjectRef, user api.User, method rules.Method) error {

	collectionDef, ok := s.Collections[target[0]]
	if !ok {
		return notFoundError{target}
	}

	return s.checkIsAuthorized(target, user, method, collectionDef.Rules)
}

func (s *server) GetDocument(target api.ObjectRef, user api.User) (interface{}, error) {

	if err := s.checkIsAuthorizedForDoc(target, user, rules.READ); err != nil {
		return nil, err
	}

	return s.DataRepository.Get(target)
}

func (s *server) GetCollection(target api.ObjectRef, limit int, orderBy []string, user api.User) (interface{}, error) {

	if err := s.checkIsAuthorizedForCollection(target, user, rules.READ); err != nil {
		return nil, err
	}

	features, err := s.DataRepository.GetAll(target, orderBy, limit)
	if err != nil {
		return nil, err
	}

	c := api.Collection{
		ID:       target.ID(),
		Features: features,
	}

	return c, nil
}

func (s *server) AddDocument(target api.ObjectRef, payload api.DocumentProperties, user api.User) (interface{}, error) {

	if err := s.checkIsAuthorizedForCollection(target, user, rules.WRITE); err != nil {
		return nil, err
	}

	return s.DataRepository.Add(target, payload)
}

func (s *server) PutDocument(target api.ObjectRef, payload api.Document, user api.User) error {

	if err := s.checkIsAuthorizedForDoc(target, user, rules.WRITE); err != nil {
		return err
	}

	if payload.ID != target.ID() {
		return badRequest("Invalid ID")
	}

	return s.DataRepository.Put(target, payload.Properties)
}

func (s *server) PatchDocument(target api.ObjectRef, payload api.DocumentProperties, user api.User) error {

	if err := s.checkIsAuthorizedForDoc(target, user, rules.WRITE); err != nil {
		return err
	}

	return s.DataRepository.Patch(target, payload)
}

func (s *server) DeleteDocument(target api.ObjectRef, user api.User) error {

	if err := s.checkIsAuthorizedForDoc(target, user, rules.DELETE); err != nil {
		return err
	}

	return s.DataRepository.Delete(target)
}

func (s *server) DeleteCollection(target api.ObjectRef, user api.User) error {

	if err := s.checkIsAuthorizedForCollection(target, user, rules.DELETE); err != nil {
		return err
	}

	return s.DataRepository.DeleteCollection(target)
}

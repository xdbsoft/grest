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
	"time"

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
		RuleChecker:    rules.NewChecker(cfg.Rules),
	}

	return &s, nil
}

type server struct {
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

	log.Println("Error: ", err)
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

func (s *server) GetRuleAndCheckPath(target api.ObjectRef, user api.User, isWrite bool) (rules.RuleCheck, error) {
	r := s.RuleChecker.SelectMatchingRule(target)

	if !r.IsValid() {
		return rules.RuleCheck{}, notAuthorizedError{target}
	}

	ok, err := r.CheckPath(user, false, s.GetDocumentFunc(user))
	if err != nil {
		return rules.RuleCheck{}, err
	}
	if !ok {
		return rules.RuleCheck{}, notAuthorizedError{target}
	}

	return r, nil
}

func (s *server) GetDocumentFunc(user api.User) func(api.ObjectRef) (api.Document, error) {
	return func(target api.ObjectRef) (api.Document, error) {
		return s.GetDocument(target, user)
	}
}

func (s *server) GetDocument(target api.ObjectRef, user api.User) (api.Document, error) {

	r, err := s.GetRuleAndCheckPath(target, user, false)
	if err != nil {
		return api.Document{}, err
	}

	tx, err := s.DataRepository.Begin()
	if err != nil {
		return api.Document{}, err
	}
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	data, err := tx.Get(target)
	if err != nil {
		return api.Document{}, err
	}

	ok, err := r.CheckContent(user, false, data, api.Document{}, s.GetDocumentFunc(user))
	if err != nil {
		return api.Document{}, err
	}
	if !ok {
		return api.Document{}, notAuthorizedError{target}
	}

	return data, nil
}

func (s *server) GetCollection(target api.ObjectRef, limit int, orderBy []string, user api.User) (interface{}, error) {

	r, err := s.GetRuleAndCheckPath(target, user, false)
	if err != nil {
		return nil, err
	}

	tx, err := s.DataRepository.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	cu, err := tx.GetAll(target, orderBy)
	if err != nil {
		return nil, err
	}
	defer cu.Close()

	c := api.Collection{
		ID: target.ID(),
	}
	for len(c.Features) < limit {

		fetched, err := cu.Fetch(10)
		if err != nil {
			return nil, err
		}
		for _, f := range fetched {

			ok, err := r.CheckContent(user, false, f, api.Document{}, s.GetDocumentFunc(user))
			if err != nil {
				return nil, err
			}

			if ok {
				c.Features = append(c.Features, f)
				if len(c.Features) == limit {
					break
				}
			}
		}
		if len(fetched) == 0 {
			break
		}
	}

	return c, nil
}

func (s *server) AddDocument(target api.ObjectRef, payload api.DocumentProperties, user api.User) (interface{}, error) {

	r, err := s.GetRuleAndCheckPath(target, user, true)
	if err != nil {
		return nil, err
	}

	tx, err := s.DataRepository.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	t := time.Now()
	newDoc := api.Document{
		ID:                   "*",
		CreationDate:         t,
		LastModificationDate: t,
		Properties:           payload,
	}

	ok, err := r.CheckContent(user, true, api.Document{}, newDoc, s.GetDocumentFunc(user))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, notAuthorizedError{target}
	}

	doc, err := tx.Add(target, payload)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func (s *server) PutDocument(target api.ObjectRef, payload api.Document, user api.User) error {

	if payload.ID != target.ID() {
		return badRequest("Invalid ID")
	}

	r, err := s.GetRuleAndCheckPath(target, user, true)
	if err != nil {
		return err
	}

	tx, err := s.DataRepository.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	t := time.Now()
	newDoc := api.Document{
		ID:                   target.ID(),
		CreationDate:         t,
		LastModificationDate: t,
		Properties:           payload.Properties,
	}

	data, err := tx.Get(target)
	if IsNotFound(err) {
		newDoc.CreationDate = data.CreationDate
	} else if err != nil {
		return err
	}

	ok, err := r.CheckContent(user, true, data, newDoc, s.GetDocumentFunc(user))
	if err != nil {
		return err
	}
	if !ok {
		return notAuthorizedError{target}
	}

	err = tx.Put(target, newDoc.Properties)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func patchPayload(data, patch map[string]interface{}) map[string]interface{} {

	res := make(map[string]interface{})

	for k := range data {

		if patched, found := patch[k]; found {

			dataChild, dataOk := data[k].(map[string]interface{})
			patchChild, patchOk := patched.(map[string]interface{})
			if dataOk && patchOk {
				res[k] = patchPayload(dataChild, patchChild)
			} else {
				res[k] = patched
			}

		} else {
			res[k] = data[k]
		}

	}

	for k := range patch {
		if _, found := data[k]; found {
			continue
		}

		res[k] = patch[k]
	}

	return res
}

func (s *server) PatchDocument(target api.ObjectRef, payload api.DocumentProperties, user api.User) error {

	r, err := s.GetRuleAndCheckPath(target, user, true)
	if err != nil {
		return err
	}

	tx, err := s.DataRepository.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	data, err := tx.Get(target)
	if err != nil {
		return err
	}

	newDoc := api.Document{
		ID:                   target.ID(),
		CreationDate:         data.CreationDate,
		LastModificationDate: time.Now(),
		Properties:           patchPayload(data.Properties, payload),
	}

	ok, err := r.CheckContent(user, true, data, newDoc, s.GetDocumentFunc(user))
	if err != nil {
		return err
	}
	if !ok {
		return notAuthorizedError{target}
	}

	err = tx.Patch(target, newDoc.Properties)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *server) DeleteDocument(target api.ObjectRef, user api.User) error {

	r, err := s.GetRuleAndCheckPath(target, user, true)
	if err != nil {
		return err
	}

	tx, err := s.DataRepository.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	data, err := tx.Get(target)
	if err != nil {
		return err
	}

	ok, err := r.CheckContent(user, true, data, api.Document{}, s.GetDocumentFunc(user))
	if err != nil {
		return err
	}
	if !ok {
		return notAuthorizedError{target}
	}

	err = tx.Delete(target)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *server) DeleteCollection(target api.ObjectRef, user api.User) error {

	r, err := s.GetRuleAndCheckPath(target, user, true)
	if err != nil {
		return err
	}

	tx, err := s.DataRepository.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	cu, err := tx.GetAll(target, nil)
	if err != nil {
		return err
	}
	defer cu.Close()

	data, err := cu.Fetch(10)
	if err != nil {
		return err
	}
	for len(data) > 0 {

		for _, d := range data {
			ok, err := r.CheckContent(user, true, d, api.Document{}, s.GetDocumentFunc(user))
			if err != nil {
				return err
			}
			if ok {
				documentRef := append(target, d.ID)
				tx.Delete(documentRef)
			}
		}

		data, err = cu.Fetch(10)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

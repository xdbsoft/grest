package rules

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/xdbsoft/grest/api"
	"github.com/xdbsoft/gript"
)

type Checker struct {
	rules []Rule
}

type RetrievalFunc func(api.ObjectRef, api.User) (api.Document, error)

func NewChecker(rules []Rule) Checker {
	return Checker{rules: rules}
}

func isVariable(s string) (bool, string) {

	if len(s) >= 3 {
		if s[0] == '{' && s[len(s)-1] == '}' {
			return true, s[1 : len(s)-1]
		}
	}
	return false, ""
}

func checkCondition(condition string, variables map[string]interface{}) (bool, error) {
	if len(condition) == 0 {
		return true, nil
	}
	r, err := gript.Eval(condition, variables)
	if err != nil {
		return false, err
	}
	result, ok := r.(bool)
	if !ok {
		return false, errors.New("Invalid condition: result is not boolean")
	}
	return result, nil
}

type RuleCheck struct {
	rule          Rule
	pathVariables map[string]interface{}
	user          api.User
}

func (r RuleCheck) IsValid() bool {
	return len(r.rule.Path) > 0
}

func (c Checker) SelectMatchingRule(target api.ObjectRef, user api.User) RuleCheck {

	docTarget := target
	if !docTarget.IsDocument() {
		docTarget = append(docTarget, "*")
	}

	for _, rule := range c.rules {

		path := strings.Split(rule.Path, "/")

		pathVariables := make(map[string]interface{})
		match := false
		if len(docTarget) == len(path) {

			match = true
			for i := range path {

				if isVar, name := isVariable(path[i]); isVar {
					pathVariables[name] = docTarget[i]
				} else {
					if path[i] != docTarget[i] {
						match = false
						break
					}
				}
			}
		}

		if match {
			return RuleCheck{
				rule:          rule,
				pathVariables: pathVariables,
				user:          user,
			}
		}
	}

	return RuleCheck{}
}

func (r RuleCheck) RetrieveWith(a Allow, get RetrievalFunc) map[string]interface{} {

	withContent := make(map[string]interface{})
	for _, w := range a.With {

		//Replace variables
		path := strings.Split(w.Path, "/")
		for i := range path {
			if ok, v := isVariable(path[i]); ok {
				splittedVar := strings.Split(v, ".")
				if len(splittedVar) == 1 {
					path[i] = fmt.Sprint(r.pathVariables[splittedVar[0]])
				} else if len(splittedVar) == 2 && splittedVar[0] == "path" {
					path[i] = fmt.Sprint(r.pathVariables[splittedVar[1]])
				} else if len(splittedVar) == 2 && splittedVar[0] == "user" {
					switch splittedVar[1] {
					case "id":
						path[i] = r.user.ID
					case "name":
						path[i] = r.user.Name
					case "email":
						path[i] = r.user.Email
					default:
						path[i] = "<nil>"
					}
				} else {
					path[i] = "<nil>"
				}
			}
		}

		//Get requested item
		target := api.ObjectRef(path)
		item, err := get(target, r.user)
		if err == nil {
			withContent[w.Name] = item
		} else {
			withContent[w.Name] = nil
		}
	}
	return withContent
}

func (r RuleCheck) CheckPath(isWrite bool, get RetrievalFunc) (bool, error) {

	a := r.rule.Read
	if isWrite {
		a = r.rule.Write
	}

	withContent := r.RetrieveWith(a, get)

	variables := map[string]interface{}{
		"path": r.pathVariables,
		"user": r.user,
		"with": withContent,
	}

	return checkCondition(a.IfPath, variables)
}

func (r RuleCheck) PrepareCheckContent(isWrite bool, get RetrievalFunc) RuleCheckForContent {

	a := r.rule.Read
	if isWrite {
		a = r.rule.Write
	}

	withContent := r.RetrieveWith(a, get)

	return RuleCheckForContent{
		IfContent:     a.IfContent,
		User:          r.user,
		PathVariables: r.pathVariables,
		WithContent:   withContent,
	}
}

type RuleCheckForContent struct {
	IfContent     string
	User          api.User
	PathVariables map[string]interface{}
	WithContent   map[string]interface{}
}

func (r RuleCheckForContent) Check(content api.Document, newContent api.Document) (bool, error) {

	variables := map[string]interface{}{
		"path":       r.PathVariables,
		"user":       r.User,
		"content":    content,
		"newContent": newContent,
		"with":       r.WithContent,
	}
	if len(content.ID) == 0 {
		variables["content"] = nil
	}
	if len(newContent.ID) == 0 {
		variables["newContent"] = nil
	}

	return checkCondition(r.IfContent, variables)
}

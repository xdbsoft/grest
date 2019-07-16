package api

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/xdbsoft/gript"
)

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

type RuleChecker struct{}

func isVariable(s string) (bool, string) {

	if len(s) >= 3 {
		if s[0] == '{' && s[len(s)-1] == '}' {
			return true, s[1 : len(s)-1]
		}
	}
	return false, ""
}

func (c RuleChecker) CheckCondition(condition string, variables map[string]interface{}) (bool, error) {
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

func (c RuleChecker) Check(rules []Rule, target DocumentRef, user User, method Method) (bool, error) {

	for _, rule := range rules {

		path := ObjectRef(strings.Split(rule.Path, "/"))

		variables := make(map[string]interface{})
		pathVariables := make(map[string]interface{})
		match := false
		if len(target) >= len(path) {

			match = true
			for i := range path {

				if isVar, name := isVariable(path[i]); isVar {
					pathVariables[name] = target[i]
				} else {
					if path[i] != target[i] {
						match = false
						break
					}
				}
			}
		}

		variables["path"] = pathVariables
		variables["user"] = map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		}

		if match {
			for _, a := range rule.Allow {

				found := false
				for _, am := range a.Methods {
					if am == method {
						found = true
						break
					}
				}

				if found {
					if len(a.If) > 0 {

						ok, err := c.CheckCondition(a.If, variables)
						if err != nil {
							return false, err
						}

						if !ok {
							return false, nil
						}
					}
					return true, nil
				}

			}
		}
	}
	return true, nil
}

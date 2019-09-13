package rules

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/xdbsoft/grest/api"
	"github.com/xdbsoft/gript"
)

type Checker struct{}

func isVariable(s string) (bool, string) {

	if len(s) >= 3 {
		if s[0] == '{' && s[len(s)-1] == '}' {
			return true, s[1 : len(s)-1]
		}
	}
	return false, ""
}

func (c Checker) checkCondition(condition string, variables map[string]interface{}) (bool, error) {
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

func (c Checker) checkAllow(allow Allow, method Method, variables map[string]interface{}) (bool, error) {
	methodFound := false
	for _, am := range allow.Methods {
		if am == method {
			methodFound = true
			break
		}
	}

	if !methodFound {
		return false, nil
	}

	if len(allow.If) > 0 {

		ok, err := c.checkCondition(allow.If, variables)
		if err != nil {
			return false, err
		}

		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func (c Checker) Check(rules []Rule, target api.ObjectRef, user api.User, method Method) (bool, error) {

	matchCount := 0
	for _, rule := range rules {

		path := strings.Split(rule.Path, "/")

		pathVariables := make(map[string]interface{})
		match := false
		if len(target) == len(path) {

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

		if match {
			matchCount++

			variables := make(map[string]interface{})
			variables["path"] = pathVariables
			variables["user"] = map[string]interface{}{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
			}

			for _, a := range rule.Allow {

				allowed, err := c.checkAllow(a, method, variables)
				if err != nil {
					return false, err
				}
				if allowed {
					return true, nil
				}
			}
		}
	}
	var err error
	if matchCount == 0 {
		err = errors.New("No matched rules")
	}
	return false, err
}

// Copyright 2021 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validate

import (
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"k8s.io/kube-openapi/pkg/validation/errors"
	"k8s.io/kube-openapi/pkg/validation/spec"
	celmodel "k8s.io/kube-openapi/third_party/forked/celopenapi/model"
	"reflect"
)

// CelRules defines the format of the x-kubernetes-validator schema extension.
type CelRules []CelRule

// CelRule defines the format of each rule in CelRules.
type CelRule struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

func newCelExpressionValidator(path string, schema *spec.Schema, rules CelRules) *celExpressionValidator {
	env, _ := cel.NewEnv()
	reg := celmodel.NewRegistry(env)
	rt, err := celmodel.NewRuleTypes("__root__", schema, reg)
	if err != nil {
		panic(err)
	}
	opts, err := rt.EnvOptions(env.TypeProvider())
	if err != nil {
		panic(err)
	}

	var propDecls []*expr.Decl
	if root, ok := rt.FindDeclType("__root__"); ok {
		if root.IsObject() {
			for k, f := range root.Fields {
				propDecls = append(propDecls, decls.NewVar(k, f.Type.ExprType()))
			}
		}
		// TODO: handle types other than object
	}
	opts = append(opts, cel.Declarations(propDecls...))
	env, err = env.Extend(opts...)
	if err != nil {
		panic(err)
	}

	programs := make([]cel.Program, len(rules))
	for i, rule := range rules {
		ast, issues := env.Compile(rule.Rule)
		if issues != nil {
			panic(fmt.Errorf("compilation failed: %v", issues)) // TODO: remove when we pre-compile cel rules
		}
		// TODO: add type checking information (Decls)
		prog, err := env.Program(ast)
		if err != nil {
			panic(fmt.Errorf("program instantiation failed: %v", err)) // TODO: remove when we pre-compile cel rules
		}
		programs[i] = prog
	}
	return &celExpressionValidator{Path: path, Rules: rules, Programs: programs}
}

type celExpressionValidator struct {
	Path     string
	Rules    CelRules
	Programs []cel.Program
}

func (c celExpressionValidator) SetPath(path string) {
	c.Path = path
}

func (c celExpressionValidator) Applies(source interface{}, _ reflect.Kind) bool {
	switch source.(type) {
	case *spec.Schema:
		return true
	}
	return false
}

func (c celExpressionValidator) Validate(data interface{}) *Result {
	// TODO: convert from Unstructured to CEL types
	res := new(Result)
	for i, program := range c.Programs {
		rule := c.Rules[i]
		evalResult, _, err := program.Eval(data)
		if err != nil {
			res.AddErrors(errors.ErrorExecutingValidatorRule(c.Path, "", rule.Rule, err, data))
			continue
		}
		if evalResult.Value() != true {
			res.AddErrors(errors.FailedValidatorRule(c.Path, "", rule.Rule, rule.Message, data))
		}
	}
	return res
}

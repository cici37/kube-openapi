/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cel

import (
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"k8s.io/kube-openapi/pkg/validation/spec"
	celmodel "k8s.io/kube-openapi/third_party/forked/celopenapi/model"
)

// CelRules defines the format of the x-kubernetes-validator schema extension.
type CelRules []CelRule

// CelRule defines the format of each rule in CelRules.
type CelRule struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Compile compiles all the CEL validation rules in the CelRules and returns a slice containing a compiled program for each provided CelRule, or an array of errors.
func Compile(schema *spec.Schema, celRules CelRules) ([]cel.Program, []error) {
	var scopedTypeName = "self"
	var allErrors []error
	var propDecls []*expr.Decl
	var root *celmodel.DeclType
	var ok bool
	env, _ := cel.NewEnv()
	reg := celmodel.NewRegistry(env)
	rt, err := celmodel.NewRuleTypes(scopedTypeName, schema, reg)
	if err != nil {
		allErrors = append(allErrors, err)
		return nil, allErrors
	}
	opts, err := rt.EnvOptions(env.TypeProvider())
	root, ok = rt.FindDeclType(scopedTypeName)
	if !ok {
		root = celmodel.SchemaDeclType(schema).MaybeAssignTypeName(scopedTypeName)
	}
	if root.IsObject() {
		for k, f := range root.Fields {
			propDecls = append(propDecls, decls.NewVar(k, f.Type.ExprType()))
		}
	}
	propDecls = append(propDecls, decls.NewVar(scopedTypeName, root.ExprType()))
	opts = append(opts, cel.Declarations(propDecls...))
	env, err = env.Extend(opts...)
	if err != nil {
		allErrors = append(allErrors, err)
		return nil, allErrors
	}
	programs := make([]cel.Program, len(celRules))
	for i, rule := range celRules {
		ast, issues := env.Compile(rule.Rule)
		if issues != nil {
			allErrors = append(allErrors, fmt.Errorf("compilation failed for rule: %v with message: %v", rule.Message, issues.Err()))
		} else {
			prog, err := env.Program(ast)
			if err != nil {
				allErrors = append(allErrors, fmt.Errorf("program instantiation failed for rule: %v with message: %v", rule.Message, err))
			} else {
				programs[i] = prog
			}
		}
	}

	return programs, allErrors
}

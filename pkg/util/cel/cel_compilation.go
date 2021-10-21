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
	celext "github.com/google/cel-go/ext"
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

// Compile is used for cel compilation.
// rootName is used as type name of provided schema
func Compile(schema *spec.Schema, CelRules CelRules, rootName string) ([]cel.Program, []error) {
	if len(rootName) == 0 {
		rootName = "self"
	}
	var allErrors []error
	var propDecls []*expr.Decl
	env, _ := cel.NewEnv()
	if schema.Type.Contains("string") || schema.Type.Contains("integer") || schema.Type.Contains("boolean") || schema.Type.Contains("number") {
		switch schema.Type[0] {
		case "string":
			propDecls = append(propDecls, decls.NewVar(rootName, decls.String))
		case "integer":
			propDecls = append(propDecls, decls.NewVar(rootName, decls.Int))
		case "boolean":
			propDecls = append(propDecls, decls.NewVar(rootName, decls.Bool))
		case "number":
			propDecls = append(propDecls, decls.NewVar(rootName, decls.Double))
		default:
			allErrors = append(allErrors, fmt.Errorf("not supported for type: %v", schema.Type[0]))
			return nil, allErrors
		}
		var err error
		env, err = cel.NewEnv(
			celext.Strings(),
			celext.Encoders(),
			cel.Declarations(propDecls...))
		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("error initializing CEL environment: %w", err))
			return nil, allErrors
		}
	} else {
		reg := celmodel.NewRegistry(env)
		rt, err := celmodel.NewRuleTypes(rootName, schema, reg)
		if err != nil {
			allErrors = append(allErrors, err)
			return nil, allErrors
		}
		opts, err := rt.EnvOptions(env.TypeProvider())
		if err != nil {
			allErrors = append(allErrors, err)
			return nil, allErrors
		}

		if root, ok := rt.FindDeclType(rootName); ok {
			if root.IsObject() || root.IsMap() {
				for k, f := range root.Fields {
					propDecls = append(propDecls, decls.NewVar(k, f.Type.ExprType()))
				}
			} else if root.IsList() {

			} else {
				allErrors = append(allErrors, fmt.Errorf("unsupported type"))
				return nil, allErrors
			}
		}
		opts = append(opts, cel.Declarations(propDecls...))
		env, err = env.Extend(opts...)
		if err != nil {
			allErrors = append(allErrors, err)
			return nil, allErrors
		}
	}

	programs := make([]cel.Program, len(CelRules))
	for i, rule := range CelRules {
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

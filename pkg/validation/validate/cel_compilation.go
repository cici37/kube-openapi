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

package validate

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

func CompileCel(schema *spec.Schema, CelRules CelRules, rootName string) ([]cel.Program, []error) {
	if len(rootName) == 0 {
		rootName = "__root__"
	}
	allErrors := []error{}
	env, _ := cel.NewEnv()
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

	var propDecls []*expr.Decl
	if root, ok := rt.FindDeclType(rootName); ok {
		if root.IsObject() {
			for k, f := range root.Fields {
				propDecls = append(propDecls, decls.NewVar(k, f.Type.ExprType()))
			}
		}
	}
	opts = append(opts, cel.Declarations(propDecls...))
	env, err = env.Extend(opts...)
	if err != nil {
		allErrors = append(allErrors, err)
		return nil, allErrors
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

func CompileCelForScalarElement(CelRules CelRules, schemaName string, schemaType string) ([]cel.Program, []error) {
	allErrors := []error{}

	var celDecls []*expr.Decl
	switch schemaType {
	case "string":
		celDecls = append(celDecls, decls.NewVar(schemaName, decls.String))
	case "integer":
		celDecls = append(celDecls, decls.NewVar(schemaName, decls.Int))
	case "boolean":
		celDecls = append(celDecls, decls.NewVar(schemaName, decls.Bool))
	default:
		allErrors = append(allErrors, fmt.Errorf("not supported for type: %v", schemaType))
		return nil, allErrors
	}
	env, err := cel.NewEnv(
		celext.Strings(),
		celext.Encoders(),
		cel.Declarations(celDecls...))
	if err != nil {
		allErrors = append(allErrors, fmt.Errorf("error initializing CEL environment: %w", err))
		return nil, allErrors
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

/*
Copyright 2021 The Kubernetes Authors.
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
	"k8s.io/kube-openapi/pkg/validation/spec"
	"testing"
)

func TestCelValueValidator(t *testing.T) {
	schema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{"object"},
			Properties: map[string]spec.Schema{
				"minReplicas": {
					SchemaProps: spec.SchemaProps{
						Type:   []string{"integer"},
						Format: "int64",
					},
				},
				"maxReplicas": {
					SchemaProps: spec.SchemaProps{
						Type:   []string{"integer"},
						Format: "int64",
					},
				},
				"nestedObj": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"val": {
								SchemaProps: spec.SchemaProps{
									Type:   []string{"integer"},
									Format: "int64",
								},
							},
						},
					},
				},
			},
		},
	}
	cases := []struct {
		name    string
		input   map[string]interface{}
		expr    string
		isValid bool
	}{
		{
			name: "valid",
			input: map[string]interface{}{
				"minReplicas": int64(5),
				"maxReplicas": int64(10),
			},
			expr:    "minReplicas < maxReplicas",
			isValid: true,
		},
		{
			name: "valid nested",
			input: map[string]interface{}{
				"nestedObj": map[string]interface{}{
					"val": int64(10),
				},
			},
			expr:    "nestedObj.val == 10",
			isValid: true,
		},
		{
			name: "invalid",
			input: map[string]interface{}{
				"minReplicas": int64(11),
				"maxReplicas": int64(10),
			},
			expr:    "minReplicas < maxReplicas",
			isValid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			validator := newCelExpressionValidator("", schema, []CelRule{{Rule: tc.expr}})
			result := validator.Validate(tc.input)
			if result.IsValid() != tc.isValid {
				t.Fatalf("Expected isValid=%t, but got %t. Errors: %v", tc.isValid, result.IsValid(), result.Errors)
			}
		})
	}
}

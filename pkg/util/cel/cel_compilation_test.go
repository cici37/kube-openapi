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

package cel

import (
	"k8s.io/kube-openapi/pkg/validation/spec"
	"testing"
)

func TestCelCompilation(t *testing.T) {
	cases := []struct {
		name      string
		input     spec.Schema
		expr      string
		wantError bool
	}{
		{
			name: "valid",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
					Properties: map[string]spec.Schema{
						"minReplicas": {
							SchemaProps: spec.SchemaProps{
								Type: []string{"integer"},
							},
						},
						"maxReplicas": {
							SchemaProps: spec.SchemaProps{
								Type: []string{"integer"},
							},
						},
					},
				},
			},
			expr:      "minReplicas < maxReplicas",
			wantError: false,
		},
		{
			name: "valid nested",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
					Properties: map[string]spec.Schema{
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
			},
			expr:      "nestedObj.val == 10",
			wantError: false,
		},
		{
			name: "valid for scalar element",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
					Properties: map[string]spec.Schema{
						"stringTest": {
							SchemaProps: spec.SchemaProps{
								Type: []string{"string"},
							},
						},
					},
				},
			},
			expr:      "stringTest.startsWith('s')",
			wantError: false,
		},
		{
			name: "valid for string",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"string"},
				},
			},
			expr:      "self.startsWith('s')",
			wantError: false,
		},
		{
			name: "valid for boolean",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"boolean"},
				},
			},
			expr:      "self == true",
			wantError: false,
		},
		{
			name: "valid for integer",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"integer"},
				},
			},
			expr:      "self > 0",
			wantError: false,
		},
		{
			name: "valid nested array",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"array"},
					Items: &spec.SchemaOrArray{
						Schema: &spec.Schema{
							SchemaProps: spec.SchemaProps{
								Type: []string{"array"},
								Items: &spec.SchemaOrArray{
									Schema: &spec.Schema{
										SchemaProps: spec.SchemaProps{
											Type: []string{"array"},
											Items: &spec.SchemaOrArray{
												Schema: &spec.Schema{
													SchemaProps: spec.SchemaProps{
														Type: []string{"string"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expr:      "size(self.items) == 10",
			wantError: false,
		},
		{
			name: "valid nested array of object",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"array"},
					Items: &spec.SchemaOrArray{
						Schema: &spec.Schema{
							SchemaProps: spec.SchemaProps{
								Type: []string{"object"},
								Properties: map[string]spec.Schema{
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
						},
					},
				},
			},
			expr:      "size(self) == 10",
			wantError: false,
		},
		{
			name: "valid nested array of object",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"array"},
					Items: &spec.SchemaOrArray{
						Schema: &spec.Schema{
							SchemaProps: spec.SchemaProps{
								Type: []string{"object"},
								Properties: map[string]spec.Schema{
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
						},
					},
				},
			},
			expr:      "val == 1",
			wantError: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, allErrors := Compile(&tt.input, []CelRule{{Rule: tt.expr}}, "")
			if !tt.wantError && len(allErrors) > 0 {
				t.Errorf("Expected no error, but got: %v", allErrors)
			} else if tt.wantError && len(allErrors) == 0 {
				t.Error("Expected error, but got none")
			}
		})
	}
}

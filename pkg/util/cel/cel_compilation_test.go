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
	"fmt"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"strings"
	"testing"
)

func TestCelCompilation(t *testing.T) {
	cases := []struct {
		name              string
		input             spec.Schema
		expr              string
		errMessage        string
		wantError         bool
		checkErrorMessage bool
	}{
		{
			name: "valid object",
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
			expr:              "minReplicas < maxReplicas",
			errMessage:        "minReplicas should be smaller than maxReplicas",
			wantError:         false,
			checkErrorMessage: false,
		},
		{
			name: "valid for string",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"string"},
				},
			},
			expr:              "self.startsWith('s')",
			errMessage:        "scoped field should start with 's'",
			wantError:         false,
			checkErrorMessage: false,
		},
		{
			name: "valid for byte",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type:   []string{"string"},
					Format: "byte",
				},
			},
			expr:              "string(self).endsWith('s')",
			errMessage:        "scoped field should end with 's'",
			wantError:         false,
			checkErrorMessage: false,
		},
		{
			name: "valid for boolean",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"boolean"},
				},
			},
			expr:              "self == true",
			errMessage:        "scoped field should be true",
			wantError:         false,
			checkErrorMessage: false,
		},
		{
			name: "valid for integer",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"integer"},
				},
			},
			expr:              "self > 0",
			errMessage:        "scoped field should be greater than 0",
			wantError:         false,
			checkErrorMessage: false,
		},
		{
			name: "valid for number",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"number"},
				},
			},
			expr:              "self > 1.0",
			errMessage:        "scoped field should be greater than 1.0",
			wantError:         false,
			checkErrorMessage: false,
		},
		{
			name: "valid nested object of object",
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
			expr:              "nestedObj.val == 10",
			errMessage:        "val should be equal to 10",
			wantError:         false,
			checkErrorMessage: false,
		},
		{
			name: "valid nested object of array",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
					Properties: map[string]spec.Schema{
						"nestedObj": {
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
			expr:              "size(self.nestedObj[0]) == 10",
			errMessage:        "size of first element in nestedObj should be equal to 10",
			wantError:         false,
			checkErrorMessage: false,
		},
		{
			name: "valid nested array of array",
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
			expr:              "size(self[0][0]) == 10",
			errMessage:        "size of items under items of scoped field should be equal to 10",
			wantError:         false,
			checkErrorMessage: false,
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
			expr:              "self[0].nestedObj.val == 10",
			errMessage:        "val under nestedObj under properties under items should be equal to 10",
			wantError:         false,
			checkErrorMessage: false,
		},
		{
			name: "valid map",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
					AdditionalProperties: &spec.SchemaOrBool{
						Allows: true,
						Schema: &spec.Schema{
							SchemaProps: spec.SchemaProps{
								Type:     []string{"boolean"},
								Nullable: false,
							},
						},
					},
				},
			},
			expr:              "size(self) > 0",
			errMessage:        "size of scoped field should be greater than 0",
			wantError:         false,
			checkErrorMessage: false,
		},
		{
			name: "invalid checking for number",
			input: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"number"},
				},
			},
			expr:              "size(self) == 10",
			errMessage:        "size of scoped field should be equal to 10",
			wantError:         true,
			checkErrorMessage: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, allErrors := Compile(&tt.input, []CelRule{{Rule: tt.expr, Message: tt.errMessage}})
			if tt.checkErrorMessage {
				var pass = false
				for _, err := range allErrors {
					if strings.Contains(fmt.Sprint(err), tt.errMessage) {
						pass = true
					}
				}
				if !pass {
					t.Errorf("Expected error massage contains: %v, but got error: %v", tt.errMessage, allErrors)
				}
			} else {
				if !tt.wantError && len(allErrors) > 0 {
					t.Errorf("Expected no error, but got: %v", allErrors)
				} else if tt.wantError && len(allErrors) == 0 {
					t.Error("Expected error, but got none")
				}
			}
		})
	}
}

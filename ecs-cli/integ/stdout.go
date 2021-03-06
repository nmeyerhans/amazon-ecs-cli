// +build integ

// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package integ

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// Stdout is the standard output content from running a test.
type Stdout []byte

// HasAllSnippets returns true if stdout contains each snippet in wantedSnippets, false otherwise.
func (b Stdout) HasAllSnippets(t *testing.T, wantedSnippets []string) bool {
	s := string(b)
	for _, snippet := range wantedSnippets {
		if !assert.Contains(t, s, snippet) {
			return false
		}
	}
	return true
}

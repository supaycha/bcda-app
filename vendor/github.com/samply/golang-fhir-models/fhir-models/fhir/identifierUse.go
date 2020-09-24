// Copyright 2019 The Samply Development Community
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fhir

import (
	"encoding/json"
	"fmt"
	"strings"
)

// THIS FILE IS GENERATED BY https://github.com/samply/golang-fhir-models
// PLEASE DO NOT EDIT BY HAND

// IdentifierUse is documented here http://hl7.org/fhir/ValueSet/identifier-use
type IdentifierUse int

const (
	IdentifierUseUsual IdentifierUse = iota
	IdentifierUseOfficial
	IdentifierUseTemp
	IdentifierUseSecondary
	IdentifierUseOld
)

func (code IdentifierUse) MarshalJSON() ([]byte, error) {
	return json.Marshal(code.Code())
}
func (code *IdentifierUse) UnmarshalJSON(json []byte) error {
	s := strings.Trim(string(json), "\"")
	switch s {
	case "usual":
		*code = IdentifierUseUsual
	case "official":
		*code = IdentifierUseOfficial
	case "temp":
		*code = IdentifierUseTemp
	case "secondary":
		*code = IdentifierUseSecondary
	case "old":
		*code = IdentifierUseOld
	default:
		return fmt.Errorf("unknown IdentifierUse code `%s`", s)
	}
	return nil
}
func (code IdentifierUse) String() string {
	return code.Code()
}
func (code IdentifierUse) Code() string {
	switch code {
	case IdentifierUseUsual:
		return "usual"
	case IdentifierUseOfficial:
		return "official"
	case IdentifierUseTemp:
		return "temp"
	case IdentifierUseSecondary:
		return "secondary"
	case IdentifierUseOld:
		return "old"
	}
	return "<unknown>"
}
func (code IdentifierUse) Display() string {
	switch code {
	case IdentifierUseUsual:
		return "Usual"
	case IdentifierUseOfficial:
		return "Official"
	case IdentifierUseTemp:
		return "Temp"
	case IdentifierUseSecondary:
		return "Secondary"
	case IdentifierUseOld:
		return "Old"
	}
	return "<unknown>"
}
func (code IdentifierUse) Definition() string {
	switch code {
	case IdentifierUseUsual:
		return "The identifier recommended for display and use in real-world interactions."
	case IdentifierUseOfficial:
		return "The identifier considered to be most trusted for the identification of this item. Sometimes also known as \"primary\" and \"main\". The determination of \"official\" is subjective and implementation guides often provide additional guidelines for use."
	case IdentifierUseTemp:
		return "A temporary identifier."
	case IdentifierUseSecondary:
		return "An identifier that was assigned in secondary use - it serves to identify the object in a relative context, but cannot be consistently assigned to the same object again in a different context."
	case IdentifierUseOld:
		return "The identifier id no longer considered valid, but may be relevant for search purposes.  E.g. Changes to identifier schemes, account merges, etc."
	}
	return "<unknown>"
}

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

import "encoding/json"

// THIS FILE IS GENERATED BY https://github.com/samply/golang-fhir-models
// PLEASE DO NOT EDIT BY HAND

// Measure is documented here http://hl7.org/fhir/StructureDefinition/Measure
type Measure struct {
	Id                              *string                   `bson:"id,omitempty" json:"id,omitempty"`
	Meta                            *Meta                     `bson:"meta,omitempty" json:"meta,omitempty"`
	ImplicitRules                   *string                   `bson:"implicitRules,omitempty" json:"implicitRules,omitempty"`
	Language                        *string                   `bson:"language,omitempty" json:"language,omitempty"`
	Text                            *Narrative                `bson:"text,omitempty" json:"text,omitempty"`
	Extension                       []Extension               `bson:"extension,omitempty" json:"extension,omitempty"`
	ModifierExtension               []Extension               `bson:"modifierExtension,omitempty" json:"modifierExtension,omitempty"`
	Url                             *string                   `bson:"url,omitempty" json:"url,omitempty"`
	Identifier                      []Identifier              `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version                         *string                   `bson:"version,omitempty" json:"version,omitempty"`
	Name                            *string                   `bson:"name,omitempty" json:"name,omitempty"`
	Title                           *string                   `bson:"title,omitempty" json:"title,omitempty"`
	Subtitle                        *string                   `bson:"subtitle,omitempty" json:"subtitle,omitempty"`
	Status                          PublicationStatus         `bson:"status" json:"status"`
	Experimental                    *bool                     `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date                            *string                   `bson:"date,omitempty" json:"date,omitempty"`
	Publisher                       *string                   `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact                         []ContactDetail           `bson:"contact,omitempty" json:"contact,omitempty"`
	Description                     *string                   `bson:"description,omitempty" json:"description,omitempty"`
	UseContext                      []UsageContext            `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction                    []CodeableConcept         `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Purpose                         *string                   `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Usage                           *string                   `bson:"usage,omitempty" json:"usage,omitempty"`
	Copyright                       *string                   `bson:"copyright,omitempty" json:"copyright,omitempty"`
	ApprovalDate                    *string                   `bson:"approvalDate,omitempty" json:"approvalDate,omitempty"`
	LastReviewDate                  *string                   `bson:"lastReviewDate,omitempty" json:"lastReviewDate,omitempty"`
	EffectivePeriod                 *Period                   `bson:"effectivePeriod,omitempty" json:"effectivePeriod,omitempty"`
	Topic                           []CodeableConcept         `bson:"topic,omitempty" json:"topic,omitempty"`
	Author                          []ContactDetail           `bson:"author,omitempty" json:"author,omitempty"`
	Editor                          []ContactDetail           `bson:"editor,omitempty" json:"editor,omitempty"`
	Reviewer                        []ContactDetail           `bson:"reviewer,omitempty" json:"reviewer,omitempty"`
	Endorser                        []ContactDetail           `bson:"endorser,omitempty" json:"endorser,omitempty"`
	RelatedArtifact                 []RelatedArtifact         `bson:"relatedArtifact,omitempty" json:"relatedArtifact,omitempty"`
	Library                         []string                  `bson:"library,omitempty" json:"library,omitempty"`
	Disclaimer                      *string                   `bson:"disclaimer,omitempty" json:"disclaimer,omitempty"`
	Scoring                         *CodeableConcept          `bson:"scoring,omitempty" json:"scoring,omitempty"`
	CompositeScoring                *CodeableConcept          `bson:"compositeScoring,omitempty" json:"compositeScoring,omitempty"`
	Type                            []CodeableConcept         `bson:"type,omitempty" json:"type,omitempty"`
	RiskAdjustment                  *string                   `bson:"riskAdjustment,omitempty" json:"riskAdjustment,omitempty"`
	RateAggregation                 *string                   `bson:"rateAggregation,omitempty" json:"rateAggregation,omitempty"`
	Rationale                       *string                   `bson:"rationale,omitempty" json:"rationale,omitempty"`
	ClinicalRecommendationStatement *string                   `bson:"clinicalRecommendationStatement,omitempty" json:"clinicalRecommendationStatement,omitempty"`
	ImprovementNotation             *CodeableConcept          `bson:"improvementNotation,omitempty" json:"improvementNotation,omitempty"`
	Definition                      []string                  `bson:"definition,omitempty" json:"definition,omitempty"`
	Guidance                        *string                   `bson:"guidance,omitempty" json:"guidance,omitempty"`
	Group                           []MeasureGroup            `bson:"group,omitempty" json:"group,omitempty"`
	SupplementalData                []MeasureSupplementalData `bson:"supplementalData,omitempty" json:"supplementalData,omitempty"`
}
type MeasureGroup struct {
	Id                *string                  `bson:"id,omitempty" json:"id,omitempty"`
	Extension         []Extension              `bson:"extension,omitempty" json:"extension,omitempty"`
	ModifierExtension []Extension              `bson:"modifierExtension,omitempty" json:"modifierExtension,omitempty"`
	Code              *CodeableConcept         `bson:"code,omitempty" json:"code,omitempty"`
	Description       *string                  `bson:"description,omitempty" json:"description,omitempty"`
	Population        []MeasureGroupPopulation `bson:"population,omitempty" json:"population,omitempty"`
	Stratifier        []MeasureGroupStratifier `bson:"stratifier,omitempty" json:"stratifier,omitempty"`
}
type MeasureGroupPopulation struct {
	Id                *string          `bson:"id,omitempty" json:"id,omitempty"`
	Extension         []Extension      `bson:"extension,omitempty" json:"extension,omitempty"`
	ModifierExtension []Extension      `bson:"modifierExtension,omitempty" json:"modifierExtension,omitempty"`
	Code              *CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
	Description       *string          `bson:"description,omitempty" json:"description,omitempty"`
	Criteria          Expression       `bson:"criteria" json:"criteria"`
}
type MeasureGroupStratifier struct {
	Id                *string                           `bson:"id,omitempty" json:"id,omitempty"`
	Extension         []Extension                       `bson:"extension,omitempty" json:"extension,omitempty"`
	ModifierExtension []Extension                       `bson:"modifierExtension,omitempty" json:"modifierExtension,omitempty"`
	Code              *CodeableConcept                  `bson:"code,omitempty" json:"code,omitempty"`
	Description       *string                           `bson:"description,omitempty" json:"description,omitempty"`
	Criteria          *Expression                       `bson:"criteria,omitempty" json:"criteria,omitempty"`
	Component         []MeasureGroupStratifierComponent `bson:"component,omitempty" json:"component,omitempty"`
}
type MeasureGroupStratifierComponent struct {
	Id                *string          `bson:"id,omitempty" json:"id,omitempty"`
	Extension         []Extension      `bson:"extension,omitempty" json:"extension,omitempty"`
	ModifierExtension []Extension      `bson:"modifierExtension,omitempty" json:"modifierExtension,omitempty"`
	Code              *CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
	Description       *string          `bson:"description,omitempty" json:"description,omitempty"`
	Criteria          Expression       `bson:"criteria" json:"criteria"`
}
type MeasureSupplementalData struct {
	Id                *string           `bson:"id,omitempty" json:"id,omitempty"`
	Extension         []Extension       `bson:"extension,omitempty" json:"extension,omitempty"`
	ModifierExtension []Extension       `bson:"modifierExtension,omitempty" json:"modifierExtension,omitempty"`
	Code              *CodeableConcept  `bson:"code,omitempty" json:"code,omitempty"`
	Usage             []CodeableConcept `bson:"usage,omitempty" json:"usage,omitempty"`
	Description       *string           `bson:"description,omitempty" json:"description,omitempty"`
	Criteria          Expression        `bson:"criteria" json:"criteria"`
}
type OtherMeasure Measure

// MarshalJSON marshals the given Measure as JSON into a byte slice
func (r Measure) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		OtherMeasure
		ResourceType string `json:"resourceType"`
	}{
		OtherMeasure: OtherMeasure(r),
		ResourceType: "Measure",
	})
}

// UnmarshalMeasure unmarshals a Measure.
func UnmarshalMeasure(b []byte) (Measure, error) {
	var measure Measure
	if err := json.Unmarshal(b, &measure); err != nil {
		return measure, err
	}
	return measure, nil
}

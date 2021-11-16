/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package stats

import (
	"reflect"
	"testing"
)

func Test_parseThresholdExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		wantCondition *thresholdExpression
		wantErr       bool
	}{
		{"unknown expression's operator fails", "count!20", nil, true},
		{"unknown expression's method fails", "foo>20", nil, true},
		{"non numerical expression's value fails", "count>abc", nil, true},
		{"valid threshold expression syntax", "count>20", &thresholdExpression{AggregationMethod: "count", Operator: ">", Value: 20}, false},
	}
	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotCondition, err := parseThresholdExpression(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Errorf("parse() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}
			if !reflect.DeepEqual(gotCondition, testCase.wantCondition) {
				t.Errorf("parse() = %v, want %v", gotCondition, testCase.wantCondition)
			}
		})
	}
}

func Test_parseThresholdAggregationMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"count method is parsed", "count", "count", false},
		{"rate method is parsed", "rate", "rate", false},
		{"value method is parsed", "value", "value", false},
		{"avg method is parsed", "avg", "avg", false},
		{"min method is parsed", "min", "min", false},
		{"max method is parsed", "max", "max", false},
		{"med method is parsed", "med", "med", false},
		{"percentile method with integer value is parsed", "p(99)", "p(99)", false},
		{"percentile method with floating point value is parsed", "p(99.9)", "p(99.9)", false},
		{"parsing invalid method fails", "foo", "", true},
		{"parsing incomplete percentile expression fails", "p(99", "", true},
		{"parsing non-numerical percentile value fails", "p(foo)", "", true},
	}
	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseThresholdAggregationMethod(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Errorf("parseMethod() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}
			if got != testCase.want {
				t.Errorf("parseMethod() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_scanThresholdExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantMethod   string
		wantOperator string
		wantValue    string
		wantErr      bool
	}{
		{"expression with <= operator is scanned", "foo<=bar", "foo", "<=", "bar", false},
		{"expression with < operator is scanned", "foo<bar", "foo", "<", "bar", false},
		{"expression with >= operator is scanned", "foo>=bar", "foo", ">=", "bar", false},
		{"expression with > operator is scanned", "foo>bar", "foo", ">", "bar", false},
		{"expression with === operator is scanned", "foo===bar", "foo", "===", "bar", false},
		{"expression with == operator is scanned", "foo==bar", "foo", "==", "bar", false},
		{"expression with != operator is scanned", "foo!=bar", "foo", "!=", "bar", false},
		{"expression's method is trimmed", "  foo  <=bar", "foo", "<=", "bar", false},
		{"expression's value is trimmed", "foo<=  bar  ", "foo", "<=", "bar", false},
		{"expression with unknown operator fails", "foo!bar", "", "", "", true},
	}
	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotMethod, gotOperator, gotValue, err := scanThresholdExpression(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Errorf("scan() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}
			if gotMethod != testCase.wantMethod {
				t.Errorf("scan() gotMethod = %v, want %v", gotMethod, testCase.wantMethod)
			}
			if gotOperator != testCase.wantOperator {
				t.Errorf("scan() gotOperator = %v, want %v", gotOperator, testCase.wantOperator)
			}
			if gotValue != testCase.wantValue {
				t.Errorf("scan() gotValue = %v, want %v", gotValue, testCase.wantValue)
			}
		})
	}
}

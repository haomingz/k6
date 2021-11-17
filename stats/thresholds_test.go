/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2016 Load Impact
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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.k6.io/k6/lib/types"
)

func TestNewThreshold(t *testing.T) {
	t.Parallel()

	src := `rate<0.01`
	abortOnFail := false
	gracePeriod := types.NullDurationFrom(2 * time.Second)
	wantParsed := &thresholdExpression{tokenRate, tokenLess, 0.01}

	gotThreshold, err := newThreshold(src, abortOnFail, gracePeriod)

	assert.NoError(t, err)
	assert.Equal(t, src, gotThreshold.Source)
	assert.False(t, gotThreshold.LastFailed)
	assert.Equal(t, abortOnFail, gotThreshold.AbortOnFail)
	assert.Equal(t, gracePeriod, gotThreshold.AbortGracePeriod)
	assert.Equal(t, wantParsed, gotThreshold.parsed)
}

func TestNewThreshold_InvalidThresholdConditionExpression(t *testing.T) {
	t.Parallel()

	src := "1+1==2"
	abortOnFail := false
	gracePeriod := types.NullDurationFrom(2 * time.Second)

	gotThreshold, err := newThreshold(src, abortOnFail, gracePeriod)

	assert.Error(t, err, "instantiating a threshold with an invalid expression should fail")
	assert.Nil(t, gotThreshold, "instantiating a threshold with an invalid expression should return a nil Threshold")
}

func TestThreshold_runNoTaint(t *testing.T) {
	t.Parallel()

	type fields struct {
		Source           string
		LastFailed       bool
		AbortOnFail      bool
		AbortGracePeriod types.NullDuration
		parsed           *thresholdExpression
	}
	type args struct {
		sinks map[string]float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			"valid expression using the > operator over passing threshold",
			fields{"rate>0.01", false, false, types.NullDurationFrom(2 * time.Second), &thresholdExpression{tokenRate, tokenGreater, 0.01}},
			args{map[string]float64{"rate": 1}},
			true,
			false,
		},
		{
			"valid expression using the >= operator over passing threshold",
			fields{"rate>=0.01", false, false, types.NullDurationFrom(2 * time.Second), &thresholdExpression{tokenRate, tokenGreaterEqual, 0.01}},
			args{map[string]float64{"rate": 0.01}},
			true,
			false,
		},
		{
			"valid expression using the <= operator over passing threshold",
			fields{"rate<=0.01", false, false, types.NullDurationFrom(2 * time.Second), &thresholdExpression{tokenRate, tokenLessEqual, 0.01}},
			args{map[string]float64{"rate": 0.01}},
			true,
			false,
		},
		{
			"valid expression using the < operator over passing threshold",
			fields{"rate<0.01", false, false, types.NullDurationFrom(2 * time.Second), &thresholdExpression{tokenRate, tokenLess, 0.01}},
			args{map[string]float64{"rate": 0.00001}},
			true,
			false,
		},
		{
			"valid expression using the == operator over passing threshold",
			fields{"rate==0.01", false, false, types.NullDurationFrom(2 * time.Second), &thresholdExpression{tokenRate, tokenLooselyEqual, 0.01}},
			args{map[string]float64{"rate": 0.01}},
			true,
			false,
		},
		{
			"valid expression using the === operator over passing threshold",
			fields{"rate===0.01", false, false, types.NullDurationFrom(2 * time.Second), &thresholdExpression{tokenRate, tokenStrictlyEqual, 0.01}},
			args{map[string]float64{"rate": 0.01}},
			true,
			false,
		},
		{
			"valid expression using != operator over passing threshold",
			fields{"rate!=0.01", false, false, types.NullDurationFrom(2 * time.Second), &thresholdExpression{tokenRate, tokenBangEqual, 0.01}},
			args{map[string]float64{"rate": 0.02}},
			true,
			false,
		},
		{
			"valid expression over failing threshold",
			fields{"rate>0.01", false, false, types.NullDurationFrom(2 * time.Second), &thresholdExpression{tokenRate, tokenGreater, 0.01}},
			args{map[string]float64{"rate": 0.00001}},
			false,
			false,
		},
		{
			"valid expression over non-existing sink",
			fields{"rate>0.01", false, false, types.NullDurationFrom(2 * time.Second), &thresholdExpression{tokenRate, tokenGreater, 0.01}},
			args{map[string]float64{"med": 27.2}},
			false,
			true,
		},
		{
			// The ParseThresholdCondition constructor should ensure that no invalid
			// operator gets through, but let's protect our future selves anyhow.
			"invalid expression operator",
			fields{"rate&0.01", false, false, types.NullDurationFrom(2 * time.Second), &thresholdExpression{tokenRate, "&", 0.01}},
			args{map[string]float64{"rate": 0.00001}},
			false,
			true,
		},
	}
	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			threshold := &Threshold{
				Source:           testCase.fields.Source,
				LastFailed:       testCase.fields.LastFailed,
				AbortOnFail:      testCase.fields.AbortOnFail,
				AbortGracePeriod: testCase.fields.AbortGracePeriod,
				parsed:           testCase.fields.parsed,
			}
			got, err := threshold.runNoTaint(testCase.args.sinks)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Threshold.runNoTaint() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}
			if got != testCase.want {
				t.Errorf("Threshold.runNoTaint() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestThresholdRun(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		sinks := map[string]float64{"rate": 0.0001}
		threshold, err := newThreshold(`rate<0.01`, false, types.NullDuration{})
		assert.NoError(t, err)

		t.Run("no taint", func(t *testing.T) {
			b, err := threshold.runNoTaint(sinks)
			assert.NoError(t, err)
			assert.True(t, b)
			assert.False(t, threshold.LastFailed)
		})

		t.Run("taint", func(t *testing.T) {
			t.Parallel()

			b, err := threshold.run(sinks)
			assert.NoError(t, err)
			assert.True(t, b)
			assert.False(t, threshold.LastFailed)
		})
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		sinks := map[string]float64{"rate": 1}
		threshold, err := newThreshold(`rate<0.01`, false, types.NullDuration{})
		assert.NoError(t, err)

		t.Run("no taint", func(t *testing.T) {
			b, err := threshold.runNoTaint(sinks)
			assert.NoError(t, err)
			assert.False(t, b)
			assert.False(t, threshold.LastFailed)
		})

		t.Run("taint", func(t *testing.T) {
			b, err := threshold.run(sinks)
			assert.NoError(t, err)
			assert.False(t, b)
			assert.True(t, threshold.LastFailed)
		})
	})
}

func TestNewThresholds(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		ts, err := NewThresholds([]string{})
		assert.NoError(t, err)
		assert.Len(t, ts.Thresholds, 0)
	})
	t.Run("two", func(t *testing.T) {
		t.Parallel()

		sources := []string{`rate<0.01`, `p(95)<200`}
		ts, err := NewThresholds(sources)
		assert.NoError(t, err)
		assert.Len(t, ts.Thresholds, 2)
		for i, th := range ts.Thresholds {
			assert.Equal(t, sources[i], th.Source)
			assert.False(t, th.LastFailed)
			assert.False(t, th.AbortOnFail)
		}
	})
}

func TestNewThresholdsWithConfig(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		ts, err := NewThresholds([]string{})
		assert.NoError(t, err)
		assert.Len(t, ts.Thresholds, 0)
	})
	t.Run("two", func(t *testing.T) {
		t.Parallel()

		configs := []thresholdConfig{
			{`rate<0.01`, false, types.NullDuration{}},
			{`p(95)<200`, true, types.NullDuration{}},
		}
		ts, err := newThresholdsWithConfig(configs)
		assert.NoError(t, err)
		assert.Len(t, ts.Thresholds, 2)
		for i, th := range ts.Thresholds {
			assert.Equal(t, configs[i].Threshold, th.Source)
			assert.False(t, th.LastFailed)
			assert.Equal(t, configs[i].AbortOnFail, th.AbortOnFail)
		}
	})
}

func TestThresholdsRunAll(t *testing.T) {
	t.Parallel()

	zero := types.NullDuration{}
	oneSec := types.NullDurationFrom(time.Second)
	twoSec := types.NullDurationFrom(2 * time.Second)
	testdata := map[string]struct {
		succeeded bool
		err       bool
		abort     bool
		grace     types.NullDuration
		sources   []string
	}{
		"one passing":                {true, false, false, zero, []string{`rate<0.01`}},
		"one failing":                {false, false, false, zero, []string{`p(95)<200`}},
		"two passing":                {true, false, false, zero, []string{`rate<0.1`, `rate<0.01`}},
		"two failing":                {false, false, false, zero, []string{`p(95)<200`, `rate<0.1`}},
		"two mixed":                  {false, false, false, zero, []string{`rate<0.01`, `p(95)<200`}},
		"one aborting":               {false, false, true, zero, []string{`p(95)<200`}},
		"abort with grace period":    {false, false, true, oneSec, []string{`p(95)<200`}},
		"no abort with grace period": {false, false, true, twoSec, []string{`p(95)<200`}},
	}

	for name, data := range testdata {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			thresholds, err := NewThresholds(data.sources)
			thresholds.sinked = map[string]float64{"rate": 0.0001, "p(95)": 500}
			thresholds.Thresholds[0].AbortOnFail = data.abort
			thresholds.Thresholds[0].AbortGracePeriod = data.grace

			runDuration := 1500 * time.Millisecond

			assert.NoError(t, err)

			succeeded, err := thresholds.runAll(runDuration)

			if data.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if data.succeeded {
				assert.True(t, succeeded)
			} else {
				assert.False(t, succeeded)
			}

			if data.abort && data.grace.Duration < types.Duration(runDuration) {
				assert.True(t, thresholds.Abort)
			} else {
				assert.False(t, thresholds.Abort)
			}
		})
	}
}

func TestThresholds_Run(t *testing.T) {
	t.Parallel()

	type args struct {
		sink     Sink
		duration time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"Running thresholds of existing sink",
			args{DummySink{"p(95)": 1234.5}, 0},
			true,
			false,
		},
		{
			"Running thresholds of existing sink but failing threshold",
			args{DummySink{"p(95)": 3000}, 0},
			false,
			false,
		},
		{
			"Running threshold on non existing sink fails",
			args{DummySink{"dummy": 0}, 0},
			false,
			true,
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			thresholds, err := NewThresholds([]string{"p(95)<2000"})
			assert.NoError(t, err, "Initializing new thresholds should not fail")

			got, err := thresholds.Run(testCase.args.sink, testCase.args.duration)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Thresholds.Run() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}
			if got != testCase.want {
				t.Errorf("Thresholds.Run() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestThresholdsJSON(t *testing.T) {
	t.Parallel()

	testdata := []struct {
		JSON        string
		sources     []string
		abortOnFail bool
		gracePeriod types.NullDuration
		outputJSON  string
	}{
		{
			`[]`,
			[]string{},
			false,
			types.NullDuration{},
			"",
		},
		{
			`["rate<0.01"]`,
			[]string{"rate<0.01"},
			false,
			types.NullDuration{},
			"",
		},
		{
			`["rate<0.01"]`,
			[]string{"rate<0.01"},
			false,
			types.NullDuration{},
			`["rate<0.01"]`,
		},
		{
			`["rate<0.01","p(95)<200"]`,
			[]string{"rate<0.01", "p(95)<200"},
			false,
			types.NullDuration{},
			"",
		},
		{
			`[{"threshold":"rate<0.01"}]`,
			[]string{"rate<0.01"},
			false,
			types.NullDuration{},
			`["rate<0.01"]`,
		},
		{
			`[{"threshold":"rate<0.01","abortOnFail":true,"delayAbortEval":null}]`,
			[]string{"rate<0.01"},
			true,
			types.NullDuration{},
			"",
		},
		{
			`[{"threshold":"rate<0.01","abortOnFail":true,"delayAbortEval":"2s"}]`,
			[]string{"rate<0.01"},
			true,
			types.NullDurationFrom(2 * time.Second),
			"",
		},
		{
			`[{"threshold":"rate<0.01","abortOnFail":false}]`,
			[]string{"rate<0.01"},
			false,
			types.NullDuration{},
			`["rate<0.01"]`,
		},
		{
			`[{"threshold":"rate<0.01"}, "p(95)<200"]`,
			[]string{"rate<0.01", "p(95)<200"},
			false,
			types.NullDuration{},
			`["rate<0.01","p(95)<200"]`,
		},
	}

	for _, data := range testdata {
		data := data

		t.Run(data.JSON, func(t *testing.T) {
			t.Parallel()

			var ts Thresholds
			assert.NoError(t, json.Unmarshal([]byte(data.JSON), &ts))
			assert.Equal(t, len(data.sources), len(ts.Thresholds))
			for i, src := range data.sources {
				assert.Equal(t, src, ts.Thresholds[i].Source)
				assert.Equal(t, data.abortOnFail, ts.Thresholds[i].AbortOnFail)
				assert.Equal(t, data.gracePeriod, ts.Thresholds[i].AbortGracePeriod)
			}

			t.Run("marshal", func(t *testing.T) {
				data2, err := MarshalJSONWithoutHTMLEscape(ts)
				assert.NoError(t, err)
				output := data.JSON
				if data.outputJSON != "" {
					output = data.outputJSON
				}
				assert.Equal(t, output, string(data2))
			})
		})
	}

	t.Run("bad JSON", func(t *testing.T) {
		t.Parallel()

		var ts Thresholds
		assert.Error(t, json.Unmarshal([]byte("42"), &ts))
		assert.Nil(t, ts.Thresholds)
		assert.False(t, ts.Abort)
	})

	t.Run("bad source", func(t *testing.T) {
		t.Parallel()

		var ts Thresholds
		assert.Error(t, json.Unmarshal([]byte(`["="]`), &ts))
		assert.Nil(t, ts.Thresholds)
		assert.False(t, ts.Abort)
	})
}

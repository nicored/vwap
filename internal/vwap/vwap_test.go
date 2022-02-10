package vwap

import (
	"reflect"
	"sync"
	"testing"
)

func TestNewVWap(t *testing.T) {
	type args struct {
		maxPts int
	}
	tests := map[string]struct {
		args args
		want *VWAP
	}{
		"it should successfully set maxPts to defaultMaxDataPoints when 0 is provided": {
			args: args{
				maxPts: 0,
			},
			want: &VWAP{
				mux:     &sync.Mutex{},
				maxPts:  defaultMaxDataPoints,
				dataPts: []dataPoint{},
				sumPQ:   0,
				sumQ:    0,
				vwap:    0,
			},
		},
		"it should successfully set maxPts to defaultMaxDataPoints when -5 is provided": {
			args: args{
				maxPts: 0,
			},
			want: &VWAP{
				mux:     &sync.Mutex{},
				maxPts:  defaultMaxDataPoints,
				dataPts: []dataPoint{},
				sumPQ:   0,
				sumQ:    0,
				vwap:    0,
			},
		},
		"it should successfully set maxPts to 50": {
			args: args{
				maxPts: 50,
			},
			want: &VWAP{
				mux:     &sync.Mutex{},
				maxPts:  50,
				dataPts: []dataPoint{},
				sumPQ:   0,
				sumQ:    0,
				vwap:    0,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := New(tt.args.maxPts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVWap_Push(t *testing.T) {
	type fields struct {
		maxPts  int
		dataPts []dataPoint
		sumPQ   float64
		sumQ    float64
		vwap    float64
	}
	type vwapVars struct {
		sumPQ float64
		sumQ  float64
		vwap  float64
		nPts  int
	}
	type args struct {
		price  float64
		volume float64
	}
	tests := map[string]struct {
		fields     fields
		args       args
		wantErr    bool
		wantErrMsg string
		wantValues vwapVars
	}{
		"it should return division by 0 error": {
			fields: fields{
				maxPts:  0,
				dataPts: []dataPoint{},
			},
			args:       args{5, 0},
			wantErr:    true,
			wantErrMsg: divBy0Err.Error(),
			wantValues: vwapVars{
				sumPQ: 0,
				sumQ:  0,
				vwap:  0,
				nPts:  0,
			},
		},
		"it should successfully add the first data point": {
			fields: fields{
				maxPts:  0,
				dataPts: []dataPoint{},
			},
			args:       args{5, 2},
			wantErr:    false,
			wantErrMsg: divBy0Err.Error(),
			wantValues: vwapVars{
				sumPQ: 10,
				sumQ:  2,
				vwap:  5, // 5*2 / 2
				nPts:  1,
			},
		},
		"it should successfully recompute the vwaps when pushin a new data point": {
			fields: fields{
				maxPts: 0,
				dataPts: []dataPoint{
					{5, 2},
				},
				sumPQ: 10,
				sumQ:  2,
				vwap:  5,
			},
			args:       args{4, 5},
			wantErr:    false,
			wantErrMsg: "",
			wantValues: vwapVars{
				sumPQ: 30,
				sumQ:  7,
				vwap:  30. / 7., // ((5*2) + (4*5)) / (2 + 5) = 30 / 7
				nPts:  2,
			},
		},
		"it should successfully recompute without the 1st data point when reaching maxPts": {
			fields: fields{
				maxPts: 2,
				dataPts: []dataPoint{
					{5, 2},
					{4, 5},
				},
				sumPQ: 30,
				sumQ:  7,
				vwap:  30. / 7.,
			},
			args:       args{3, 1},
			wantErr:    false,
			wantErrMsg: "",
			wantValues: vwapVars{
				sumPQ: 23,
				sumQ:  6,
				vwap:  23. / 6., // (30 - (5*2) + (3 * 1)) / (7 - 2 + 1) = 23 / 6
				nPts:  2,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			v := New(tt.fields.maxPts)
			v.dataPts = tt.fields.dataPts
			v.sumPQ = tt.fields.sumPQ
			v.sumQ = tt.fields.sumQ
			v.vwap = tt.fields.vwap

			if err := v.Push(tt.args.price, tt.args.volume); (err != nil) != tt.wantErr {
				t.Errorf("Push() error = %v, wantErr %v", err, tt.wantErr)

				if tt.wantErrMsg != err.Error() {
					t.Errorf("Push() errorMsg = %s, wantErrMsg = %s", err, tt.wantErrMsg)
				}
			}

			got := vwapVars{
				vwap:  v.Value(),
				sumPQ: v.sumPQ,
				sumQ:  v.sumQ,
				nPts:  v.NPoints(),
			}
			if !reflect.DeepEqual(got, tt.wantValues) {
				t.Errorf("Push() = %v, want %v", got, tt.wantValues)
			}
		})
	}
}

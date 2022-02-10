package vwap

import (
	"fmt"
	"sync"
)

const (
	defaultMaxDataPoints = 200
)

var (
	divBy0Err = fmt.Errorf("error calculating vwap: sum of volumes equals to 0")
)

// VWAP is used to compute the VWAP value from a list of data points
type VWAP struct {
	mux    *sync.Mutex
	maxPts int

	dataPts []dataPoint
	sumPQ   float64
	sumQ    float64
	vwap    float64
}

// New creates a new VWAP used to compute the VWAP from a list of data points
func New(maxPts int) *VWAP {
	if maxPts < 1 {
		maxPts = defaultMaxDataPoints
	}

	return &VWAP{
		mux:     &sync.Mutex{},
		maxPts:  maxPts,
		dataPts: []dataPoint{},
		sumPQ:   0,
		sumQ:    0,
		vwap:    0,
	}
}

// Value returns the value of the pre-computed VWAP
func (v *VWAP) Value() float64 {
	return v.vwap
}

// NPoints returns the number of data points currently held by VWAP
func (v *VWAP) NPoints() int {
	return len(v.dataPts)
}

// Push uses the provided price and volume to recompute the VWAP
// When the data points list reaches maxPts, the oldest data point falls off
// and the new one is added and used in the calculation
func (v *VWAP) Push(price float64, volume float64) error {
	v.mux.Lock()
	defer v.mux.Unlock()

	sumPQ := v.sumPQ + (price * volume)
	sumQ := v.sumQ + volume

	// when reaching the max number of processable data points, we recompute
	// the sums of PQ and Q without the first data point
	if len(v.dataPts) == v.maxPts {
		sumPQ = sumPQ - (v.dataPts[0].price * v.dataPts[0].volume)
		sumQ = sumQ - (v.dataPts[0].volume)
	}

	// sumPQ should never be 0, but safeguarding just in case
	if sumPQ == 0 {
		return divBy0Err
	}

	// it is now safe to remove the first data point from the list
	if len(v.dataPts) == v.maxPts {
		v.dataPts = v.dataPts[1:]
	}

	// and also safe to add the new data point to the list and store the sums
	// and calculate the vwap
	v.dataPts = append(v.dataPts, newDataPoint(price, volume))
	v.sumPQ = sumPQ
	v.sumQ = sumQ
	v.vwap = sumPQ / sumQ

	return nil
}

// dataPoint represents a single element of data points used by VWAP
// to compute the final value
type dataPoint struct {
	price  float64
	volume float64
}

func newDataPoint(price float64, volume float64) dataPoint {
	return dataPoint{
		price:  price,
		volume: volume,
	}
}

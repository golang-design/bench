package stat

import "errors"

// A DeltaTest compares the old and new metrics and returns the
// expected probability that they are drawn from the same distribution.
//
// If a probability cannot be computed, the DeltaTest returns an
// error explaining why. Common errors include ErrSamplesEqual
// (all samples are equal), ErrSampleSize (there aren't enough samples),
// and ErrZeroVariance (the sample has zero variance).
//
// As a special case, the missing test NoDeltaTest returns -1, nil.
type DeltaTest func(old, new *Metrics) (float64, error)

// Errors returned by DeltaTest.
var (
	ErrSamplesEqual      = errors.New("all equal")
	ErrSampleSize        = errors.New("too few samples")
	ErrZeroVariance      = errors.New("zero variance")
	ErrMismatchedSamples = errors.New("samples have different lengths")
)

// NoDeltaTest applies no delta test; it returns -1, nil.
func NoDeltaTest(old, new *Metrics) (pval float64, err error) {
	return -1, nil
}

// TTest is a DeltaTest using the two-sample Welch t-test.
func TTest(old, new *Metrics) (pval float64, err error) {
	t, err := TwoSampleWelchTTest(
		Sample{Xs: old.RValues},
		Sample{Xs: new.RValues},
		LocationDiffers,
	)
	if err != nil {
		return -1, err
	}
	return t.P, nil
}

// UTest is a DeltaTest using the Mann-Whitney U test.
func UTest(old, new *Metrics) (pval float64, err error) {
	u, err := MannWhitneyUTest(old.RValues, new.RValues, LocationDiffers)
	if err != nil {
		return -1, err
	}
	return u.P, nil
}

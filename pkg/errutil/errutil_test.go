package errutil_test

import (
	"errors"
	"testing"

	"github.com/ross-weir/rosetta-ergo/pkg/errutil"
	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	for i := 0; i < len(errutil.Errors); i++ {
		assert.Equal(t, int32(i), errutil.Errors[i].Code)
	}
}

func TestWrapErr(t *testing.T) {
	err := errors.New("testing")
	typedErr := errutil.WrapErr(errutil.ErrUnclearIntent, err)

	assert.Equal(t, errutil.ErrUnclearIntent.Code, typedErr.Code)
	assert.Equal(t, errutil.ErrUnclearIntent.Message, typedErr.Message)
	assert.Equal(t, err.Error(), typedErr.Details["context"])

	// Assert we don't overwrite our reference.
	assert.Nil(t, errutil.ErrUnclearIntent.Details)
}

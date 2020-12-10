package utils

import (
	"fmt"
	"reflect"
)

func CheckSpanNames(gotSpanNames, wantSpanNames []string) error {
	if !reflect.DeepEqual(gotSpanNames, wantSpanNames) {
		return fmt.Errorf("got span names: %+q, want span names: %+q", gotSpanNames, wantSpanNames)
	}

	return nil
}

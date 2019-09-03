package flags

import (
	"reflect"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
)

func TestDefaultEnvoyConfigOptions(t *testing.T) {
	defaultOptions := configinfo.DefaultEnvoyConfigOptions()
	actualOptions := EnvoyConfigOptionsFromFlags()

	if !reflect.DeepEqual(defaultOptions, actualOptions) {
		t.Fatalf("DefaultEnvoyConfigOptions does not match envoyConfigOptionsFromFlags:\nhave: %v\nwant: %v",
			defaultOptions, actualOptions)
	}
}

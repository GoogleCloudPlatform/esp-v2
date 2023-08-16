package routegen_test

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/routegen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/routegen/routegentest"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/imdario/mergo"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestNewCORSRouteGensFromOPConfig(t *testing.T) {
	testdata := []routegentest.SuccessOPTestCase{
		{
			Desc: "cors default allow_origin=* routes",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				CorsPreset:       "basic",
				CorsAllowOrigin:  "*",
				CorsAllowMethods: "GET,POST,PUT,OPTIONS",
				CorsMaxAge:       2 * time.Minute,
			},
			WantHostConfig: `
{
	"routes": [
		{
			"decorator": {
				"operation": "ingress"
			},
			"match": {
				"headers": [
					{
						"name": ":method",
						"stringMatch": {
							"exact": "OPTIONS"
						}
					},
					{
						"name": "origin",
						"presentMatch": true
					},
					{
						"name": "access-control-request-method",
						"presentMatch": true
					}
				],
				"prefix": "/"
			},
			"route": {
				"cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local"
			}
		},
		{
			"decorator": {
				"operation": "ingress"
			},
			"directResponse": {
				"body": {
					"inlineString": "The CORS preflight request is missing one (or more) of the following required headers [Origin, Access-Control-Request-Method] or has an unmatched Origin header."
				},
				"status": 400
			},
			"match": {
				"headers": [
					{
						"name": ":method",
						"stringMatch": {
							"exact": "OPTIONS"
						}
					}
				],
				"prefix": "/"
			}
		}
	]
}
			`,
		},
		{
			Desc: "cors exact origin routes",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				CorsPreset:       "basic",
				CorsAllowOrigin:  "http://example.com",
				CorsAllowMethods: "GET,POST,PUT,OPTIONS",
				CorsMaxAge:       2 * time.Minute,
			},
			WantHostConfig: `
{
	"routes": [
		{
			"decorator": {
				"operation": "ingress"
			},
			"match": {
				"headers": [
					{
						"name": ":method",
						"stringMatch": {
							"exact": "OPTIONS"
						}
					},
					{
						"name": "origin",
						"stringMatch": {
							"exact": "http://example.com"
						}
					},
					{
						"name": "access-control-request-method",
						"presentMatch": true
					}
				],
				"prefix": "/"
			},
			"route": {
				"cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local"
			}
		},
		{
			"decorator": {
				"operation": "ingress"
			},
			"directResponse": {
				"body": {
					"inlineString": "The CORS preflight request is missing one (or more) of the following required headers [Origin, Access-Control-Request-Method] or has an unmatched Origin header."
				},
				"status": 400
			},
			"match": {
				"headers": [
					{
						"name": ":method",
						"stringMatch": {
							"exact": "OPTIONS"
						}
					}
				],
				"prefix": "/"
			}
		}
	]
}
			`,
		},
		{
			Desc: "cors regex origin routes",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				CorsPreset:           "cors_with_regex",
				CorsAllowOriginRegex: ".*",
				CorsAllowMethods:     "GET,POST,PUT,OPTIONS",
				CorsMaxAge:           2 * time.Minute,
			},
			WantHostConfig: `
{
	"routes": [
		{
			"decorator": {
				"operation": "ingress"
			},
			"match": {
				"headers": [
					{
						"name": ":method",
						"stringMatch": {
							"exact": "OPTIONS"
						}
					},
					{
						"name": "origin",
						"stringMatch": {
							"safeRegex": {
								"regex": ".*"
							}
						}
					},
					{
						"name": "access-control-request-method",
						"presentMatch": true
					}
				],
				"prefix": "/"
			},
			"route": {
				"cluster": "backend-cluster-bookstore.endpoints.project123.cloud.goog_local"
			}
		},
		{
			"decorator": {
				"operation": "ingress"
			},
			"directResponse": {
				"body": {
					"inlineString": "The CORS preflight request is missing one (or more) of the following required headers [Origin, Access-Control-Request-Method] or has an unmatched Origin header."
				},
				"status": 400
			},
			"match": {
				"headers": [
					{
						"name": ":method",
						"stringMatch": {
							"exact": "OPTIONS"
						}
					}
				],
				"prefix": "/"
			}
		}
	]
}
			`,
		},
		{
			Desc: "cors is disabled",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn:         options.ConfigGeneratorOptions{},
			WantHostConfig: `{}`,
		},
		{
			Desc: "cors is disabled, even when other options are set",
			ServiceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			OptsIn: options.ConfigGeneratorOptions{
				CorsAllowMethods: "GET",
				CorsMaxAge:       2 * time.Minute,
			},
			WantHostConfig: `{}`,
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, routegen.NewCORSRouteGensFromOPConfig)
	}
}

func TestNewCORSRouteGensFromOPConfig_BadInputRouteGen(t *testing.T) {
	testdata := []routegentest.GenConfigErrorOPTestCase{
		{
			Desc:            "Incorrect preset",
			ServiceConfigIn: &servicepb.Service{},
			OptsIn: options.ConfigGeneratorOptions{
				CorsPreset: "foo_bar",
			},
			OptsMergeBehavior: mergo.WithOverwriteWithEmptyValue,
			WantGenErrors: []string{
				`cors_preset must be either "basic" or "cors_with_regex"`,
			},
		},
		{
			Desc:            "Incorrect configured basic Cors",
			ServiceConfigIn: &servicepb.Service{},
			OptsIn: options.ConfigGeneratorOptions{
				CorsPreset: "basic",
				// Missing origin, but origin regex configured
				CorsAllowOriginRegex: "^https?://.+\\\\.example\\\\.com\\/?$",
				CorsMaxAge:           2 * time.Minute,
			},
			OptsMergeBehavior: mergo.WithOverwriteWithEmptyValue,
			WantGenErrors: []string{
				"cors_allow_origin cannot be empty when cors_preset=basic",
			},
		},
		{
			Desc:            "Incorrect configured regex Cors",
			ServiceConfigIn: &servicepb.Service{},
			OptsIn: options.ConfigGeneratorOptions{
				CorsPreset:           "cors_with_regex",
				CorsAllowOriginRegex: `^https?://.+\\.example\\.com\/$$$_$(*##*(@!)((!_!(@$`,
				CorsMaxAge:           2 * time.Minute,
			},
			OptsMergeBehavior: mergo.WithOverwriteWithEmptyValue,
			WantGenErrors: []string{
				`error parsing regexp`,
			},
		},
		{
			Desc:            "Oversize cors origin regex",
			ServiceConfigIn: &servicepb.Service{},
			OptsIn: options.ConfigGeneratorOptions{
				CorsPreset:           "cors_with_regex",
				CorsAllowOriginRegex: makeOverSizeRegexForTest(),
				CorsAllowHeaders:     "Origin,Content-Type,Accept",
				CorsMaxAge:           2 * time.Minute,
			},
			OptsMergeBehavior: mergo.WithOverwriteWithEmptyValue,
			WantGenErrors: []string{
				`invalid cors origin regex: regex program size`,
			},
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, routegen.NewCORSRouteGensFromOPConfig)
	}
}

// makeOverSizeRegexForTest generates an oversize cors origin regex
// or a oversize uri template.
func makeOverSizeRegexForTest() string {
	overSizeRegex := ""
	for i := 0; i < 333; i += 1 {
		// Form regex in a way that it cannot be simplified.
		overSizeRegex += "[abc]+123"
	}
	return overSizeRegex
}

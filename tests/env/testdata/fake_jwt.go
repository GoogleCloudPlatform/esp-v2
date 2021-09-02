// Copyright 2019 Google LLC
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

package testdata

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

type ProviderConfig struct {
	Id                    string
	Issuer                string
	Keys                  string
	IsInvalid             bool // If invalid, then a server that always returns 503 is started up
	IsNonexistent         bool // If non-existent, then a server is not started up
	IsPeriodicallyFailing bool // If periodically failing, server succeeds 1/SuccessPeriod times, returns 503 otherwise
	SuccessPeriod         int
	HardcodedJwksUri      string // This needs to be set for non-existent, since the URL cannot be derived
	JwtLocations          []*confpb.JwtLocation
}

var (
	// Configuration for non-OpenID providers.
	ProviderConfigs = []*ProviderConfig{
		{
			Id:        BrokenProvider,
			Issuer:    BrokenIssuer,
			IsInvalid: true,
		},
		{
			Id:     GoogleServiceAccountProvider,
			Issuer: ApiProxyTestingIssuer,
			Keys:   FakeCloudJwks,
		},
		{
			Id:     GoogleJwtProvider,
			Issuer: ApiProxyTestingIssuer,
			Keys:   FakeCloudJwks,
		},
		{

			Id:     EndpointsJwtProvider,
			Issuer: JwtEndpointsIssuer,
			Keys:   FakeEndpointsJwks,
		},
		{
			Id:     TestAuthProvider,
			Issuer: Es256Issuer,
			Keys:   PubKeys,
		},
		{
			Id:     TestAuth1Provider,
			Issuer: Rs256Issuer,
			Keys:   PubKeys,
		},
		{
			Id:     InvalidProvider,
			Issuer: InvalidIssuer,
			Keys:   "invalid-jwks",
		},
		{
			Id:               NonexistentProvider,
			Issuer:           NonexistentIssuer,
			IsNonexistent:    true,
			HardcodedJwksUri: fmt.Sprintf("http://%v:55550/pkey", platform.GetLoopbackAddress()),
		},
		{
			Id:     ServiceControlProvider,
			Issuer: Es256Issuer,
			Keys:   ServiceControlJwtPayloadPubKeys,
		},
		{
			Id:     X509Provider,
			Issuer: FakeIssuer,
			Keys:   x509PubKeys,
		},
		{
			Id:     CustomJwtLocationProvider,
			Issuer: Rs256Issuer,
			Keys:   PubKeys,
			JwtLocations: []*confpb.JwtLocation{
				{
					In: &confpb.JwtLocation_Header{
						Header: "jwt-header-foo",
					},
					ValuePrefix: "jwt-prefix-foo",
				},
				{
					In: &confpb.JwtLocation_Header{
						Header: "jwt-header-bar",
					},
					ValuePrefix: "jwt-prefix-bar",
				},
				{
					In: &confpb.JwtLocation_Query{Query: "jwt-param-foo"},
				},
				{
					In: &confpb.JwtLocation_Query{Query: "jwt-param-bar"},
				},
			},
		},
	}
)

// Providers
const (
	OpenIdProvider               string = "openID_provider"
	OpenIdInvalidProvider        string = "openID_invalid_provider"
	OpenIdNonexistentProvider    string = "openID_nonexist_provider"
	GoogleServiceAccountProvider string = "google_service_account"
	GoogleJwtProvider            string = "google_jwt"
	EndpointsJwtProvider         string = "endpoints_jwt"
	BrokenProvider               string = "broken_provider"
	TestAuthProvider             string = "test_auth"
	TestAuth1Provider            string = "test_auth_1"
	InvalidProvider              string = "invalid_jwks_provider"
	NonexistentProvider          string = "nonexist_jwks_provider"
	ServiceControlProvider       string = "service_control_jwt_payload_auth"
	X509Provider                 string = "x509_jwt_provider"
	CustomJwtLocationProvider    string = "custom_jwt_location_provider"
)

// Issuers
const (
	ApiProxyTestingIssuer string = "api-proxy-testing@cloud.goog"
	JwtEndpointsIssuer    string = "jwt-client.endpoints.sample.google.com"
	BrokenIssuer          string = "http://broken_issuer.com"
	Es256Issuer           string = "es256-issuer"
	Rs256Issuer           string = "rs256-issuer"
	InvalidIssuer         string = "invalid_jwks_provider"
	NonexistentIssuer     string = "nonexist_jwks_provider"
	FakeIssuer            string = "fake.issuer"
)

// Keys and tokens
const (
	ServiceControlJwtPayloadPubKeys = `{
	 "keys": [
		{
			"e":"AQAB",
			"kid":"DHFbpoIUqrY8t2zpA2qXfCmr5VO5ZEr4RzHU_-envvQ",
			"kty":"RSA",
			"n":"xAE7eB6qugXyCAG3yhh7pkDkT65pHymX-P7KfIupjf59vsdo91bSP9C8H07pSAGQO1MV_xFj9VswgsCg4R6otmg5PV2He95lZdHtOcU5DXIg_pbhLdKXbi66GlVeK6ABZOUW3WYtnNHD-91gVuoeJT_DwtGGcp4ignkgXfkiEm4sw-4sfb4qdt5oLbyVpmW6x9cfa7vs2WTfURiCrBoUqgBo_-4WTiULmmHSGZHOjzwa8WtrtOQGsAFjIbno85jp6MnGGGZPYZbDAa_b3y5u-YpW7ypZrvD8BgtKVjgtQgZhLAGezMt0ua3DRrWnKqTZ0BJ_EyxOGuHJrLsn00fnMQ"
		}
	 ]
	}`

	PubKeys = `{
		"keys": [
		{
			"kty": "EC",
			"crv": "P-256",
			"x": "lqldKduURoauGtQskOXRTTociai06C-Ug_lwDqcXdd4",
			"y": "t3FPM5-BhLsjyTG6QcDkTotU6PTMmrT6KCfr4L_0Lhk",
			"alg": "ES256",
			"kid": "1a"
		},
		{
			"kty": "RSA",
			"n": "zaS0LKbCovc6gdmwwEbovLBqEuat2ihKmuXMEAh7yjk--Pw55djgkpiAFaoTr0-iEnJB8QKQAkssU5mQcKHCtKRfVH9TZv3JC8mXeSg1dvS-AckkGqXwuPpYyaTUDZsd7u3xW3lSX4QtrLNcwCo0TRFmUGcpkecy6omJdD8kwhWXYOEkDPZqZXlvWkLfyuelWE8Wcrv-X_v8UrCMOOECRPRxl5tmC93vMnZZAHN35gyLizaPOkXPR69DN-_d34aiLctphiqzTJUlMlpIU2SciXj2CaOMFzioy-cRb9sbr8eN91cDPDs4r-EiFB6bcoAJxaHCyxdhJYihFGfwGjhCkQ",
			"e": "AQAB",
			"alg": "RS256",
			"kid": "2b"
		}
		]
	}`

	// Copied from: https://github.com/google/jwt_verify_lib/pull/33/files
	x509PubKeys = `{
  "82cfd797903063a0b78ce1cbf5e2fe036a6de242": "-----BEGIN CERTIFICATE-----\nMIIC+jCCAeKgAwIBAgIIEN2Xgd3Y1CMwDQYJKoZIhvcNAQEFBQAwIDEeMBwGA1UE\nAxMVMTA2OTQ3MDEyMjYwNDg4NzM2MTU3MB4XDTE5MDIyNzE3NTA1N1oXDTI5MDIy\nNDE3NTA1N1owIDEeMBwGA1UEAxMVMTA2OTQ3MDEyMjYwNDg4NzM2MTU3MIIBIjAN\nBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA00bLFfPv/jeyVU6xuStcwHdSBa+m\nlOX/9oWFwMsQucENe+QYKJmkAqdATz3BKJ354iknMy556Y8cBHbZa9X6gxi2BIPW\nzkuKTruDJrQrg6cgR6RHZ9WNoxGLRtyhq8PimV8DVtMSLYVy3p/gMwEtuQY4jiXS\nhhvCZxuJZIJnabNqTU5AGWfduQgDcLRd25cShKxDNOtfcBWQ+ZQWt5qkZGz5XFQ/\nt1+bND+hA3dC3bwLc9yFrgU+Z+XEDQErq4OG9MVezw6h6Imn6gkrdSyG1k9BjPsf\n4senqDXgtK2Iz9MuGIWcG62wV2a7qJYjnGBJfI4QKQBEdsYbuUel2wB0wQIDAQAB\nozgwNjAMBgNVHRMBAf8EAjAAMA4GA1UdDwEB/wQEAwIHgDAWBgNVHSUBAf8EDDAK\nBggrBgEFBQcDAjANBgkqhkiG9w0BAQUFAAOCAQEArrvMP0yrPQlCC/QB0iPxb4TY\nPPiDTuY4fPytUQgvSdQ4rMPSNZafe7tIS+0KDhZtblepaS5whVobVh9lS2bK+rDH\nRsM/H9XRGpyh2rJ6NYUbiyEMQ4jfNh99A02Nsz4Gaed3IE8Hml2pWLcCbp2VGDEN\nr6qrBVVWsaT736/kwVNp14S6FNhVIx1pZeKJrtOsJD+Y4f21WKlWdKdu4QVlxJoE\n9LtFur56aLhDA64D5GPjQnatRyShcWXvgEvUk5YUuBkjTDL1HSNTeqTdG6j8OEZo\nBuyfyPz4yV6BjnJWl2fk8v+9sB1B6m5LoR7ETHlWwh+elmaejFQCJN1+ED8k0w==\n-----END CERTIFICATE-----\n"
	}`

	// Generated with payloads:
	//	{
	//	"aud": "ok_audience_1",
	//	"exp": 4703162488,
	//	"foo": {
	//	"foo_list": [
	//	true,
	//	false
	//	],
	//	"foo_bool": true
	//	},
	//	"google": {
	//	"compute_engine": {
	//	"project_id": "cloudendpoint_testing",
	//	"zone": "us_west1_a",
	//	}
	//	"project_number": 12345,
	//	"google_bool": false
	//	},
	//	"iat": 1549412881,
	//	"iss": "es256-issuer",
	//	"sub": "es256-issuer"
	//	}
	ServiceControlJwtPayloadToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcX" +
		"JZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJhd" +
		"WQiOiJva19hdWRpZW5jZV8xIiwiZXhwIjo0NzAzMTYyNDg4LCJmb28iOnsiZm9vX2Jvb2w" +
		"iOnRydWUsImZvb19saXN0IjpbdHJ1ZSxmYWxzZV19LCJnb29nbGUiOnsiY29tcHV0ZV9lb" +
		"mdpbmUiOnsicHJvamVjdF9pZCI6ImNsb3VkZW5kcG9pbnRfdGVzdGluZyIsInpvbmUiOiJ" +
		"1c193ZXN0MV9hIn0sImdvb2dsZV9ib29sIjpmYWxzZSwicHJvamVjdF9udW1iZXIiOjEyM" +
		"zQ1fSwiaWF0IjoxNTQ5NTYyNDg4LCJpc3MiOiJlczI1Ni1pc3N1ZXIiLCJzdWIiOiJlczI" +
		"1Ni1pc3N1ZXIifQ.SnQ66iwlS80VFvtL-8jeEyqtaxaqW0CgN0W4DoJ5imwatHm1If_ty7" +
		"EbjZUf-ilUawxD_G-xV6_YJ59JX-C6X3SD_yYYrhJZac1V99awCxG3LxTpziiOLzTOY28-" +
		"xayHNwKLQT_qwM3RoJ4eFO1jOzcwxZdvGiyBBuoaht0cygqqFecfxjaBHtGwfyxQcR__FN" +
		"FxZ2JGwL9PK4ytttFFOey1FOIyDM3kd3O2NwMAb8zfI2vPwKizEEYnWqgsfNkzckp02W4s" +
		"01IgOPc5s2XMUjnWoSk_is1Hc527jvIOQhnSDZyHqt9QfsDKdNvZ0qj7E_3p2rbaaTiIno" +
		"gDsvj0aA"

	Es256Token = "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjFhIn0.eyJpc3MiO" +
		"iJlczI1Ni1pc3N1ZXIiLCJzdWIiOiJlczI1Ni1pc3N1ZXIiLCJhdWQiOiJva19hdWRpZW5" +
		"jZSJ9.hz9IUedX6WTbuxQSbcXBSKfvF2hK48o06CnxJn-5vyOkWfUNroJjb3JokQpweF9X" +
		"FI8RxeMGPKFMdHb8qyIlqA"

	Es256TokenPayloadBase64 = "eyJpc3MiOiJlczI1Ni1pc3N1ZXIiLCJzdWIiOiJlczI1Ni1p" +
		"c3N1ZXIiLCJhdWQiOiJva19hdWRpZW5jZSJ9"

	Rs256Token = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjJiIn0.eyJpc3MiO" +
		"iJyczI1Ni1pc3N1ZXIiLCJzdWIiOiJyczI1Ni1pc3N1ZXIiLCJhdWQiOiJva19hdWRpZW5" +
		"jZSJ9.Idf-XyipQCoMmIkI8TT3LgHUseV5AG-tJGhGrEldto-q44oNz9ZEd3KoJ3TlZGKk" +
		"nfEfaSCndsFR_yeHrI1CLdQ7kIs2SaRQP2aG4QqJAwn0-kFoTSUwxqQtV428AKrMrTeahu" +
		"6ZGOGqwaLMOKP2F7pzI2sCFAYMwLCLhbHzvzRwhIPekG8iENj5YDS5_C5GtFtUV4iL7e6K" +
		"S7ZqRTljqZB6HUjG7TL_QMZuQ7S44bLGePgx8AeMlEqBzFizG7cJKvGJjsSTiuxvESBnPN" +
		"pjm4bNFLgLXULoRsoXgU3i1DKQ0r12uztARJpq79diXf-ln7tV-TCwOXlubbb2hiP6-A"

	// All the JWT and JWKS data are generated via the repo below.
	// The second entry corresponds to the private key in the following directory.
	// https://github.com/istio/istio/tree/master/security/tools/jwt/samples
	FakeCloudJwks = `{
		"keys":[
		{
			"e":"AQAB",
			"kid":"j5noYIxnGRW4OBiuxmt6kl-zeQgxcVwfKslNiNZ7J5I",
			"kty":"RSA",
			"n":"nmFbmjsJxUw-JngfaKcFe_47ZR0Nn2FyBxftXID2bhVIfRGZTs0b0C6-IbiAJ7EGGsxMMyxqeA_kQfJi1UQ11SGXANav5y2Lk0EFla7bZCDDFo46jeQh4Ed9I7uNUUmVByz2jsVsiIEHy45wh2U_O_K_KYe_BQ0-JRi_Sh71HUOt9Kw15vbwddWSkDTbLKGm3os1Qo_t0GTT84ow2XIz_8C2zU-1eXlkFSUhudfNf6Encvu0bqJI4hVuOuYocwCNnFtuV1VqJxoNrbrEKPCx6F3Fv82cw_vKh2dGsXeuP8wclbCJHalgIYyYIE0NLFZheHZmCjtQ4zk1VAFikUQ9Fw"
		},
		{ 
            "e":"AQAB",
            "kid":"DHFbpoIUqrY8t2zpA2qXfCmr5VO5ZEr4RzHU_-envvQ",
            "kty":"RSA",
            "n":"xAE7eB6qugXyCAG3yhh7pkDkT65pHymX-P7KfIupjf59vsdo91bSP9C8H07pSAGQO1MV_xFj9VswgsCg4R6otmg5PV2He95lZdHtOcU5DXIg_pbhLdKXbi66GlVeK6ABZOUW3WYtnNHD-91gVuoeJT_DwtGGcp4ignkgXfkiEm4sw-4sfb4qdt5oLbyVpmW6x9cfa7vs2WTfURiCrBoUqgBo_-4WTiULmmHSGZHOjzwa8WtrtOQGsAFjIbno85jp6MnGGGZPYZbDAa_b3y5u-YpW7ypZrvD8BgtKVjgtQgZhLAGezMt0ua3DRrWnKqTZ0BJ_EyxOGuHJrLsn00fnMQ"
        }
		]
	}`

	// Generated by gen-jwt.py key.pem -jwks=./jwks.json --expire=3153600000
	// --iss="api-proxy-testing@cloud.goog"  > demo.jwt
	FakeCloudToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6Imo1bm9ZSXhuR1JXNE9CaXV4bXQ2a" +
		"2wtemVRZ3hjVndmS3NsTmlOWjdKNUkiLCJ0eXAiOiJKV1QifQ.eyJleHAiOjQ2OTgzMTgz" +
		"NTYsImlhdCI6MTU0NDcxODM1NiwiaXNzIjoiYXBpLXByb3h5LXRlc3RpbmdAY2xvdWQuZ2" +
		"9vZyIsInN1YiI6ImFwaS1wcm94eS10ZXN0aW5nQGNsb3VkLmdvb2cifQ.njSzHX0ug5qA9" +
		"NcAX6gmPuQmhjG0ORtAmWLpFNTp_HbEmaOfuUXWxR7OefejdU1nvik8Vb2NNmcAbM9Sgpx" +
		"Ti73lx06QBWeHRntdZOid6u527EY8y-FCnVoDnFCLNxB-dZGNphcWWsPlQvYx4QZT2WQYs" +
		"YyAVEZkN0jqPd4_aqf4nGyyCiCYmmLVxBMT6g4JulqNc0XhS5a0SskB9SWJwUWALWimqUe" +
		"E1VICEqkuY6STxB04BRd5guNq3wrJipCAg2uqkS_YklKDa9E94x0a06ARMMeESXV0sdk-5" +
		"IfxVvICeFNzbXPox2-HAMLTMiSUcaa9y1ss7-Yx6Cfybka_Jg"

	// gen-jwt.py key.pem -jwks=./jwks.json --expire=3153600000
	// --iss="api-proxy-testing@cloud.goog"
	// --aud bookstore_test_client.cloud.goog > demo.jwt
	FakeCloudTokenSingleAudience1 = "eyJhbGciOiJSUzI1NiIsImtpZCI6Imo1bm9ZSXhuR1JXNE9" +
		"CaXV4bXQ2a2wtemVRZ3hjVndmS3NsTmlOWjdKNUkiLCJ0eXAiOiJKV1QifQ.eyJhdWQi" +
		"OiJib29rc3RvcmVfdGVzdF9jbGllbnQuY2xvdWQuZ29vZyIsImV4cCI6NDY5ODMxODgx" +
		"MSwiaWF0IjoxNTQ0NzE4ODExLCJpc3MiOiJhcGktcHJveHktdGVzdGluZ0BjbG91ZC5n" +
		"b29nIiwic3ViIjoiYXBpLXByb3h5LXRlc3RpbmdAY2xvdWQuZ29vZyJ9.XgEmFFIDerz" +
		"RNs18U8QaTW4NdBl2UONNaoCN_wD_pH2UPZffnlDbmDFkA032fewGv_i6gxp2rmGV-sd" +
		"merdu8WoGXJYAtPWKwgefrGbCCEwKiH4qqt8_ZRftujC4aZML5pQ-8dUIACC9n6CpzOp" +
		"nsExPLdJg1AwjjsTnw4W88EvbmOqtK1ryQbagoD1JJOyiO7RYDR4QjKMyZm5lHvwmpuL" +
		"Yd3DZjKleR7khWBLvbU36oJYajNTaxoRX1MWptYHogTP5s5Qr7DozXi4uIpRCyoZkGRE" +
		"QTwbP7tc-BjxR65bm85UgOvSblFKW8470Th-_xiNgDrDu4zYGbgL7jxvnVA"

	// gen-jwt.py key.pem -jwks=./jwks.json --expire=3153600000
	// --iss="api-proxy-testing@cloud.goog"  --aud admin.cloud.goog > demo.jwt
	FakeCloudTokenSingleAudience2 = "eyJhbGciOiJSUzI1NiIsImtpZCI6Imo1bm9ZSXhuR1JXNE9C" +
		"aXV4bXQ2a2wtemVRZ3hjVndmS3NsTmlOWjdKNUkiLCJ0eXAiOiJKV1QifQ.eyJhdWQiO" +
		"iJhZG1pbi5jbG91ZC5nb29nIiwiZXhwIjo0Njk4MzE4OTk1LCJpYXQiOjE1NDQ3MTg5O" +
		"TUsImlzcyI6ImFwaS1wcm94eS10ZXN0aW5nQGNsb3VkLmdvb2ciLCJzdWIiOiJhcGktc" +
		"HJveHktdGVzdGluZ0BjbG91ZC5nb29nIn0.ZFHyJ9TfKAymN8xETTWqvKyG2uQyuAmsb" +
		"OTZ2TEOqYMvvjhopt3mjJPnIzD_E3siQlJI86ff59De1eK8TXJvG9SH8mdERU8J6tI7v" +
		"UiErCxIGiZ5z7-CqdHin4sVlBLKRtDWZo5UGXbgN41SBGtnCsLmxxU1lZTtNFnH3ezMP" +
		"hzA7t00exo3FUhHNt0AdVqUCfVFEMPvCVCaVhgCRS_ukuNjFeTGpy3RuXlCUPnVA3RR9" +
		"U4anKQE67U6a6vZmMaT8yZDvp0u1S8WFt5XTMfeWeTJcl6ehLZmR_5Onjsb9E3lZvkjC" +
		"bSCysNPy_PpODs8dPVpu2aJSciIFY6K_Emm5A"

	// python3 gen-jwt.py key.pem -jwks=./jwks.json --expire=31536000000
	//  --iss=api-proxy-testing@cloud.goog --aud need-pad
	FakeCloudTokenSingleAudience3 = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRI" +
		"RmJwb0lVcXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJK" +
		"V1QifQ.eyJhdWQiOiJuZWVkLXBhZCIsImV4cCI6MzMxNjY1MzYxMjQsImlhdCI6MTYzM" +
		"DUzNjEyNCwiaXNzIjoiYXBpLXByb3h5LXRlc3RpbmdAY2xvdWQuZ29vZyIsInN1YiI6I" +
		"mFwaS1wcm94eS10ZXN0aW5nQGNsb3VkLmdvb2cifQ.egwbVVrwywLJXBLCg00mg-hS7d" +
		"qsdl2frZdG4rDd_6Vqpt_cTlZZAAC3bppSMCWpDn96y-2-K0zxzjkl3JzcZ8o5KY1F5p" +
		"oVwuOMsxr7WJatrj1-96NSCE4TXuyqtjG2p_0l3vhbE87bA9TN-XkK9-0NOMxUDWZOMK" +
		"n_SM18XT4s2XFZa9z8fFMf2fkmoPiwu9sVKLYwhv94xPwU4VWJs-AWpkyUBIrwR21n8H" +
		"WEIDqb43oK6RzngfkJV4qj253W2KGCkpoq91C06FWptAMwFIyR6Dnu8BAlXyl4le9gxd" +
		"ysPObKgkjxcyyKawhuzZowWy2Kk1fcAUXzuS6tlUkVzQ"
	FakeCloudTokenSingleAudience3Payload = "eyJhdWQiOiJuZWVkLXBhZCIsImV4cCI6" +
		"MzMxNjY1MzYxMjQsImlhdCI6MTYzMDUzNjEyNCwiaXNzIjoiYXBpLXByb3h5LXRlc3Rp" +
		"bmdAY2xvdWQuZ29vZyIsInN1YiI6ImFwaS1wcm94eS10ZXN0aW5nQGNsb3VkLmdvb2cifQ"

	// gen-jwt.py key.pem -jwks=./jwks.json --expire=3153600000
	// --iss="api-proxy-testing@cloud.goog"
	//--aud admin.cloud.goog,bookstore_test_client.cloud.goog > demo.jwt
	FakeCloudTokenMultiAudiences = "eyJhbGciOiJSUzI1NiIsImtpZCI6Imo1bm9ZSXhuR1" +
		"JXNE9CaXV4bXQ2a2wtemVRZ3hjVndmS3NsTmlOWjdKNUkiLCJ0eXAiOiJKV1QifQ." +
		"eyJhdWQiOlsiYWRtaW4uY2xvdWQuZ29vZyIsImJvb2tzdG9yZV90ZXN0X2NsaWVu" +
		"dC5jbG91ZC5nb29nIl0sImV4cCI6NDY5ODMxODk5OSwiaWF0IjoxNTQ0NzE4OTk5" +
		"LCJpc3MiOiJhcGktcHJveHktdGVzdGluZ0BjbG91ZC5nb29nIiwic3ViIjoiYXBp" +
		"LXByb3h5LXRlc3RpbmdAY2xvdWQuZ29vZyJ9.gruGdEcpVCe7tsO50w2DKA9w-FT" +
		"6KdugDOXLNZuopsdPG-2TdVZoLwPKEU94Eu4l67ufibbYmM3mCqqLXDn4WusK22h" +
		"YL5jMMbFXyJkodv1e2MW6W08ZehYlMhO3qU-knBfGVm1f2Dia0b02QYsGRQtB2rb" +
		"Me7l-APbG0XHmoAg9j0fAe5qJYTjFrMr5t72i7BwyqpFriDC_l7bM663DeoFzVA3" +
		"1t3GfDzzjGRJl65OiW-2rKSmSLt8k2mKtZ2ihwF7LF0FetyZzaMhDvQRkuGpaWhF" +
		"HB7Ty8qmsHaRXY4-RKhq1TcBO25qHcYxNXzF_fDFxA0zjdCtzuBdrtraDWA"

	// python3 gen-jwt.py key.pem -jwks=./jwks.json --expire=3153600000
	// --iss="api-proxy-testing@cloud.goog"
	// --aud https://bookstore.endpoints.cloudesf-testing.cloud.goog > demo.jwt
	FakeCloudGrpcBookstoreDefaultToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOiJodHRwczovL2Jvb2tzdG9yZS5lbmRwb2ludHMuY2xvdWRlc2YtdGVzdGluZy5jbG91ZC5nb29nIiwiZXhwIjo0NzM1MzI0NTY2LCJpYXQiOjE1ODE3MjQ1NjYsImlzcyI6ImFwaS1wcm94eS10ZXN0aW5nQGNsb3VkLmdvb2ciLCJzdWIiOiJhcGktcHJveHktdGVzdGluZ0BjbG91ZC5nb29nIn0.LwJXfeXwTg9X2HEdQ5lEkSORq_6ZU2Xl8H4r-TmNwi89oBXxZZ2DKxmquA7bJdHVhhQcPCK9KWo5UA0BpFn3k1u890wMVhptl_u-qp_IsIGXQvEDSf7ZM4AmWXwrEo5yNNVPgIo7ipyTBrEW7cwkRyRlf2cDtnvM_Y4X7CcR1nOB515SaGaGPeMyCTvsy6q5du72d2G8lCaXWFp3w1fx7CxLlj5rR5tEiUswur3cECFnlqQSr68yHqpEHIwAyWBrT1-UmwwNBRhE4Ajzhf9vvFd3iLT7hJ-fzdJTZFWtcSlA8fFmqeWrIZdxRe9kPQ1CK5nYI4MVfuflO4QEU8t_Ag"

	// JWT and JWKS generated with issuer "jwt-client.endpoints.sample.google.com"
	// The second entry corresponds to the private key in the following directory.
	// https://github.com/istio/istio/tree/master/security/tools/jwt/samples
	FakeEndpointsJwks = `{
		"keys":[
		{
			"e":"AQAB",
			"kid":"cgS5aK5-j1u5cqKgcgaGlNem14L9gKWCuUOpNrk5X4M",
			"kty":"RSA",
			"n":"w5PWEX5dQ-kjBkx4ZhzXeXqC7PkhduwZ8hHOkVANIKiNLt1sUr17G1hFe8uJka-T1jBWWi7VqidluXcNAuCtbQQ_m1nZhCOjmA803rAQJJQYPxIYXXVYQ-yAAubG5RA_ImVQaXAmoRC5vvU2BnxMYbvPtGoLLOrpTY123d4m-z094Qh4MMUG0KZr52IFjCzTJR8fGetvYZZfrrEwQn5EXcb3WJYx_kdjMRnPeUUIZdxUJOmAAxE5qADxCB12p00S9T-D9WhqiET8S9MjgXzstoWmFLeDDVakgc14t23uK910NDoYRv6XXq9GyhGa0_PqUD3UCUJC4Sz48Onv6-SyCw"
		},
		{ 
            "e":"AQAB",
            "kid":"DHFbpoIUqrY8t2zpA2qXfCmr5VO5ZEr4RzHU_-envvQ",
            "kty":"RSA",
            "n":"xAE7eB6qugXyCAG3yhh7pkDkT65pHymX-P7KfIupjf59vsdo91bSP9C8H07pSAGQO1MV_xFj9VswgsCg4R6otmg5PV2He95lZdHtOcU5DXIg_pbhLdKXbi66GlVeK6ABZOUW3WYtnNHD-91gVuoeJT_DwtGGcp4ignkgXfkiEm4sw-4sfb4qdt5oLbyVpmW6x9cfa7vs2WTfURiCrBoUqgBo_-4WTiULmmHSGZHOjzwa8WtrtOQGsAFjIbno85jp6MnGGGZPYZbDAa_b3y5u-YpW7ypZrvD8BgtKVjgtQgZhLAGezMt0ua3DRrWnKqTZ0BJ_EyxOGuHJrLsn00fnMQ"
        }
		]
	}`

	// No audiences in this valid token.
	// --iss=jwt-client.endpoints.sample.google.com
	FakeEndpointsToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImNnUzVhSzUtajF1NWNxS2d" +
		"jZ2FHbE5lbTE0TDlnS1dDdVVPcE5yazVYNE0iLCJ0eXAiOiJKV1QifQ.eyJleHAiOjQ2O" +
		"TgzMTk3NTIsImlhdCI6MTU0NDcxOTc1MiwiaXNzIjoiand0LWNsaWVudC5lbmRwb2ludH" +
		"Muc2FtcGxlLmdvb2dsZS5jb20iLCJzdWIiOiJqd3QtY2xpZW50LmVuZHBvaW50cy5zYW1" +
		"wbGUuZ29vZ2xlLmNvbSJ9.Gzwwef04bGz0meRp5q6r4GG3hVlWciIwDq4X5FEEsYUrUJT" +
		"8t0bHyI8Eq7NKYwxg8bppXiMYbHlnQnge8wvUG7YGZaymBP3_32Tc4SlT0xG8ca_O-S4x" +
		"tD_YtRhGlddubup_u_U-SoXqgsAYINrBouD8cIBIbTu68gZtmq7CgsdiU-vh5K3BPCY4A" +
		"hFVkL0n0Pro9C-RtiHcTn6v2nnWMiF6sbyTaxJljpt_PI5AXw2g-nPqeR9pNL-Y0w02Zs" +
		"7CD1Fb6i0jMPeRoCBIQsLCLGTw2yL0hTRRtbFxTjZ2b9Ogvw_r3k8dxR4vaObkvc8pWJW" +
		"i7zQ9iUJoZVrYKzZtOw"

	// python3 gen-jwt.py key.pem -jwks=./jwks.json --expire=3153600000
	// --iss="jwt-client.endpoints.sample.google.com"
	// --aud https://bookstore.endpoints.cloudesf-testing.cloud.goog > demo.jwt
	FakeEndpointsGrpcBookstoreDefaultToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOiJodHRwczovL2Jvb2tzdG9yZS5lbmRwb2ludHMuY2xvdWRlc2YtdGVzdGluZy5jbG91ZC5nb29nIiwiZXhwIjo0NzM1MzI3NDk1LCJpYXQiOjE1ODE3Mjc0OTUsImlzcyI6Imp3dC1jbGllbnQuZW5kcG9pbnRzLnNhbXBsZS5nb29nbGUuY29tIiwic3ViIjoiand0LWNsaWVudC5lbmRwb2ludHMuc2FtcGxlLmdvb2dsZS5jb20ifQ.Ex9QwvLMXWmAQSwPX-qcwJe6G11smOON66b-h3ILdDtHpFPc1VtIDOZxi5yI6gaG7XlYMNdGiI8xEOgC99LsYYrQH9765V_qy9cnBtf8b-U5zJKHmL1KHhzUyV3M9ZSVV6VYzh67WxsPDyaNLj4OSSKfmXe268kYW6fUQzEQ9vmwb06Zt8P9jIWYrnZLNTw2h2iFLAVyzsCFPPzCXpeTKl1YVVJaqBs8bBBRk-DMmew__QZZ9C_DJuWUsl1zWqb5REzU4ZKfxysIfGmXDr8A_FmdS-LwOdbi9qnR25BzrvNc-TIyODPIcbqFC9v1DToJJ9NhDhBgBIRh-tEdU56IEA"

	// Bad Token
	FakeBadToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyenBBMnFYZk" +
		"NtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ." +
		"eyJleHAiOjQ2ODcxODkyNTEsImlhdCI6MTUzMzU4OTI1MSwiaXNzIjoid3JvbmctaXNzdWVy" +
		"QHNlY3VyZS5pc3Rpby5pbyIsInN1YiI6Indyb25nLWlzc3VlckBzZWN1cmUuaXN0aW8uaW8i" +
		"fQ.Ye7RKrEgr3mUxRE1OF5sCaaH6kg_OT-mAM1HI3tTUp0ljVuxZLCcTXPvvEAjyeiNUm8fj" +
		"eeER0fsXv7y8wTaA4FFw9x8NT9xS8pyLi6RsTwdjkq0-Plu93VQk1R98BdbEVT-T5vVz7uA" +
		"CES4LQBqsvvTcLBbBNUvKs_eJyZG71WJuymkkbL5Ki7CB73sQUMl2T3eORC7DJt" +
		"yn_C9Dxy2cwCzHrLZnnGz839_bX_yi29dI4veYCNBgU-9ZwehqfgSCJWYUoBTrdM06" +
		"N3jEemlWB83ZY4OXoW0pNx-ecu3asJVbwyxV2_HT6_aUsdHwTYwHv2hXBjdKEfwZxSsBxbKpA"

	// ./gen-jwt.py key.pem -jwks=./jwks.json --expire=3153600000 --iss="invalid_jwks_provider" --aud bookstore_test_client.cloud.goog
	FakeInvalidJwksProviderToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVc" +
		"XJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJh" +
		"dWQiOiJib29rc3RvcmVfdGVzdF9jbGllbnQuY2xvdWQuZ29vZyIsImV4cCI6NDcwOTkyND" +
		"Y4NSwiaWF0IjoxNTU2MzI0Njg1LCJpc3MiOiJpbnZhbGlkX2p3a3NfcHJvdmlkZXIiLCJz" +
		"dWIiOiJpbnZhbGlkX2p3a3NfcHJvdmlkZXIifQ.WbaMjVS6kyMuTBlvumtAcYlYtt2l-nW" +
		"KNzZOXrVBU_Fg6RLXEsit0EWOhdOh0BQgFtTlUgD2H9iVWsCcWFe5zFQOSOBJplW8OdCgr" +
		"KUzPu_ADehemlx30K_J8mz224k1ve2YiHWFYoKPp7dp-B4xTODjvqNEajFrnX-" +
		"WV5dUcY6y9WIaGWqqMfYjb2Jcojf__JWFOgQwB1vYfGLErhaPpmObWnJi7rDIRDa-hFOfx" +
		"1MXZIWNE9dZKjD8xUUlGC_BsJ62uaNVGTpHV5h_uhehTIX9xmsQwsDGGlyKn4SxVTXvKkY" +
		"6der_JVuTHz1kkbGWjqwa3o1vwFs5gS3nT94ClQ"

	// ./gen-jwt.py key.pem -jwks=./jwks.json --expire=3153600000 --iss="nonexist_jwks_provider" --aud bookstore_test_client.cloud.goog
	FakeNonexistJwksProviderToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJ" +
		"wb0lVcXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1Qif" +
		"Q.eyJhdWQiOiJib29rc3RvcmVfdGVzdF9jbGllbnQuY2xvdWQuZ29vZyIsImV4cCI6NDcx" +
		"MTA0OTc3MiwiaWF0IjoxNTU3NDQ5NzcyLCJpc3MiOiJub25leGlzdF9qd2tzX3Byb3ZpZG" +
		"VyIiwic3ViIjoibm9uZXhpc3Rfandrc19wcm92aWRlciJ9.w56DsKD9Y0VMZn85JvDwds4" +
		"lVjLcj4MEBQgF8lYproPkIR_URO0fcBy28k656y1eBDgldqS7k79_KNTcxWHShoUFXrcCD" +
		"k-_Q3RlBT_DJFhT2qlqhSYnQkqLjhpU7LGjbObi988DscTbzGiJ1VjKhVpEITiho867r11" +
		"Ou48cubokIJTE0T-" +
		"2MKZxKsYn8NRVpdyy39Bp3IUv9AUbk4qEKB69pbfSt5H2Z6P_waYfv6m-GieQZWGlhO90Y" +
		"ytoPuPekKhe8JVV2f5yCwLE89S9ZD8779_1G4UGOsyBfxGvOicoZ9nqtGbJYHnqMN3gjh-" +
		"BWr3cm9Mswm8TCkP0Lv2cvQ"

	// ./gen-jwt.py key.pem -jwks=jwks.json --expire=3153600000 --iss=http://127.0.0.1:32025 --aud=ok_audience
	FakeOpenIDToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lV" +
		"cXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJ" +
		"hdWQiOiJva19hdWRpZW5jZSIsImV4cCI6NDcxMTcxODE2MiwiaWF0IjoxNTU4MTE4MTYyL" +
		"CJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjMyMDI1Iiwic3ViIjoiaHR0cDovLzEyNy4wLjA" +
		"uMTozMjAyNSJ9.O2dM3kilFqDfwrG8qtYMPyy8c_mnSiulsIp_KkfI4tUdaATV5M5Hf-1e" +
		"VPGJXjmkzqG_hf8JHAF8yzjODWt7Cj_6xG21gW2n4NlnVdKb9a3iSQYecZ4hNwiQmCjKNy" +
		"r8vrCkp6wEUShMZvjN330UivnRnHLsyjEliqqL9R9r7TQkM1VpJcm-0G25g7KxKmPC4kuO" +
		"KsjIidjnEuFTuj_gM0PvC_hzK6vHt0vlQ-HfmB1ybKfYR0e1EBEjpWiU5c3u6uHxyUeBTR" +
		"-ATE_AMNnYROxvP9U62ICA10GQYMn-KO5hkzALih2ZsaXbY5iwC9gllf1plpJiNuWpWmqN" +
		"3KSmDA"

	// ./gen-jwt.py key.pem -jwks=jwks.json --expire=3153600000 --iss=http://127.0.0.1:32026 --aud=ok_audience
	FakeInvalidOpenIDToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lV" +
		"cXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJ" +
		"hdWQiOiJva19hdWRpZW5jZSIsImV4cCI6NDcxMTcxOTE2MCwiaWF0IjoxNTU4MTE5MTYwL" +
		"CJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjMyMDI2Iiwic3ViIjoiaHR0cDovLzEyNy4wLjA" +
		"uMTozMjAyNiJ9.ih2BG89Of6MTA331_UVPEif_XTw5WOjnZiVjTIta3i1gaG3suDRTYyd4" +
		"Hn4OvDMpiO-cm1eXbU_n940oFLcjr2HMSxDDSHopCjAB5KedFEi4Mb0V7GQ-stn9UsVvv7" +
		"MQbRY7GBxWmuxyYMXNmzUHvLVT41-UEu6jheIfyQV8nrXfVAIdQSWJSuQnq8_C88cPCIu5" +
		"ZaUv2AMZVFgarjvdJz45JCEKXToX-36_6K6iRGrrgN6k1j8re3tyITxHtkBMwB7EyY7aRK" +
		"qjWCGaGFreIGKNzY8Chcw_a8HZAAz7nNfkkBuIgZs2GEVwkqQeDWgtrct1oztS8bYcguro" +
		"zMsCFw"

	// ./gen-jwt.py key.pem -jwks=jwks.json --expire=3153600000 --iss=http://127.0.0.1:32027 --aud=ok_audience
	FakeNonexistOpenIDToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lV" +
		"cXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJ" +
		"hdWQiOiJva19hdWRpZW5jZSIsImV4cCI6NDcxMTcxOTMzNCwiaWF0IjoxNTU4MTE5MzM0L" +
		"CJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjMyMDI3Iiwic3ViIjoiaHR0cDovLzEyNy4wLjA" +
		"uMTozMjAyNyJ9.jXody0fj7PMdYaINWggZch4fnoFo7bGeF6cMqJnwgdanSSW_FcwXsx2X" +
		"dWoHLF153Qt0OGAZOE29ffti9LLkKzyYAGjsvatbPj0crtSAwQAzCyqy8-BMXBxawfNWuK" +
		"Inmvyk1Xn9Hf-midyqlQdQGztDwksleTFxFQzd3MoTY7z8Pw_WxTrpQTI1HAjboE6OnsH4" +
		"rLcncoKX5MX8kOnEZjO0US1nfbPHQnpjKdgq_42uusJVCYau__zMMoEhLlCYxTKrdmWQ_j" +
		"LW0v8IOSbixa74w9TwlCr0TKzsd-8e4Jr4gksDNxtzJWPwKAuvvd6J9q5CZXQ-WmszDNCK" +
		"vYbOQA"

	// Copied from: https://github.com/google/jwt_verify_lib/pull/33/files
	//  python generate-jwt.py --iss="fake.issuer" "fake.audience" sa_file
	// Modified script to set long expiration time
	X509Token = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjgyY2ZkNzk3OTAz" +
		"MDYzYTBiNzhjZTFjYmY1ZTJmZTAzNmE2ZGUyNDIifQ.eyJpc3MiOiJmYWtlLmlzc3VlciI" +
		"sImlhdCI6MTU3ODY4NTY4MSwiYXVkIjoiZmFrZS5hdWRpZW5jZSIsImV4cCI6MzE1MzYwM" +
		"DAwMCwic3ViIjoiZmFrZS5pc3N1ZXIifQ.fMsL-HA3pWK77kl0GdAPqw56wwRYTF9T6WLC" +
		"tvmEQ4KATA0uLAFyVGISrouNTfntHgFT8ObajDiLKZpBpbMQzqSRxEJomW5lR7UzZ8-V6z" +
		"ne5t7ZwkYUjzAdpTnuFNUhdYzkOKeuH_UIYw-XSFUJD8UJC3w1eBarGu32k_LyXQ3zt08F" +
		"41G8sbg5JVarjYAnYCQbSnqtKxQVhuQ_Lrwf3mcrnSqeRAPummfK1RB6lp2l9SW3A9IqX_" +
		"NZGEelQRvYU8fo8x5rlCK_UI9oIAlEiMStLQ7AntkXwE6yX_yw4pqlh7NtEiphcBDkXect" +
		"qm8FGi5udDWS7dvUXf01VokK9g"
)

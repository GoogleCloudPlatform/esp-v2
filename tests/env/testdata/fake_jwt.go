// Copyright 2018 Google Cloud Platform Proxy Authors
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

const (
	// From https://github.com/istio/istio/blob/master/security/tools/jwt/samples/demo.jwt
	FakeGoodToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyen" +
		"BBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ." +
		"eyJleHAiOjQ2ODU5ODk3MDAsImZvbyI6ImJhciIsImlhdCI6MTUzMjM4OT" +
		"cwMCwiaXNzIjoidGVzdGluZ0BzZWN1cmUuaXN0aW8uaW8iLCJzdWIiOiJ0Z" +
		"XN0aW5nQHNlY3VyZS5pc3Rpby5pbyJ9.CfNnxWP2tcnR9q0vxyxweaF3ovQY" +
		"HYZl82hAUsn21bwQd9zP7c-LS9qd_vpdLG4Tn1A15NxfCjp5f7QNBUo-KC9PJ" +
		"qYpgGbaXhaGx7bEdFWjcwv3nZzvc7M__ZpaCERdwU7igUmJqYGBYQ51vr2njU9" +
		"ZimyKkfDe3axcyiBZde7G6dabliUosJvvKOPcKIWPccCgefSj_GNfwIip3-SsFd" +
		"lR7BtbVUcqR-yv-XOxJ3Uc1MI0tz3uMiiZcyPV7sNCU4KRnemRIMHVOfuvHsU60" +
		"_GhGbiSFzgPTAa9WTltbnarTbxudb_YEOx12JiwYToeX0DCPb43W1tzIBxgm8NxUg"

	FakeBadToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyenBBMnFYZk" +
		"NtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ." +
		"eyJleHAiOjQ2ODcxODkyNTEsImlhdCI6MTUzMzU4OTI1MSwiaXNzIjoid3JvbmctaXNzdWVy" +
		"QHNlY3VyZS5pc3Rpby5pbyIsInN1YiI6Indyb25nLWlzc3VlckBzZWN1cmUuaXN0aW8uaW8i" +
		"fQ.Ye7RKrEgr3mUxRE1OF5sCaaH6kg_OT-mAM1HI3tTUp0ljVuxZLCcTXPvvEAjyeiNUm8fj" +
		"eeER0fsXv7y8wTaA4FFw9x8NT9xS8pyLi6RsTwdjkq0-Plu93VQk1R98BdbEVT-T5vVz7uA" +
		"CES4LQBqsvvTcLBbBNUvKs_eJyZG71WJuymkkbL5Ki7CB73sQUMl2T3eORC7DJt" +
		"yn_C9Dxy2cwCzHrLZnnGz839_bX_yi29dI4veYCNBgU-9ZwehqfgSCJWYUoBTrdM06" +
		"N3jEemlWB83ZY4OXoW0pNx-ecu3asJVbwyxV2_HT6_aUsdHwTYwHv2hXBjdKEfwZxSsBxbKpA"

	FakeJwks = `{
    "keys":[
      {
      	"e":"AQAB",
      	"kid":"DHFbpoIUqrY8t2zpA2qXfCmr5VO5ZEr4RzHU_-envvQ",
      	"kty":"RSA",
      	"n":"xAE7eB6qugXyCAG3yhh7pkDkT65pHymX-P7KfIupjf59vsdo91bSP9C8H07pSAGQO1MV_xFj9VswgsCg4R6otmg5PV2He95lZdHtOcU5DXIg_pbhLdKXbi66GlVeK6ABZOUW3WYtnNHD-91gVuoeJT_DwtGGcp4ignkgXfkiEm4sw-4sfb4qdt5oLbyVpmW6x9cfa7vs2WTfURiCrBoUqgBo_-4WTiULmmHSGZHOjzwa8WtrtOQGsAFjIbno85jp6MnGGGZPYZbDAa_b3y5u-YpW7ypZrvD8BgtKVjgtQgZhLAGezMt0ua3DRrWnKqTZ0BJ_EyxOGuHJrLsn00fnMQ"
      }
    ]
  }`

	// This token is generated, by run gen-jwt.py key.pem -jwks=./test_jwks.json
	// --expire=3153600000 --aud bookstore_test_client.cloud.goog > test_demo.jwt.
	// using https://github.com/istio/istio/tree/master/security/tools/jwt/samples
	FakeGoodTokenSingleAud = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyen" +
		"BBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOiJib2" +
		"9rc3RvcmVfdGVzdF9jbGllbnQuY2xvdWQuZ29vZyIsImV4cCI6NDY5ODIzMTYyMCwiaWF0I" +
		"joxNTQ0NjMxNjIwLCJpc3MiOiJ0ZXN0aW5nQHNlY3VyZS5pc3Rpby5pbyIsInN1YiI6InRl" +
		"c3RpbmdAc2VjdXJlLmlzdGlvLmlvIn0.RM7PA2RajHn_1dYwrZweFQ6WhnwDq_w5N4ew2r1" +
		"XJKIbVn0rZ2maxLpWkWWGfcHeJzve4F6PfERkj9V47xLtjpBkjgwPszjCJ4etWlTek92oHU" +
		"z0JBqB6iK4GPhkWb2PSpMQGU97x4fP3FoNAa1jVYzZ68CZVflW4-ucr5tZ1oCle6dLa6B6u" +
		"oPX77J6Aq9247Nd-HnDWjcRCRAbpoU7Oo4bVCpcPXzQAH_nl5bXoXz2oTVcBLHDy_nl9_M0" +
		"_n2FOS-XQw62drXY0T1PSEnYXtAbQQPMLdnBh_17OGyzvY4BlP_97mprDFbK-bue8Hszljw" +
		"vjk-JzJJ1r6s3mgYWZw"

		// created by --aud admin.cloud.goog with scripts above
	FakeGoodAdminToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyenBBM" +
		"nFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOiJhZG1pbi" +
		"5jbG91ZC5nb29nIiwiZXhwIjo0Njk4MjM0NTY3LCJpYXQiOjE1NDQ2MzQ1NjcsImlzcyI6In" +
		"Rlc3RpbmdAc2VjdXJlLmlzdGlvLmlvIiwic3ViIjoidGVzdGluZ0BzZWN1cmUuaXN0aW8uaW" +
		"8ifQ.BqrYjROnDJ13ivw8alZML934rtYqrK9BcSURNv2impwyW9Z4tjTkDoaaM4rU9osZD-h" +
		"u3v6tFeRJHFLNE7C4BEDJj9aSJoM_jGfM3D6ZCvML5pA8Ci_EdqVNvrTtaAm3Qhw_jfEPcHO" +
		"10f_xi3Y9FMqldPjJ8KUft62Lpqodbtp9kCmx6uZ0vZy_FxI3P4N-p27sCUue07yfiBWSGcJ" +
		"ss6FsPfaTYlhyMQ3TuUnabOQeD9BsvIXclbjf1QNOuEQQGLdgPZCt6TtNXJ88nvM4CzlA2-g" +
		"0mRojC-joaETdsBL6Xe9LxgkGXzxR5SbJdy2eI-V5srirnO6_WR2-yBkwJw"

		// created by --aud admin.cloud.goog,bookstore_test_client.cloud.goog
	FakeGoodTokenMultiAud = "eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQye" +
		"nBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiY" +
		"WRtaW4uY2xvdWQuZ29vZyIsImJvb2tzdG9yZV90ZXN0X2NsaWVudC5jbG91ZC5nb29nIl0s" +
		"ImV4cCI6NDY5ODIzNTc1OCwiaWF0IjoxNTQ0NjM1NzU4LCJpc3MiOiJ0ZXN0aW5nQHNlY3V" +
		"yZS5pc3Rpby5pbyIsInN1YiI6InRlc3RpbmdAc2VjdXJlLmlzdGlvLmlvIn0.W13l-9IVwI" +
		"XfAtZPsA4QwGZ9uYRmdUpRxj8gbRF0j_xgCJNbwByRDAiLGJWbozRWZnyMjzUWUHOTOLqPp" +
		"nlFkS_Gmx8sF2sS8gSQjGIxClaeqCjQAQNHRgA-8DU-MAP8vsDoCqzj8vuhDjVZr1JPxyAH" +
		"4ze2Ssut1QvaYN8TfcvmWwjWBA4seMj8S9AC0VrjrqzbhvAjF63arKOqtokDlYbf-fN_Nx1" +
		"WfvaJSd06CHxTs8V-MPGqMkR1HeqgS1LhgVdWCTyvo_1KPzXrGMjXRsT-Oyv8I2BrGF5dhv" +
		"iXdp21lFou2M7bri_rxPJTU7ui3aoFtJXQEYiwW4tusWiPTA"
)

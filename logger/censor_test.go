package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCensorTextWithRegex_DefaultPhrases(t *testing.T) {

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simplified phrase with default phrases",
			input:    `password=supersecret&secret=topsecret&"access_token":"abc123"&"refresh_token":"r123"&"id_token":"i123"`,
			expected: `password=[REDACTED]&secret=[REDACTED]&"access_token":[REDACTED]&"refresh_token":[REDACTED]&"id_token":[REDACTED]`,
		},

		{name: "real-life http get request with default phrases",
			input:    " Sending GET request:  url=https://cdi.dev/fabric_manager/api/v1/tenants/12345678-1234-1234-1234-123456789012?tenant_uuid=12345678-1234-1234-1234-123456789012, headers=map[Authorization:[Bearer eyJh8B-LMMw]];",
			expected: " Sending GET request:  url=https://cdi.dev/fabric_manager/api/v1/tenants/12345678-1234-1234-1234-123456789012?tenant_uuid=12345678-1234-1234-1234-123456789012, headers=map[Authorization:[Bearer[REDACTED]]];",
		},

		{name: "real-life http post request with default phrases",
			input:    "Initiating POST request: ; endpoint=/realms/12345678-1234-1234-1234-123456789012/protocol/openid-connect/token, payload=client_id=cdi&client_secret=SensitiveInfo&grant_type=password&password=foobar&response=id_token+token&scope=openid&username=jdoe&token=eyJh-LMMw",
			expected: "Initiating POST request: ; endpoint=/realms/12345678-1234-1234-1234-123456789012/protocol/openid-connect/token, payload=client_id=cdi&client_secret=[REDACTED]&grant_type=password&password=[REDACTED]&response=id_token+token&scope=openid&username=jdoe&token=[REDACTED]",
		},
		{name: "simplified real-life http response with default phrases",
			input:    `response_body={"access_token":"eyJhn0.eyJQ.W418g","expires_in":1750,"refresh_expires_in":7200,"refresh_token":"eyJhbGcX43A","token_type":"Bearer","id_token":"eyJhbGciOi36vyHeg","not-before-policy":0,"session_state":"d8321164-bd12-4606-922e-f170f2b2088d","scope":"openid pgcdi_privileges email profile"};`,
			expected: `response_body={"access_token":[REDACTED],"expires_in":1750,"refresh_expires_in":7200,"refresh_token":[REDACTED],"token_type":"Bearer","id_token":[REDACTED],"not-before-policy":0,"session_state":"d8321164-bd12-4606-922e-f170f2b2088d","scope":"openid pgcdi_privileges email profile"};`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			observed := CensorTextWithRegex(tc.input)
			assert.Equal(t, tc.expected, observed)
		})
	}
}

func TestCensorTextWithRegex_CustomPhraseAdded(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		customRedactor []CensorRegexRedactor
		expected       string
	}{
		{name: "custom redactor with single phrase",
			input: `password=supersecret&secret=topsecret&income=123&"access_token":"abc123"&"refresh_token":"r123"&"id_token":"i123"`,
			customRedactor: []CensorRegexRedactor{
				{
					Regex:  `(?i)income=.*?&`,
					Prefix: "income=",
					Suffix: "&",
				},
			},
			expected: `password=[REDACTED]&secret=[REDACTED]&income=[REDACTED]&"access_token":[REDACTED]&"refresh_token":[REDACTED]&"id_token":[REDACTED]`,
		},

		{name: "custom redactor with single phrase; no pre/su fixes, case insensitive",
			input: `password=supersecret&secret=topsecret&"access_token":"abc123"&"refresh_token":"r123"&"id_token":"i123",foobar,DiRtyWoRd`,
			customRedactor: []CensorRegexRedactor{
				{
					Regex: `(?i)dirtyWord`,
				},
			},
			expected: `password=[REDACTED]&secret=[REDACTED]&"access_token":[REDACTED]&"refresh_token":[REDACTED]&"id_token":[REDACTED],foobar,[REDACTED]`,
		},

		{name: "custom redactor with single phrase; no pre/su fixes, case sensitive",
			input: `password=supersecret&secret=topsecret&"access_token":"abc123"&"refresh_token":"r123"&"id_token":"i123",foobar,CATch-22`,
			customRedactor: []CensorRegexRedactor{
				{
					Regex: `CATch-22`,
				},
			},
			expected: `password=[REDACTED]&secret=[REDACTED]&"access_token":[REDACTED]&"refresh_token":[REDACTED]&"id_token":[REDACTED],foobar,[REDACTED]`,
		},

		{name: "custom redactor with 2 phrases",
			input: `password=supersecret&secret=topsecret&income=123&ssn=987&"access_token":"abc123"&"refresh_token":"r123"&"id_token":"i123"`,
			customRedactor: []CensorRegexRedactor{
				{
					Regex:  `(?i)income=.*?&`,
					Prefix: "income=",
					Suffix: "&",
				},
				{
					Regex:  `(?i)ssn=.*?&`,
					Prefix: "ssn=",
					Suffix: "&",
				},
			},
			expected: `password=[REDACTED]&secret=[REDACTED]&income=[REDACTED]&ssn=[REDACTED]&"access_token":[REDACTED]&"refresh_token":[REDACTED]&"id_token":[REDACTED]`,
		},
		{name: "custom redactor with 2 phrases case sensitive",
			input: `password=supersecret&secret=topsecret&Income=123&SSN=987&"access_token":"abc123"&"refresh_token":"r123"&"id_token":"i123"`,
			customRedactor: []CensorRegexRedactor{
				{
					Regex:  `Income=.*?&`,
					Prefix: "Income=",
					Suffix: "&",
				},
				{
					Regex:  `SSN=.*?&`,
					Prefix: "SSN=",
					Suffix: "&",
				},
			},
			expected: `password=[REDACTED]&secret=[REDACTED]&Income=[REDACTED]&SSN=[REDACTED]&"access_token":[REDACTED]&"refresh_token":[REDACTED]&"id_token":[REDACTED]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			observed := CensorTextWithRegex(tc.input, tc.customRedactor...)
			assert.Equal(t, tc.expected, observed)
		})
	}
}

func TestCensorTextWithRegex_EmptySuffix(t *testing.T) {

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "phrase with access token",
			input:    `"access_token":"abc123"`,
			expected: `"access_token":[REDACTED]`,
		},
		{name: "phrase with refresh token",
			input:    `"refresh_token":"abc123"`,
			expected: `"refresh_token":[REDACTED]`,
		},
		{name: "phrase with id token",
			input:    `"id_token":"abc123"`,
			expected: `"id_token":[REDACTED]`,
		},
		{name: "phrase with all tokens combined",
			input:    `"access_token":"abc123"&"refresh_token":"abc123"&"id_token":"abc123"`,
			expected: `"access_token":[REDACTED]&"refresh_token":[REDACTED]&"id_token":[REDACTED]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			observed := CensorTextWithRegex(tc.input)
			assert.Equal(t, tc.expected, observed)
		})
	}

}

func TestCensorTextWithRegex_NoCustomPhrases(t *testing.T) {
	input := `some text with no forbidden content`
	observed := CensorTextWithRegex(input)
	assert.Equal(t, input, observed)
}

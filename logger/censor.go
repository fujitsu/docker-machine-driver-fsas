package logger

import (
	"fmt"
	"regexp"
)

type CensorRegexRedactor struct {
	Regex  string
	Prefix string
	Suffix string
}

/*
The phrases below were prepared mainly for handling HTTP requests/responses
where parameters are separated with ampersand (&) e.g. payload=client_id=cdi&client_secret=Pa$$word&grant_type=password.
*/
var defaultForbiddenPhrases = []CensorRegexRedactor{
	{Regex: "(?i)password=.*?&",
		Prefix: "password=",
		Suffix: "&",
	},
	{Regex: "(?i)secret=.*?&",
		Prefix: "secret=",
		Suffix: "&",
	},
	// Match tokens in query strings and form payloads. Capture group 1 preserves
	// the leading delimiter while the value stops before the next field or log delimiter.
	{Regex: `(?i)(^|[?&]|payload=)access_token=[^&\s,;]*`,
		Prefix: `${1}access_token=`,
	},
	{Regex: `(?i)(^|[?&]|payload=)refresh_token=[^&\s,;]*`,
		Prefix: `${1}refresh_token=`,
	},
	{Regex: `(?i)(^|[?&]|payload=)id_token=[^&\s,;]*`,
		Prefix: `${1}id_token=`,
	},
	{Regex: "(?i)&token=.*",
		Prefix: "&token=",
	},
	{Regex: `(?i)\[Bearer.*?\]`,
		Prefix: "[Bearer",
		Suffix: "]",
	},
	// Preserve delimiters and the OpenSSH key type while redacting only the
	// base64 key material. This leaves YAML comments and subsequent log fields intact.
	{Regex: `(?m)(^|[ \t=-])((?:ssh-(?:rsa|dss|ed25519)|ecdsa-sha2-nistp(?:256|384|521)|sk-(?:ssh-ed25519|ecdsa-sha2-nistp256)@openssh\.com)(?:-cert-v01@openssh\.com)?)[ \t]+[A-Za-z0-9+/]+={0,3}`,
		Prefix: `${1}${2} `,
	},
	// Preserve YAML indentation and the field name while redacting arbitrary
	// single-line file content, including gzip+base64 write_files payloads.
	{Regex: `(?m)^([ \t-]*content:[ \t]*).+$`,
		Prefix: `${1}`,
	},
	// Match compact or formatted JSON fields. The value expression accepts escaped
	// characters without stopping at an escaped quote.
	{Regex: `(?i)"access_token"\s*:\s*"([^"\\]|\\.)*"`,
		Prefix: `"access_token":`,
	},
	{Regex: `(?i)"refresh_token"\s*:\s*"([^"\\]|\\.)*"`,
		Prefix: `"refresh_token":`,
	},
	{Regex: `(?i)"id_token"\s*:\s*"([^"\\]|\\.)*"`,
		Prefix: `"id_token":`,
	},
	{Regex: `(?i)"password"\s*:\s*"([^"\\]|\\.)*"`,
		Prefix: `"password":`,
	},
	{Regex: `(?i)"client_secret"\s*:\s*"([^"\\]|\\.)*"`,
		Prefix: `"client_secret":`,
	},
	{Regex: "(?i)SUSEConnect -r.*?-e", Prefix: "SUSEConnect -r ", Suffix: " -e"},
}

/*
CensorTextWithRegex Returns text with censored phrases. Additional phrases that should be hidden in text
might be added as param 'forbiddenPhrases'. If param 'forbiddenPhrases' is not set then
only default forbidden phrases are taken into account
*/
func CensorTextWithRegex(text string, forbiddenPhrases ...CensorRegexRedactor) string {
	forbiddenPhrases = append(defaultForbiddenPhrases, forbiddenPhrases...)
	return censorTextWithRegex(text, forbiddenPhrases)
}

/*
censorTextWithRegex Returns text with censored phrases. Param 'forbiddenPhrases' contains phrases
that should not be seen - they are replaced with fixed text
*/
func censorTextWithRegex(text string, forbiddenPhrases []CensorRegexRedactor) string {
	var censoredText = text

	for _, f := range forbiddenPhrases {
		re := regexp.MustCompile(f.Regex)
		censoredText = re.ReplaceAllString(censoredText, fmt.Sprintf("%s[REDACTED]%s", f.Prefix, f.Suffix))
	}

	return censoredText
}

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
	{Regex: "(?i)&token=.*",
		Prefix: "&token=",
	},
	{Regex: `(?i)\[Bearer.*?\]`,
		Prefix: "[Bearer",
		Suffix: "]",
	},
	{Regex: `(?i)"access_token":".*?"`,
		Prefix: `"access_token":`,
	},
	{Regex: `(?i)"refresh_token":".*?"`,
		Prefix: `"refresh_token":`,
	},
	{Regex: `(?i)"id_token":".*?"`,
		Prefix: `"id_token":`,
	},
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

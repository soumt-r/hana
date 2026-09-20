package stdimpl

import (
	"regexp"
	"strings"

	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/value"
)

// Regular expressions use the syntax Go's regexp and JavaScript share (the
// browser engines run the pattern as a JavaScript RegExp). Empty matches are
// ignored by 모두찾기/치환/분할, which is where the two engines' rules differ.

func compile(pattern string) (*regexp.Regexp, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, errs.New(errs.RegexInvalid, pattern)
	}
	return re, nil
}

// textAndPattern reads the leading (text, pattern) arguments every regex
// function starts with.
func textAndPattern(args []interface{}) (string, *regexp.Regexp, error) {
	text, err := stringArg(args, 0)
	if err != nil {
		return "", nil, err
	}
	pattern, err := stringArg(args, 1)
	if err != nil {
		return "", nil, err
	}
	re, err := compile(pattern)
	return text, re, err
}

func regexTest(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	text, re, err := textAndPattern(args)
	if err != nil {
		return nil, err
	}
	return re.MatchString(text), nil
}

func regexFind(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	text, re, err := textAndPattern(args)
	if err != nil {
		return nil, err
	}
	loc := re.FindStringIndex(text)
	if loc == nil {
		return nil, nil
	}
	return text[loc[0]:loc[1]], nil
}

// regexGroups is the first match as a list: the whole match, then each group
// (비어있음 for a group that did not take part).
func regexGroups(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	text, re, err := textAndPattern(args)
	if err != nil {
		return nil, err
	}
	loc := re.FindStringSubmatchIndex(text)
	if loc == nil {
		return nil, nil
	}
	return value.NewList(groupList(text, loc)), nil
}

func groupList(text string, loc []int) []interface{} {
	out := make([]interface{}, len(loc)/2)
	for g := range out {
		if loc[2*g] >= 0 {
			out[g] = text[loc[2*g]:loc[2*g+1]]
		}
	}
	return out
}

// matches are the non-empty matches of re in text, as submatch index lists.
func matches(re *regexp.Regexp, text string) [][]int {
	var out [][]int
	for _, loc := range re.FindAllStringSubmatchIndex(text, -1) {
		if loc[1] > loc[0] {
			out = append(out, loc)
		}
	}
	return out
}

func regexFindAll(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	text, re, err := textAndPattern(args)
	if err != nil {
		return nil, err
	}
	out := []interface{}{}
	for _, loc := range matches(re, text) {
		out = append(out, text[loc[0]:loc[1]])
	}
	return value.NewList(out), nil
}

// regexReplace swaps every match for the replacement, where $1..$9 (or $10...)
// are the match's groups and $$ is a dollar sign.
func regexReplace(args []interface{}) (interface{}, error) {
	if err := exactly(args, 3); err != nil {
		return nil, err
	}
	text, re, err := textAndPattern(args)
	if err != nil {
		return nil, err
	}
	replacement, err := stringArg(args, 2)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	last := 0
	for _, loc := range matches(re, text) {
		out.WriteString(text[last:loc[0]])
		out.WriteString(expandReplacement(replacement, groupList(text, loc)))
		last = loc[1]
	}
	out.WriteString(text[last:])
	return out.String(), nil
}

func expandReplacement(replacement string, groups []interface{}) string {
	var out strings.Builder
	for i := 0; i < len(replacement); i++ {
		c := replacement[i]
		if c != '$' || i+1 >= len(replacement) {
			out.WriteByte(c)
			continue
		}
		next := replacement[i+1]
		if next == '$' {
			out.WriteByte('$')
			i++
			continue
		}
		if next < '0' || next > '9' {
			out.WriteByte(c)
			continue
		}
		j := i + 1
		for j < len(replacement) && replacement[j] >= '0' && replacement[j] <= '9' {
			j++
		}
		n := 0
		for _, d := range replacement[i+1 : j] {
			n = n*10 + int(d-'0')
		}
		if n < len(groups) {
			if s, ok := groups[n].(string); ok {
				out.WriteString(s)
			}
		} else {
			out.WriteString(replacement[i:j])
		}
		i = j - 1
	}
	return out.String()
}

func regexSplit(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	text, re, err := textAndPattern(args)
	if err != nil {
		return nil, err
	}
	out := []interface{}{}
	last := 0
	for _, loc := range matches(re, text) {
		out = append(out, text[last:loc[0]])
		last = loc[1]
	}
	out = append(out, text[last:])
	return value.NewList(out), nil
}

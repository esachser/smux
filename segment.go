package smux

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type segment interface {
	Match(p string, parms *[]PathParam) bool
	CatchAll() bool
	String() string
	Comparable() string
	NumVars() int
}

type segmentstring string

func (s segmentstring) String() string {
	return string(s)
}

func (s segmentstring) Comparable() string {
	return string(s)
}

func (s segmentstring) Match(p string, parms *[]PathParam) bool {
	return p == string(s)
}

func (s segmentstring) CatchAll() bool {
	return false
}

func (s segmentstring) NumVars() int {
	return 0
}

type segmentcatchallstring struct{}

func (s segmentcatchallstring) String() string {
	return "*"
}

func (s segmentcatchallstring) Comparable() string {
	return "*"
}

func (s segmentcatchallstring) Match(p string, parms *[]PathParam) bool {
	return true
}

func (s segmentcatchallstring) CatchAll() bool {
	return true
}

func (s segmentcatchallstring) NumVars() int {
	return 1
}

type segmentregex struct {
	r        *regexp.Regexp
	original string
	fields   []string
}

var bracketSimplistic = regexp.MustCompile(`\{[a-zA-Z]\w*(?::[a-zA-Z]\w*)?\}`)
var pathTypesSubstitutions = map[string]string{
	"int":    `-?\d+`,
	"uint":   `\d+`,
	"uuidv4": `[[:xdigit:]]{8}-[[:xdigit:]]{4}-4[[:xdigit:]]{3}-[89aAbB][[:xdigit:]]{3}-[[:xdigit:]]{12}`,
	"id":     `[[:xdigit:]]+`,
}

func pathSubstitute(s string) (string, string, error) {
	splt := strings.Split(s, ":")
	fieldName := splt[0]
	if len(fieldName) > 0 && fieldName[0] == '{' {
		fieldName = fieldName[1:]
	}

	if len(splt) == 1 {
		l := len(fieldName)
		if l > 0 && fieldName[l-1] == '}' {
			fieldName = fieldName[:l-1]
		}
		return fieldName, "(.+?)", nil
	} else if len(splt) == 2 {
		t := splt[1]
		if len(t) > 0 {
			t = t[:len(t)-1]
		}
		change, f := pathTypesSubstitutions[t]
		if !f {
			return "", "", fmt.Errorf("invalid type %v on bracket %v", t, s)
		} else {
			return fieldName, "(" + change + ")", nil
		}
	} else {
		return "", "", nil
	}
}

func NewSegmentRegex(r string) (segment, error) {
	if strings.Contains(r, "/") {
		return nil, fmt.Errorf("%v must be a segment of path", r)
	}

	// This block will remove brackets and test if the rest is valid url path
	// Replace bracket things with nothing
	r1 := bracketSimplistic.ReplaceAllString(r, "")
	u, err := url.Parse(r1)
	if err != nil || u.EscapedPath() != r1 {
		return nil, fmt.Errorf("invalid url path %v", r)
	}

	findings := bracketSimplistic.FindAllStringIndex(r, -1)
	fieldNames := make([]string, len(findings))

	r2 := "^"
	if len(findings) == 0 {
		r2 = regexp.QuoteMeta(r)
	} else {
		init := 0
		for i, f := range findings {
			fi := f[0]
			fe := f[1]
			r2 += regexp.QuoteMeta(r[init:fi])
			fname, sub, err := pathSubstitute(r[fi:fe])
			if err != nil {
				return nil, err
			}
			fieldNames[i] = fname
			r2 += sub
			init = fe
		}
		r2 += regexp.QuoteMeta(r[init:])
	}
	r2 += "$"

	if err != nil {
		return nil, err
	}

	reg, err := regexp.Compile(r2)
	if err != nil {
		return nil, err
	}
	return segmentregex{r: reg, original: r, fields: fieldNames}, nil
}

func (r segmentregex) String() string {
	return r.original
}

func (r segmentregex) Comparable() string {
	return r.r.String()
}

func (r segmentregex) Match(p string, parms *[]PathParam) bool {
	findings := r.r.FindAllStringSubmatch(p, -1)
	if len(findings) == 0 {
		return false
	}
	if len(findings[0]) == 0 {
		return false
	}

	if !(len(findings) == 1) {
		return false
	}

	// ctx.pathKeys = append(ctx.pathKeys, r.fields...)
	// ctx.pathParams = append(ctx.pathParams, findings[0][1:]...)
	// ctx.AddPathParams(r.fields, findings[0][1:])
	// for i, finded := range findings[0][1:] {
	// 	ctx.PathParams[r.fields[i]] = finded
	// }
	// parms := make([]PathParam, len(r.fields))
	for i := range findings[0][1:] {
		*parms = append(*parms, PathParam{Key: r.fields[i], Value: findings[0][1+i]})
	}
	return true
}

func (s segmentregex) NumVars() int {
	return strings.Count(s.original, "{")
}

func (segmentregex) CatchAll() bool {
	return false
}

type segmentany struct {
	comp      string
	original  string
	fieldname string
}

func newSegmentAny(s string) segment {
	s1 := s[1 : len(s)-1]
	seg := segmentany{comp: "(.+?)", original: s, fieldname: s1}

	return seg
}

func (s segmentany) String() string {
	return s.original
}

func (s segmentany) Comparable() string {
	return s.comp
}

func (s segmentany) Match(p string, parms *[]PathParam) bool {
	// ctx.pathKeys = append(ctx.pathKeys, s.fieldname)
	// ctx.pathParams = append(ctx.pathParams, p)
	// ctx.AddPathParam(s.fieldname, p)
	// ctx.PathParams[s.fieldname] = p
	if len(p) == 0 {
		return false
	}
	*parms = append(*parms, PathParam{Key: s.fieldname, Value: p})
	return true
}

func (s segmentany) NumVars() int {
	return 1
}

func (s segmentany) CatchAll() bool {
	return false
}

func createSegment(s string) (segment, error) {
	if s == "{*}" {
		return segmentcatchallstring{}, nil
	} else if strings.Contains(s, "{") {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") && !strings.Contains(s, ":") {
			return newSegmentAny(s), nil
		}
		return NewSegmentRegex(s)
	} else {
		return segmentstring(s), nil
	}
}

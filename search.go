package smux

import (
	"fmt"
	"net/url"
	"strings"
)

type searchkind int

const (
	searchstatic searchkind = iota + 1
	searchany
	searchnumber
	searchsignednumber
	searchuuid
	searchuuidv4
	searchid
)

type searchnode struct {
	kind      searchkind
	str       string
	paramname string
	next      *searchnode
	idx       int
	all       string
	final     bool
	compare   string
	original  string
	numvars   int
}

var _ segment = &searchnode{}

func (s *searchnode) Match(p string, parms *[]PathParam) bool {
	return s.search(p, parms)
}

func (searchnode) CatchAll() bool {
	return false
}

func (s searchnode) String() string {
	return s.original
}

func (s searchnode) Comparable() string {
	return s.compare
}

func (s searchnode) NumVars() int {
	return s.numvars
}

var searchPathTypesSubstitutions = map[string]searchkind{
	"int":    searchsignednumber,
	"uint":   searchnumber,
	"uuidv4": searchuuidv4,
	"uuid":   searchuuid,
	"id":     searchid,
}

func searchPathSubstitute(s string) (string, searchkind, error) {
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
		return fieldName, searchany, nil
	} else if len(splt) == 2 {
		t := splt[1]
		if len(t) > 0 {
			t = t[:len(t)-1]
		}
		change, f := searchPathTypesSubstitutions[t]
		if !f {
			return "", 0, fmt.Errorf("invalid type %v on bracket %v", t, s)
		} else {
			return fieldName, change, nil
		}
	} else {
		return "", 0, nil
	}
}

func newSearchNode(r string) (segment, error) {
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

	sn := searchnode{}
	snp := &sn

	sn.compare = ""

	if len(findings) == 0 {
		snp.next = &searchnode{kind: searchstatic, str: r}
		sn.compare += r
	} else {
		init := 0
		for _, f := range findings {
			fi := f[0]
			fe := f[1]
			if fi > init {
				snp.next = &searchnode{kind: searchstatic, str: r[init:fi]}
				snp = snp.next
				sn.compare += r[init:fi]
			}
			fname, sub, err := searchPathSubstitute(r[fi:fe])
			if err != nil {
				return nil, err
			}
			snp.next = &searchnode{kind: sub, paramname: fname}
			snp = snp.next
			sn.compare += fmt.Sprintf("{%v}", sub)
			init = fe
		}
		if init < len(r) {
			snp.next = &searchnode{kind: searchstatic, str: r[init:]}
			sn.compare += r[init:]
		}
	}

	sn.next.compare = sn.compare
	sn.next.original = r

	sn.next.numvars = 0
	snp = sn.next
	for snp != nil {
		if snp.kind != searchstatic {
			sn.next.numvars += 1
		}
		snp = snp.next
	}

	return sn.next, nil
}

func (s *searchnode) accepts(all string) int {
	if s.all == "" {
		s.all = all
	}

	switch s.kind {
	case searchstatic:
		if s.final {
			return 0
		}

		l := len(s.str)
		if len(all) < l {
			return 0
		}

		if all[:l] == s.str {
			s.final = true
			s.idx = l
			return l
		}
		return 0

	case searchany:
		if s.next == nil {
			s.idx = len(s.all)
			s.final = true
			return s.idx
		}
		s.idx += 1
		s.final = true
		return 1

	case searchnumber:
		if s.next == nil {
			for _, b := range s.all {
				if b < '0' || b > '9' {
					return 0
				}
			}
			s.idx = len(s.all)
			s.final = true
			return s.idx
		}
		if all[0] < '0' || all[0] > '9' {
			return 0
		}
		s.idx += 1
		s.final = true
		return 1

	case searchsignednumber:
		if s.next == nil {
			for i, b := range s.all {
				if b < '0' || b > '9' {
					if i == 0 && b != '-' {
						return 0
					}
				}
			}
			s.idx = len(s.all)
			s.final = true
			return s.idx
		}
		if all[0] < '0' || all[0] > '9' {
			if s.idx == 0 && all[0] != '-' {
				return 0
			}
		}
		s.idx += 1
		s.final = true
		return 1

	case searchuuid:
		if s.final {
			return 0
		}

		if len(all) < 36 {
			return 0
		}

		if all[8] != '-' || all[13] != '-' || all[18] != '-' || all[23] != '-' {
			return 0
		}

		for i, b := range all[:36] {
			if i == 8 || i == 13 || i == 18 || i == 23 {
				continue
			}

			if (b < '0' || b > '9') && (b < 'a' || b > 'f') && (b < 'A' || b > 'F') {
				return 0
			}
		}
		s.idx = 36
		s.final = true
		return s.idx

	case searchuuidv4:
		if s.final {
			return 0
		}

		if len(all) < 36 {
			return 0
		}

		b := all[14]
		if b != '4' {
			return 0
		}

		b = all[19]
		if (b < '8' || b > '9') && (b < 'a' && b > 'b') && (b < 'A' && b > 'B') {
			return 0
		}

		if all[8] != '-' || all[13] != '-' || all[18] != '-' || all[23] != '-' {
			return 0
		}

		for i, b := range all[:36] {
			if i == 14 || i == 19 || i == 8 || i == 13 || i == 18 || i == 23 {
				continue
			}
			if (b < '0' || b > '9') && (b < 'a' || b > 'f') && (b < 'A' || b > 'F') {
				return 0
			}
		}
		s.idx = 36
		s.final = true
		return s.idx

	case searchid:
		if s.next == nil {
			for _, b := range s.all {
				if (b < '0' || b > '9') && (b < 'a' || b > 'f') && (b < 'A' || b > 'F') {
					return 0
				}
			}
			s.idx = len(s.all)
			s.final = true
			return s.idx
		}
		b := all[0]
		if (b < '0' || b > '9') && (b < 'a' || b > 'f') && (b < 'A' || b > 'F') {
			return 0
		}
		s.idx += 1
		s.final = true
		return 1
	}

	return 0
}

func (s *searchnode) reset() {
	s.final = false
	s.idx = 0
	s.all = ""
}

func (s searchnode) saveparam(parms *[]PathParam) {
	if s.kind != searchstatic {
		if parms != nil {
			*parms = append(*parms, PathParam{Key: s.paramname, Value: s.all[:s.idx]})
		}
	}
}

func (s *searchnode) search(input string, parms *[]PathParam) bool {
	nx := s.next

	if s.next == nil && s.kind == searchany && len(input) > 0 {
		if parms != nil {
			*parms = append(*parms, PathParam{Key: s.paramname, Value: input})
		}
		return true
	}

	i := 0
	for {
		if i == len(input) {
			if nx == nil && s.final {
				s.saveparam(parms)
				s.reset()
				return true
			} else {
				s.reset()
				return false
			}
		}

		if s.final && nx != nil {
			acc := nx.accepts(input[i:])
			if acc > 0 && nx.search(input[i+acc:], parms) {
				s.saveparam(parms)
				s.reset()
				return true
			}
		}
		if acc := s.accepts(input[i:]); acc > 0 {
			i += acc
			continue
		}
		s.reset()
		return false
	}
}

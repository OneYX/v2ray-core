package gfwlist

import (
	"bufio"
	"github.com/golang/protobuf/proto"
	"net"
	"regexp"
	"strings"
	"v2ray.com/core/app/log"
	"v2ray.com/core/common"
	"v2ray.com/core/common/errors"
)

type hostWildcardRule struct {
	pattern string
}

func (r *hostWildcardRule) match(domain string) bool {
	if strings.Contains(domain, r.pattern) {
		return true
	}
	return false
}

type urlWildcardRule struct {
	pattern     string
	prefixMatch bool
}

func (r *urlWildcardRule) match(domain string) bool {
	if r.prefixMatch {
		return strings.HasPrefix(domain, r.pattern)
	}
	return strings.Contains(domain, r.pattern)
}

type regexRule struct {
	pattern string
}

func (r *regexRule) match(domain string) bool {
	matched, err := regexp.MatchString(r.pattern, domain)
	if nil != err {
		log.Trace(errors.New("Invalid regex pattern:" + r.pattern + " with reason:" + err.Error()))
	}
	return matched
}

type whiteListRule struct {
	r gfwListRule
}

func (r *whiteListRule) match(domain string) bool {
	return r.r.match(domain)
}

type gfwListRule interface {
	match(domain string) bool
}

type GFWList struct {
	ruleMap  map[string]gfwListRule
	ruleList []gfwListRule
}

func (gfw *GFWList) FastMatchDomain(domain string) (bool, bool) {
	rootDomain := domain
	if strings.Contains(domain, ":") {
		domain, _, _ = net.SplitHostPort(domain)
		rootDomain = domain
	}

	rule, exist := gfw.ruleMap[domain]
	if !exist {
		ss := strings.Split(domain, ".")
		if len(ss) > 2 {
			rootDomain = ss[len(ss)-2] + "." + ss[len(ss)-1]
			if len(ss[len(ss)-2]) < 4 && len(ss) >= 3 {
				rootDomain = ss[len(ss)-3] + "." + rootDomain
			}
		}
		if rootDomain != domain {
			rule, exist = gfw.ruleMap[rootDomain]
		}
	}
	if exist {
		matched := rule.match(domain)
		if _, ok := rule.(*whiteListRule); ok {
			return !matched, true
		}
		return matched, true
	}
	return false, false
}

func (gfw *GFWList) IsBlockedByGFW(domain string) bool {

	fastMatchResult, exist := gfw.FastMatchDomain(domain)
	if exist {
		return fastMatchResult
	}

	for _, rule := range gfw.ruleList {
		if rule.match(domain) {
			if _, ok := rule.(*whiteListRule); ok {
				//log.Printf("#### %s is in whilte list %v", req.Host, rule.(*whiteListRule).r)
				return false
			}
			return true
		}
	}
	return false
}

func Parse(rules string) (*GFWList, error) {
	reader := bufio.NewReader(strings.NewReader(rules))
	gfw := new(GFWList)
	gfw.ruleMap = make(map[string]gfwListRule)
	//i := 0
	for {
		line, _, err := reader.ReadLine()
		if nil != err {
			break
		}
		str := strings.TrimSpace(string(line))
		//comment
		if strings.HasPrefix(str, "!") || len(str) == 0 || strings.HasPrefix(str, "[") {
			continue
		}
		var rule gfwListRule
		isWhileListRule := false
		fastMatch := false
		if strings.HasPrefix(str, "@@") {
			str = str[2:]
			isWhileListRule = true
		}
		if strings.HasPrefix(str, "/") && strings.HasSuffix(str, "/") {
			str = str[1 : len(str)-1]
			rule = &regexRule{str}
		} else {
			if strings.HasPrefix(str, "||") {
				fastMatch = true
				str = str[2:]
				rule = &hostWildcardRule{str}
			} else if strings.HasPrefix(str, "|") {
				rule = &urlWildcardRule{str[1:], true}
			} else {
				if !strings.Contains(str, "/") {
					fastMatch = true
					if strings.HasPrefix(str, ".") {
						str = str[1:]
					}
					rule = &hostWildcardRule{str}
				} else {
					rule = &urlWildcardRule{str, false}
				}
			}
		}
		if isWhileListRule {
			rule = &whiteListRule{rule}
		}
		if fastMatch {
			gfw.ruleMap[str] = rule
		} else {
			gfw.ruleList = append(gfw.ruleList, rule)
		}
	}
	return gfw, nil
}

func NewGFWList() *GFWList {
	var gfwlist Gfwlist
	err := proto.Unmarshal(GfwListData, &gfwlist)
	common.Must(err)
	gfwList, err := Parse(gfwlist.Content)
	common.Must(err)
	return gfwList
}

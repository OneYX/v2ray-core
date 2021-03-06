package router

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"v2ray.com/core/common/net"
	"v2ray.com/core/common/protocol"
	"v2ray.com/core/proxy"
	"v2ray.com/core/tools/gfwlist"
)

type Condition interface {
	Apply(ctx context.Context) bool
}

type ConditionChan []Condition

func NewConditionChan() *ConditionChan {
	var condChan ConditionChan = make([]Condition, 0, 8)
	return &condChan
}

func (v *ConditionChan) Add(cond Condition) *ConditionChan {
	*v = append(*v, cond)
	return v
}

func (v *ConditionChan) Apply(ctx context.Context) bool {
	for _, cond := range *v {
		if !cond.Apply(ctx) {
			return false
		}
	}
	return true
}

func (v *ConditionChan) Len() int {
	return len(*v)
}

type AnyCondition []Condition

func NewAnyCondition() *AnyCondition {
	var anyCond AnyCondition = make([]Condition, 0, 8)
	return &anyCond
}

func (v *AnyCondition) Add(cond Condition) *AnyCondition {
	*v = append(*v, cond)
	return v
}

func (v *AnyCondition) Apply(ctx context.Context) bool {
	for _, cond := range *v {
		if cond.Apply(ctx) {
			return true
		}
	}
	return false
}

func (v *AnyCondition) Len() int {
	return len(*v)
}

type PlainDomainMatcher string

func NewPlainDomainMatcher(pattern string) Condition {
	return PlainDomainMatcher(pattern)
}

func (v PlainDomainMatcher) Apply(ctx context.Context) bool {
	dest, ok := proxy.TargetFromContext(ctx)
	if !ok {
		return false
	}

	if !dest.Address.Family().IsDomain() {
		return false
	}
	domain := dest.Address.Domain()
	return strings.Contains(domain, string(v))
}

type RegexpDomainMatcher struct {
	pattern *regexp.Regexp
}

func NewRegexpDomainMatcher(pattern string) (*RegexpDomainMatcher, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexpDomainMatcher{
		pattern: r,
	}, nil
}

func (v *RegexpDomainMatcher) Apply(ctx context.Context) bool {
	dest, ok := proxy.TargetFromContext(ctx)
	if !ok {
		return false
	}
	if !dest.Address.Family().IsDomain() {
		return false
	}
	domain := dest.Address.Domain()
	return v.pattern.MatchString(strings.ToLower(domain))
}

type SubDomainMatcher string

func NewSubDomainMatcher(p string) Condition {
	return SubDomainMatcher(p)
}

func (m SubDomainMatcher) Apply(ctx context.Context) bool {
	dest, ok := proxy.TargetFromContext(ctx)
	if !ok {
		return false
	}
	if !dest.Address.Family().IsDomain() {
		return false
	}
	domain := dest.Address.Domain()
	pattern := string(m)
	if !strings.HasSuffix(domain, pattern) {
		return false
	}
	return len(domain) == len(pattern) || domain[len(domain)-len(pattern)-1] == '.'
}

type GfwMatcher struct {
	GFWList *gfwlist.GFWList
	cache   map[string]bool
	mu      sync.RWMutex
}

func NewGfwMatcher() Condition {
	return &GfwMatcher{GFWList: gfwlist.NewGFWList(), cache: make(map[string]bool)}
}

func (m *GfwMatcher) Apply(ctx context.Context) bool {
	dest, ok := proxy.TargetFromContext(ctx)
	if !ok {
		return false
	}
	if !dest.Address.Family().IsDomain() {
		return false
	}
	domain := dest.Address.Domain()
	result, ok := func() (result bool, found bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		v, ok := m.cache[domain]
		return v, ok
	}()
	if ok {
		return result
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if result, ok = m.cache[domain]; ok {
		return result
	}
	result = m.GFWList.IsBlockedByGFW(domain)
	m.cache[domain] = result
	return result
}

type CIDRMatcher struct {
	cidr     *net.IPNet
	onSource bool
}

func NewCIDRMatcher(ip []byte, mask uint32, onSource bool) (*CIDRMatcher, error) {
	cidr := &net.IPNet{
		IP:   net.IP(ip),
		Mask: net.CIDRMask(int(mask), len(ip)*8),
	}
	return &CIDRMatcher{
		cidr:     cidr,
		onSource: onSource,
	}, nil
}

func (v *CIDRMatcher) Apply(ctx context.Context) bool {
	ips := make([]net.IP, 0, 4)
	if resolveIPs, ok := proxy.ResolvedIPsFromContext(ctx); ok {
		for _, rip := range resolveIPs {
			if !rip.Family().IsIPv6() {
				continue
			}
			ips = append(ips, rip.IP())
		}
	}

	var dest net.Destination
	var ok bool
	if v.onSource {
		dest, ok = proxy.SourceFromContext(ctx)
	} else {
		dest, ok = proxy.TargetFromContext(ctx)
	}

	if ok && dest.Address.Family().IsIPv6() {
		ips = append(ips, dest.Address.IP())
	}

	for _, ip := range ips {
		if v.cidr.Contains(ip) {
			return true
		}
	}
	return false
}

type IPv4Matcher struct {
	ipv4net  *net.IPNetTable
	onSource bool
}

func NewIPv4Matcher(ipnet *net.IPNetTable, onSource bool) *IPv4Matcher {
	return &IPv4Matcher{
		ipv4net:  ipnet,
		onSource: onSource,
	}
}

func (v *IPv4Matcher) Apply(ctx context.Context) bool {
	ips := make([]net.IP, 0, 4)
	if resolveIPs, ok := proxy.ResolvedIPsFromContext(ctx); ok {
		for _, rip := range resolveIPs {
			if !rip.Family().IsIPv4() {
				continue
			}
			ips = append(ips, rip.IP())
		}
	}

	var dest net.Destination
	var ok bool
	if v.onSource {
		dest, ok = proxy.SourceFromContext(ctx)
	} else {
		dest, ok = proxy.TargetFromContext(ctx)
	}

	if ok && dest.Address.Family().IsIPv4() {
		ips = append(ips, dest.Address.IP())
	}

	for _, ip := range ips {
		if v.ipv4net.Contains(ip) {
			return true
		}
	}
	return false
}

type PortMatcher struct {
	port net.PortRange
}

func NewPortMatcher(portRange net.PortRange) *PortMatcher {
	return &PortMatcher{
		port: portRange,
	}
}

func (v *PortMatcher) Apply(ctx context.Context) bool {
	dest, ok := proxy.TargetFromContext(ctx)
	if !ok {
		return false
	}
	return v.port.Contains(dest.Port)
}

type NetworkMatcher struct {
	network *net.NetworkList
}

func NewNetworkMatcher(network *net.NetworkList) *NetworkMatcher {
	return &NetworkMatcher{
		network: network,
	}
}

func (v *NetworkMatcher) Apply(ctx context.Context) bool {
	dest, ok := proxy.TargetFromContext(ctx)
	if !ok {
		return false
	}
	return v.network.HasNetwork(dest.Network)
}

type UserMatcher struct {
	user []string
}

func NewUserMatcher(users []string) *UserMatcher {
	usersCopy := make([]string, 0, len(users))
	for _, user := range users {
		if len(user) > 0 {
			usersCopy = append(usersCopy, user)
		}
	}
	return &UserMatcher{
		user: usersCopy,
	}
}

func (v *UserMatcher) Apply(ctx context.Context) bool {
	user := protocol.UserFromContext(ctx)
	if user == nil {
		return false
	}
	for _, u := range v.user {
		if u == user.Email {
			return true
		}
	}
	return false
}

type InboundTagMatcher struct {
	tags []string
}

func NewInboundTagMatcher(tags []string) *InboundTagMatcher {
	tagsCopy := make([]string, 0, len(tags))
	for _, tag := range tags {
		if len(tag) > 0 {
			tagsCopy = append(tagsCopy, tag)
		}
	}
	return &InboundTagMatcher{
		tags: tagsCopy,
	}
}

func (v *InboundTagMatcher) Apply(ctx context.Context) bool {
	tag, ok := proxy.InboundTagFromContext(ctx)
	if !ok {
		return false
	}

	for _, t := range v.tags {
		if t == tag {
			return true
		}
	}
	return false
}

package gfwlist

import (
	"github.com/golang/protobuf/proto"
	"testing"
	. "v2ray.com/ext/assert"
)

func TestGfwListData(t *testing.T) {
	assert := With(t)
	var gfwlists Gfwlist
	err := proto.Unmarshal(GfwListData, &gfwlists)
	println(gfwlists.Content)
	assert(err, IsNil)
}

func TestGfwList(t *testing.T) {
	assert := With(t)
	gfwList := NewGFWList()
	assert(gfwList.IsBlockedByGFW("google.com"), IsTrue)
	assert(gfwList.IsBlockedByGFW("api.telegram.me"), IsTrue)
	assert(gfwList.IsBlockedByGFW("www.qq.com"), IsFalse)
	assert(gfwList.IsBlockedByGFW("id.heroku.com"), IsTrue)
}

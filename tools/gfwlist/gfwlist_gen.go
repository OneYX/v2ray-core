// +build generate

package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"v2ray.com/core/tools/gfwlist"

	"github.com/golang/protobuf/proto"
	"v2ray.com/core/common"
	"v2ray.com/core/common/errors"
)

//go:generate go run gfwlist_gen.go

const (
	gfwlistUrl = "https://gitlab.com/gfwlist/gfwlist/raw/master/gfwlist.txt"
)

func main() {
	resp, err := http.Get(gfwlistUrl)
	common.Must(err)
	if resp.StatusCode != 200 {
		panic(errors.New("unexpected status ", resp.StatusCode))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	common.Must(err)
	plainTxt, err := base64.StdEncoding.DecodeString(string(body))

	gfwlists := &gfwlist.Gfwlist{
		Content: string(plainTxt),
	}

	gfwListBytes, err := proto.Marshal(gfwlists)
	if err != nil {
		log.Fatalf("Failed to marshal gfwlists: %v", err)
	}

	file, err := os.OpenFile("gfwlist.generated.go", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Failed to generate gfwlist_data.go: %v", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "package gfwlist")

	fmt.Fprintln(file, "var GfwListData = "+formatArray(gfwListBytes))
}

func formatArray(a []byte) string {
	r := "[]byte{"
	for idx, val := range a {
		if idx > 0 {
			r += ","
		}
		r += fmt.Sprintf("%d", val)
	}
	r += "}"
	return r
}

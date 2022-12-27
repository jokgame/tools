package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

var flags = map[string]string{
	"CN": "https://cdn.countryflags.com/thumbs/china/flag-square-500.png",
	"IN": "https://cdn.countryflags.com/thumbs/india/flag-square-500.png",
	"EU": "https://cdn.countryflags.com/thumbs/europe/flag-square-500.png",
	"US": "https://cdn.countryflags.com/thumbs/united-states-of-america/flag-square-500.png",
	"ID": "https://cdn.countryflags.com/thumbs/indonesia/flag-square-500.png",
}

func main() {
	var countries = make(map[string]string)
	for _, page := range pages {
		parsePage(page, countries)
	}
	log.Printf("parsed %d codes: %v", len(countries), countries)

	var flags = parseFlags()
	log.Printf("parsed %d flags: %v", len(flags), flags)

	for name, url := range flags {
		code, found := countries[name]
		if found {
			download(url, code+".png")
		} else {
			log.Printf("FIXME: please rename %s.png", name)
			download(url, name+".png")
		}
	}
}

func parsePage(s string, countries map[string]string) {
	type Page struct {
		RPC [][]interface{} `json:"rpc"`
	}

	begin := strings.Index(s, "[")
	if begin <= 0 {
		panic("[ not found")
	}
	var page []Page
	json.Unmarshal([]byte(s[begin:]), &page)
	for _, v := range page[0].RPC[0][3].([]interface{})[1].([]interface{}) {
		m := v.(map[string]interface{})
		d := m["d"].(map[string]interface{})
		code := d["143"].(string)
		name := strings.ToLower(d["147"].(string))
		name = strings.TrimSpace(strings.TrimSuffix(name, "(the)"))
		name = strings.ReplaceAll(name, " ", "-")
		if _, dup := countries[name]; dup {
			panic("country " + name + " duplicated")
		}
		countries[name] = code
	}
}

func parseFlags() map[string]string {
	resp, err := http.Get("https://www.countryflags.com")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	root, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}
	var images []*html.Node
	for _, tiles := range searchNode(nil, root, func(node *html.Node) bool {
		return matchAttr(node, "class", "tiles")
	}) {
		images = searchNode(images, tiles, func(node *html.Node) bool {
			return node.Data == "img"
		})
	}
	var flags = make(map[string]string)
	for _, img := range images {
		url := getAttr(img, "src")
		name := strings.TrimPrefix(url, "https://cdn.countryflags.com/thumbs/")
		end := strings.Index(name, "/")
		if end < 0 {
			panic("unexpected url format: " + url)
		}
		flags[name[:end]] = url
	}
	return flags
}

func download(url string, filename string) {
	log.Printf("downloading %s to %s", url, filename)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("download %s error: %v", url, err)
		return
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("download %s error: %v", url, err)
		return
	}
	if err := os.WriteFile(filename, content, 0666); err != nil {
		log.Printf("download %s error: %v", url, err)
	}
}

func searchNode(dst []*html.Node, src *html.Node, matcher func(*html.Node) bool) []*html.Node {
	if matcher(src) {
		dst = append(dst, src)
	}
	for child := src.FirstChild; child != nil; child = child.NextSibling {
		dst = searchNode(dst, child, matcher)
	}
	return dst
}

func getAttr(n *html.Node, key string) string {
	val, _ := getOptionalAttr(n, key)
	return val
}

func getOptionalAttr(n *html.Node, key string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

func matchAttr(n *html.Node, attr, value string) bool {
	if n.Type == html.ElementNode {
		s, ok := getOptionalAttr(n, attr)
		if ok && s == value {
			return true
		}
	}
	return false
}

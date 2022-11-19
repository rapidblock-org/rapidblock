package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	yaml "gopkg.in/yaml.v3"
)

type yamlLiteralKey struct {
	tag   string
	value string
}

type yamlMapCacheItem struct {
	tagged  bool
	tag     string
	content []*yaml.Node
	node    *yaml.Node
}

var (
	gIntStrCache  = make(map[uint64]string, 64)
	gLiteralCache = make(map[yamlLiteralKey]*yaml.Node, 64)
	gStringCache  = make(map[string]*yaml.Node, 64)
	gAliasCache   = make(map[*yaml.Node]*yaml.Node, 64)
	gMapCache     = make([]yamlMapCacheItem, 0, 64)
	gLastAnchorID uint64
	gBuffer       bytes.Buffer
)

func yamlMakeBool(value bool) *yaml.Node {
	if value {
		return yamlMakeLiteral("!!bool", "true")
	}
	return yamlMakeLiteral("!!bool", "false")
}

func yamlMakeInt(value uint64) *yaml.Node {
	str, found := gIntStrCache[value]
	if !found {
		str = strconv.FormatUint(value, 10)
		gIntStrCache[value] = str
	}
	return yamlMakeLiteral("!!int", str)
}

func yamlMakeLiteral(tag string, value string) *yaml.Node {
	key := yamlLiteralKey{tag, value}
	node := gLiteralCache[key]
	if node == nil {
		node = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   tag,
			Value: value,
		}
		gLiteralCache[key] = node
	}
	return node
}

func yamlMakeString(value string) *yaml.Node {
	node := gStringCache[value]
	if node == nil {
		node = &yaml.Node{
			Kind: yaml.ScalarNode,
			Tag:  "!!str",
		}
		node.SetString(value)
		gStringCache[value] = node
	}
	return node
}

func yamlMakeMapCommon(tagged bool, tag string, content []*yaml.Node) *yaml.Node {
	for _, item := range gMapCache {
		if item.tagged == tagged && item.tag == tag && yamlSameContent(item.content, content) {
			return item.node
		}
	}
	style := yaml.Style(0)
	if tagged {
		style = yaml.TaggedStyle
	}
	node := &yaml.Node{
		Kind:    yaml.MappingNode,
		Style:   style,
		Tag:     tag,
		Content: content,
	}
	item := yamlMapCacheItem{tagged, tag, content, node}
	gMapCache = append(gMapCache, item)
	return node
}

func yamlMakeMap(content ...*yaml.Node) *yaml.Node {
	return yamlMakeMapCommon(false, "!!map", content)
}

func yamlMakeTaggedMap(tag string, content ...*yaml.Node) *yaml.Node {
	return yamlMakeMapCommon(true, tag, content)
}

func yamlMakeAlias(source *yaml.Node) *yaml.Node {
	node := gAliasCache[source]
	if node == nil {
		if source.Anchor == "" {
			gLastAnchorID++
			source.Anchor = strconv.FormatUint(gLastAnchorID, 10)
		}
		node = &yaml.Node{
			Kind:  yaml.AliasNode,
			Style: source.Style,
			Tag:   source.Tag,
			Alias: source,
			Value: source.Anchor,
		}
		gAliasCache[source] = node
	}
	return node
}

func yamlMakeDoc(child *yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{child},
	}
}

func yamlToString(doc *yaml.Node) string {
	gBuffer.Reset()
	gBuffer.WriteString("---\n")

	e := yaml.NewEncoder(&gBuffer)
	e.SetIndent(2)
	err := e.Encode(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to encode audit log data to YAML: %v\n", err)
		os.Exit(1)
	}

	err = e.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to encode audit log data to YAML: %v\n", err)
		os.Exit(1)
	}

	return gBuffer.String()
}

func yamlSameContent(a []*yaml.Node, b []*yaml.Node) bool {
	aLen := len(a)
	bLen := len(b)
	if aLen != bLen {
		return false
	}
	for i := 0; i < aLen; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

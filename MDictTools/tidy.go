package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var entry = flag.String("e", "", "工具启动入口")
var ruleFile = flag.String("c", "", "整理规则文件路径")
var showHelp = flag.Bool("h", false, "显示应用帮助信息并退出")

// Entry 词条内容位置
type Entry struct {
	start  int    `label:"开始坐标"`
	end    int    `label:"结束坐标"`
	word   string `label:"词头"`
	action string `label:"@@@动作名"`
	value  string `label:"@@@动作内容"`
}

// TagAttr 标签属性
type TagAttr struct {
	state        bool   `label:"属性状态"`
	lowerName    string `label:"小写属性名"`
	originalName string `label:"原始属性名"`
	value        string `label:"属性值"`
	quote        string `label:"标签值引号"`
}

// Tag 标签
type Tag struct {
	state    bool       `label:"标签状态"`
	hasAttr  bool       `label:"标签是否有属性"`
	dynamic  bool       `label:"属性是否动态修改"`
	id       int64      `label:"标签ID"`
	close    int64      `label:"结束标签ID"`
	parent   int64      `label:"上级标签ID"`
	category string     `label:"标签分类"`
	name     string     `label:"标签名"`
	value    string     `label:"标签内容"`
	attrs    []*TagAttr `label:"标签属性"`
}

func (t *Tag) String() string {
	if t.dynamic {
		var buf = bytes.NewBuffer(nil)

		buf.WriteString("<")
		buf.WriteString(t.name)

		for _, attr := range t.attrs {
			if attr.state {
				buf.WriteString(" ")
				buf.WriteString(attr.originalName)
				buf.WriteString("=")
				buf.WriteString(attr.quote)
				buf.WriteString(attr.value)
				buf.WriteString(attr.quote)
			}
		}

		if strings.HasSuffix(t.value, "/>") {
			if len(t.attrs) > 0 {
				buf.WriteString(" />")
			} else {
				buf.WriteString("/>")
			}
		} else {
			buf.WriteString(">")
		}

		return buf.String()
	}

	return t.value
}

// Drop 设置删除标记
func (t *Tag) Drop() {
	t.state = false
}

// Parse 解析标签属性
func (t *Tag) Parse() {
	if t.hasAttr && nil == t.attrs {
		var flag byte
		var pair []string
		var key, val string
		var pos, quoteN int
		var quote, isFirst, findIt bool
		var length = len(t.value)

		t.attrs = make([]*TagAttr, 0, 10)
		for k, v := range t.value {
			if ' ' == v || 2 == quoteN || k+1 == length || (k+2 == length && '/' == t.value[k+1]) {
				if !isFirst {
					isFirst = true

					continue
				} else if quote {
					continue
				} else if findIt {
					if (k+1 < length && (' ' == t.value[k+1] || '=' == t.value[k+1])) || (k > 0 && (' ' == t.value[k-1] || '=' == t.value[k-1])) {
						continue
					}

					if 2 == quoteN && flag != t.value[pos-1] {
						pair = strings.SplitN(t.value[pos-1:k], "=", 2)
					} else {
						pair = strings.SplitN(t.value[pos:k], "=", 2)
					}
					if 2 == len(pair) {
						key = strings.Trim(pair[0], "\r\n\t\"' ")
						val = strings.Trim(pair[1], "/> ")

						if "" != key && "" != val {
							t.attrs = append(t.attrs, &TagAttr{
								state:        true,
								lowerName:    strings.ToLower(key),
								originalName: key,
								value:        strings.Trim(val, val[0:1]+" "),
								quote:        val[0:1],
							})
						}
					}

					quoteN = 0
					findIt = false
				}
			} else if '"' == v {
				if !quote {
					flag = '"'
					quote = true
					if findIt {
						quoteN++
					}
				} else if quote && '"' == flag && k > 0 && '\\' != v {
					quote = false
					if findIt {
						quoteN++
					}
				}
			} else if '\'' == v {
				if !quote {
					quote = true
					flag = '\''
				} else if quote && '\'' == flag && k > 0 && '\\' != v {
					quote = false
				}
			} else if isFirst && !quote && !findIt && '=' != v {
				pos = k
				findIt = true
			}
		}
	}
}

// Get 返回属性值
func (t *Tag) Get(attr string) *TagAttr {
	t.Parse()

	for _, v := range t.attrs {
		if v.state && v.lowerName == attr {
			return v
		}
	}

	return nil
}

// Match 标签是否匹配选择器
func (t *Tag) Match(selector *TagSelector) bool {
	var val string
	var attr *TagAttr
	var values []string
	var result = true

	if t.hasAttr {
		if "class" == selector.Type {
			if "" != selector.Tag && t.name != selector.Tag {
				result = false
			} else {
				result = false

				if attr = t.Get(selector.Attr); nil != attr {
					values = strings.Split(attr.value, " ")
					for _, val = range values {
						for _, sel := range selector.Value {
							if sel == val {
								result = true

								break
							}
						}
						if result {
							break
						}
					}
				}
			}
		} else if "id" == selector.Type {
			if "" != selector.Tag && t.name != selector.Tag {
				result = false
			} else {
				if attr = t.Get("id"); nil != attr && attr.value == selector.Value[0] {
					result = true
				} else if attr = t.Get("name"); nil != attr && attr.value == selector.Value[0] {
					result = true
				} else {
					result = false
				}
			}
		} else if "attr" == selector.Type {
			if "" != selector.Tag && t.name != selector.Tag {
				result = false
			} else {
				if attr = t.Get(selector.Attr); nil == attr {
					result = false
				} else {
					if "" != selector.Value[0] {
						result = false
						if "*" == selector.Value[0] {
							result = true
						} else if '^' == selector.Value[0][0] && strings.HasPrefix(attr.value, selector.Value[0][1:]) {
							result = true
						} else if '$' == selector.Value[0][0] && strings.HasSuffix(attr.value, selector.Value[0][1:]) {
							result = true
						} else if '~' == selector.Value[0][0] && strings.Contains(attr.value, selector.Value[0][1:]) {
							result = true
						} else if attr.value == selector.Value[0] {
							result = true
						}
					}
				}
			}
		} else if "tag" == selector.Type {
			result = t.name == selector.Tag
		} else {
			result = false
		}
	} else {
		result = "tag" == selector.Type && t.name == selector.Tag
	}

	return result
}

// StripEvent 去掉事件
func (t *Tag) StripEvent() {
	if t.hasAttr && ("start" == t.category || "self" == t.category) {
		t.Parse()

		for _, attr := range t.attrs {
			if strings.HasPrefix(attr.lowerName, "on") {
				attr.state = false
				t.dynamic = true
			}
		}
	}
}

// StripEmpty 去掉空的属性
func (t *Tag) StripEmpty() {
	if t.hasAttr && ("start" == t.category || "self" == t.category) {
		t.Parse()

		for _, v := range t.attrs {
			if "" == v.value {
				v.state = false
				t.dynamic = true
			}
		}
	}
}

// StripAttr 去掉属性
func (t *Tag) StripAttr(names []string) {
	if t.hasAttr && ("start" == t.category || "self" == t.category) {
		t.Parse()

		for _, v := range t.attrs {
			for _, name := range names {
				if v.lowerName == name {
					v.state = false
					t.dynamic = true

					break
				}
			}
		}
	}
}

// TagSelector 标签选择器
type TagSelector struct {
	Type  string   `label:"类型"`
	Tag   string   `label:"标签"`
	Attr  string   `label:"属性名"`
	Value []string `label:"属性值"`
}

// TidyOption 清理参数
type TidyOption struct {
	DumpWord      bool           `label:"输出词头"`
	SkipEvent     bool           `label:"去除事件"`
	SkipEmptyAttr bool           `label:"去除空属性"`
	SkipComment   bool           `label:"去除注释"`
	EscapeBracket bool           `label:"转义括号"`
	Input         string         `label:"输入文件"`
	Style         string         `label:"Style文件"`
	Output        string         `label:"输出文件"`
	Drop          []string       `label:"删除的标签"`
	UnWrap        []string       `label:"解开的标签"`
	SkipContent   []string       `label:"跳过的内容"`
	Prepare       [][2]string    `label:"预替换的关键词"`
	Post          [][2]string    `label:"后替换的关键词"`
	selDrop       []*TagSelector `label:"删除的标签"`
	selUnWrap     []*TagSelector `label:"删除的标签"`
}

// Init 初始化
func (o *TidyOption) Init() error {
	var err error
	var msg = make([]string, 0, 10)

	if "" == o.Input {
		msg = append(msg, "输入文件属性 Input 不能为空")
	} else {
		var pos = strings.LastIndex(o.Input, ".")

		if "" == o.Output {
			o.Output = o.Input[:pos] + ".new." + o.Input[pos+1:]
		} else if o.Input == o.Output {
			msg = append(msg, "输入文件和输出文件不能相同")
		}
		if "" == o.Style {
			o.Style = o.Input[:pos] + ".Style." + o.Input[pos+1:]
			if _, err := os.Stat(o.Style); nil != err {
				o.Style = ""
			}
		}
	}

	if len(o.Drop) > 0 {
		o.selDrop = make([]*TagSelector, len(o.Drop))
		for k, v := range o.Drop {
			o.selDrop[k] = o.parseSelector(v)
		}
	}
	if len(o.UnWrap) > 0 {
		o.selUnWrap = make([]*TagSelector, len(o.UnWrap))
		for k, v := range o.UnWrap {
			o.selUnWrap[k] = o.parseSelector(v)
		}
	}

	if len(msg) > 0 {
		err = errors.New(strings.Join(msg, "\n"))
	}

	return err
}

// parseSelector 解析选择器
func (o *TidyOption) parseSelector(selector string) *TagSelector {
	var pos int
	var sel *TagSelector
	var pair, values []string

	if pos = strings.Index(selector, "."); -1 != pos {
		pair = strings.Split(selector, ".")
		sel = &TagSelector{
			Type:  "class",
			Tag:   pair[0],
			Attr:  "class",
			Value: pair[1:],
		}
	} else if pos = strings.Index(selector, "#"); -1 != pos {
		pair = strings.SplitN(selector, "#", 2)
		sel = &TagSelector{
			Type:  "id",
			Tag:   pair[0],
			Attr:  "id",
			Value: pair[1:],
		}
	} else if pos = strings.Index(selector, "["); -1 != pos {
		pair = strings.SplitN(selector, "[", 2)
		values = strings.SplitN(strings.Trim(pair[1], "] "), "=", 2)
		sel = &TagSelector{
			Type:  "attr",
			Tag:   pair[0],
			Attr:  strings.Trim(values[0], " "),
			Value: []string{strings.Trim(values[1], "\r\n\t\"' ")},
		}
	} else {
		sel = &TagSelector{
			Type: "tag",
			Tag:  selector,
		}
	}

	return sel
}

// CSSOption CSS 整理选项
type CSSOption struct {
	separator string   `label:"CSS换行分隔符"`
	Source    string   `label:"源文件路径"`
	CSS       string   `label:"CSS源文件路径"`
	Output    string   `label:"CSS保存文件路径"`
	Summary   string   `label:"源文件选择器概览"`
	SkipID    []string `label:"忽略的 CSS ID"`
	SkipClass []string `label:"忽略的 CSS CLASS"`
	SkipAttr  []string `label:"忽略的标签属性"`
}

// MergeOption 词典合并选项
type MergeOption struct {
	Source string `label:"源词典文件"`
	Target string `label:"合并到的词典文件"`
	Output string `label:"输出的词典文件"`
}

// Dom 文档标签树
type Dom struct {
	idx  int64   `label:"标签下标"`
	sub  []int64 `label:"子节点列表"`
	root []*Tag  `label:"标签列表"`
}

// GetSubIdx 返回搜索结果的节点号
func (d *Dom) GetSubIdx(idx int) int64 {
	if len(d.sub) > 0 && idx < len(d.sub) {
		return d.sub[idx]
	}

	return 0
}

// RangeToString DOM 区间节点转换为字符串
func (d *Dom) RangeToString(s int64, e int64, textOnly bool) string {
	var tag *Tag
	var buf = new(bytes.Buffer)

	for _, tag = range d.root {
		if tag.state && tag.id >= s && tag.id <= e {
			if !textOnly || (textOnly && "content" == tag.category) {
				buf.WriteString(tag.value)
			}
		}
	}

	return buf.String()
}

// ToString 将 DOM 树转换为字符串
func (d *Dom) ToString(textOnly bool) string {
	var tag *Tag
	var idx int64
	var sub map[int64]bool
	var buf = new(bytes.Buffer)

	if nil == d.sub {
		buf.WriteString(d.RangeToString(d.root[0].id, d.root[len(d.root)-1].id, textOnly))
	} else if len(d.sub) > 0 {
		sub = make(map[int64]bool, len(d.sub))
		for _, idx = range d.sub {
			sub[idx] = true
		}
		for _, tag = range d.root {
			if sub[tag.id] {
				buf.WriteString(d.RangeToString(tag.id, tag.close, textOnly))
			}
		}
	}

	return buf.String()
}

// Find 查找 DOM 子元素
func (d *Dom) Find(selector *TagSelector) *Dom {
	var tag *Tag
	var skip int64
	var sub = make([]int64, 0, len(d.root))

	for _, tag = range d.root {
		if skip > 0 && tag.id < skip {
			continue
		}
		if tag.state && ("start" == tag.category || "self" == tag.category) && tag.Match(selector) {
			skip = tag.close
			sub = append(sub, tag.id)
		}
	}

	return &Dom{idx: d.idx, root: d.root, sub: sub}
}

// Filter 过滤 DOM 子元素
func (d *Dom) Filter(text string) *Dom {
	var tag, parent *Tag
	var sub = make([]int64, 0, len(d.root))
	var need = make(map[int64]bool, len(d.sub))
	var subMap = make(map[int64]bool, len(d.sub))

	for _, v := range d.sub {
		need[v] = true
	}
	for _, tag = range d.root {
		if need[tag.id] {
			parent = tag
		} else if nil != parent && !subMap[parent.id] {
			if parent.close == tag.id {
				parent = nil
			} else if tag.id < parent.close {
				if -1 != strings.Index(tag.value, text) {
					sub = append(sub, parent.id)
					subMap[parent.id] = true
				}
			}
		}
	}

	return &Dom{idx: d.idx, root: d.root, sub: sub}
}

// Insert 插入元素
func (d *Dom) Insert(value string, idx int64, after bool) {
	var closeIdx int64
	var pos, it int
	var tag *Tag
	var nTag = &Tag{
		state:    true,
		name:     "none",
		value:    value,
		category: "raw",
	}

	for it, tag = range d.root {
		if idx == tag.id {
			if after {
				if "self" == tag.category {
					pos = it

					break
				} else {
					closeIdx = tag.close
				}
			} else {
				pos = it

				break
			}
		} else if after && closeIdx == tag.id {
			pos = it

			break
		}
	}

	if pos > 0 {
		nTag.parent = d.root[pos].parent
		if after {
			nTag.id = d.root[pos].id + 1
		} else {
			nTag.id = d.root[pos].id - 1
		}

		d.idx++
		d.root = append(d.root, nTag)

		sort.Slice(d.root, func(i int, j int) bool {
			return d.root[i].id < d.root[j].id
		})
	}
}

// Tidy 整理标签树
//
// 实现的功能
//
//	1、清理掉不是 mdx 源文件需要的标签
//	2、清理掉不正常关闭的标签
//	3、清理掉空的内容
//	4、根据选项开关清理注释
//	5、关闭未关闭的标签
//	6、根据规则清理标签
//
// 实现思路：
//
//	1、读取一个词条内容
//	2、将其解析为最小单元：开始标签、内容、结束标签、注释、自关闭标签
//	3、如果是开始标签，就将标签名入栈
//	4、如果是内容或注释且不为空，就将其入队列
//	5、如果是自关闭标签，就检查是否要清理的，如果是就直接丢弃，否则入队列
//	6、如果是关闭标签，先检查是否要清理的标签，如果是就开始反向清理内容直到开始标签，再将开始标签出栈
//	   如果不是要清理的标签，就与最后的开始标签比对，如果匹配是入队列并将开始标签出栈，不匹配就直接丢弃
//	7、结束后检查标签栈，如果不为空，就自动在后面补标签以结束内容
func (d *Dom) Tidy(entry *Entry, opt *TidyOption) {
	var tag *Tag
	var num, idx int
	var dropClose int64
	var lastTag, skip string
	var tags = make([]*Tag, 0, 300)
	var unWrapID = make(map[int64]bool, len(opt.UnWrap))
	var skipClose = map[string]bool{"a": true, "img": true, "hr": true, "br": true, "tr": true, "td": true, "th": true, "thead": true, "tbody": true, "link": true}

	for idx, tag = range d.root {
		if dropClose > 0 {
			if tag.id == dropClose {
				dropClose = 0
			}

			tag.Drop()

			continue
		}
		if ("start" == tag.category || "self" == tag.category) && "" != tag.name {
			for _, r := range opt.selDrop {
				if tag.Match(r) {
					tag.Drop()
					if tag.close > 0 {
						dropClose = tag.close
					}

					break
				}
			}
			if dropClose > 0 {
				continue
			}
			for _, r := range opt.selUnWrap {
				if tag.Match(r) {
					tag.Drop()
					if tag.close > 0 {
						unWrapID[tag.close] = true
					}

					break
				}
			}
		}

		if unWrapID[tag.id] {
			tag.Drop()
		}
		if !tag.state {
			continue
		}
		if "html" == tag.name || "head" == tag.name || "body" == tag.name || "!doctype" == tag.name {
			tag.Drop()
		} else if "content" == tag.category {
			if 0 == idx {
				continue
			}
			if "script" == lastTag || "style" == lastTag || "pre" == lastTag {
				tag.value = strings.Trim(tag.value, "\r\n\t ")
			} else {
				for _, skip = range opt.SkipContent {
					if -1 != strings.Index(tag.value, skip) {
						tag.Drop()

						if num = len(tags); num > 0 && idx+1 < len(d.root) && tags[num-1].id == d.root[idx-1].id && d.root[idx-1].close == d.root[idx+1].id {
							tags = tags[:num-1]
							d.root[idx-1].Drop()
							d.root[idx+1].Drop()
						}

						break
					}
				}
				if !tag.state {
					continue
				}

				var hasFirstSpace = false
				var hasEndSpace = false
				var lastIdx = len(tag.value) - 1

				if '\r' == tag.value[0] || '\n' == tag.value[0] || '\t' == tag.value[0] || ' ' == tag.value[0] {
					hasFirstSpace = true
				}
				if '\r' == tag.value[lastIdx] || '\n' == tag.value[lastIdx] || '\t' == tag.value[lastIdx] || ' ' == tag.value[lastIdx] {
					hasEndSpace = true
				}

				tag.value = stripSpaceMore(tag.value)
				if "" == tag.value {
					tag.Drop()

					continue
				}

				if hasFirstSpace {
					tag.value = " " + tag.value
				}
				if hasEndSpace {
					tag.value = tag.value + " "
				}
				if opt.EscapeBracket {
					tag.value = strings.ReplaceAll(tag.value, "<", "&lt")
					tag.value = strings.ReplaceAll(tag.value, ">", "&gt")
				}
			}
		} else if "comment" == tag.category {
			if opt.SkipComment {
				tag.Drop()
			}
		} else if "start" == tag.category {
			lastTag = tag.name
			tags = append(tags, tag)
		} else if "close" == tag.category {
			num = len(tags)
			if num > 0 {
				lastTag = tags[num-1].name
			} else {
				lastTag = ""
			}
			if lastTag == tag.name {
				tags = tags[:num-1]
			} else {
				tag.Drop()
			}
		}
		if opt.SkipEvent {
			tag.StripEvent()
		}
		if opt.SkipEmptyAttr {
			tag.StripEmpty()
		}
	}

	for {
		if num = len(tags); num > 0 {
			tag = tags[num-1]
			tags = tags[:num-1]
			if skipClose[tag.name] {
				continue
			}

			d.idx++
			d.root = append(d.root, &Tag{state: true, id: d.idx*10000 + 5000, parent: tag.parent, category: "close", name: tag.name, value: "</" + tag.name + ">"})
		} else {
			break
		}
	}

	sort.Slice(d.root, func(i int, j int) bool {
		return d.root[i].id < d.root[j].id
	})
}

// GetDirFiles 获取指定文件夹文件列表
func GetDirFiles(dirPath string, stripExt bool, recursion bool, suffixes ...string) []string {
	var idx int
	var flag bool
	var file string
	var ret []string

	if files, err := os.ReadDir(dirPath); nil == err && nil != files {
		ret = make([]string, 0, len(files))
		for _, v := range files {
			if v.IsDir() && recursion {
				if items := GetDirFiles(dirPath+"/"+v.Name(), stripExt, recursion, suffixes...); nil == err && len(items) > 0 {
					ret = append(ret, items...)
				}
			} else {
				flag = false
				file = v.Name()

				if len(suffixes) == 0 {
					flag = true
				} else {
					for _, suffix := range suffixes {
						if strings.HasSuffix(file, suffix) {
							flag = true
							break
						}
					}
				}
				if flag {
					if stripExt {
						if idx = strings.LastIndexByte(v.Name(), '.'); idx > 0 {
							ret = append(ret, dirPath+"/"+strings.Trim(file[0:idx], " "))
						} else {
							ret = append(ret, dirPath+"/"+v.Name())
						}
					} else {
						ret = append(ret, dirPath+"/"+v.Name())
					}
				}
			}
		}
	}

	return ret
}

// FilePutContents Put bytes to file.
// if non-exist, create this file.
func FilePutContents(filename string, content []byte, append bool) error {
	var flag int
	if append {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	fp, err := os.OpenFile(filename, flag, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		_ = fp.Close()
	}()

	_, err = fp.Write(content)
	return err
}

// LoadJSON 从文件加载 JSON 到变量
func LoadJSON(file string, in interface{}) error {
	var err error
	var data []byte

	if data, err = os.ReadFile(file); nil == err {
		err = json.Unmarshal(data, in)
	}

	return err
}

// findChar 从指定位置找字符
func findChar(data string, need string, start int, end int) int {
	var char byte
	var chars = make(map[byte]bool, len(need))

	for _, char = range []byte(need) {
		chars[char] = true
	}
	for i := start; i < end; i++ {
		char = data[i]

		if chars[char] {
			return i
		}
	}

	return -1
}

// 去除多余的空白符
func stripSpace(data string) string {
	data = strings.ReplaceAll(data, " ", " ")
	data = strings.ReplaceAll(data, "\t", " ")
	data = strings.ReplaceAll(data, " &nbsp; ", " ")

	for {
		if -1 == strings.Index(data, "  ") {
			break
		}

		data = strings.ReplaceAll(data, "  ", " ")
	}

	return strings.Trim(data, "\r\n\t ")
}

// 去除多余的空白符
func stripSpaceMore(data string) string {
	data = strings.ReplaceAll(data, "\n", " ")
	data = strings.ReplaceAll(data, "\r", " ")

	return stripSpace(data)
}

// 预解析词条
func parseBody(data []byte, start int, end int) *Entry {
	var entry *Entry
	var pair []string
	var idx, last int
	var flag, word bool

	for idx = start; idx < end; idx++ {
		if '\r' == data[idx] || '\n' == data[idx] {
			if flag && !word {
				last = idx
				word = true
				entry = &Entry{
					word:  strings.Trim(string(data[start:idx]), "\r\n\t\"`, "),
					start: start,
					end:   end,
				}
			} else if flag && word && idx-last >= 10 {
				break
			}
		} else if ' ' == data[idx] {
			continue
		} else if flag && '@' == data[idx] && idx+2 < end && '@' == data[idx+1] && '@' == data[idx+2] {
			pair = strings.SplitN(string(data[idx+3:end]), "=", 2)
			if 2 == len(pair) {
				entry.action = strings.Trim(pair[0], "\r\n\t ")
				entry.value = strings.Trim(pair[1], "\r\n\t ")

				break
			}
		} else {
			flag = true
		}
	}

	return entry
}

// splitMdxData 拆分词典内容为词条坐标
func splitMdxData(data []byte) []*Entry {
	var idx, pos int
	var dataLen = len(data)
	var entries = make([]*Entry, 0, 100000)

	for idx = 0; idx < dataLen; idx++ {
		if idx+3 < dataLen && '<' == data[idx] && '/' == data[idx+1] && '>' == data[idx+2] {
			if (idx > 0 && '\r' != data[idx-1] && '\n' != data[idx-1]) || (idx+4 < dataLen && '\r' != data[idx+3] && '\n' != data[idx+3]) {
				continue
			}
			if idx > 0 {
				entries = append(entries, parseBody(data, pos, idx-1))
			}

			pos = idx + 5
		} else if idx+3 == dataLen && pos+3 < dataLen {
			if '<' == data[dataLen-3] && '/' == data[dataLen-2] && '>' == data[dataLen-1] {
				entries = append(entries, parseBody(data, pos, dataLen-3))
			} else {
				entries = append(entries, parseBody(data, pos, dataLen))
			}

			break
		}
	}

	return stripBlockHoleEntry(entries)
}

// stripBlockHoleEntry 去除无效的词链接
func stripBlockHoleEntry(in []*Entry) []*Entry {
	var ok bool
	var miss = 0
	var out = make([]*Entry, 0, len(in))
	var link = make(map[string]string, 100)
	var mapper = make(map[string]bool, len(in))

	for _, v := range in {
		if "link" == strings.ToLower(v.action) {
			link[v.word] = v.value
		} else {
			mapper[v.word] = true
		}
	}

	for {
		miss = 0
		for k, v := range link {
			if _, ok = link[v]; !ok && !mapper[v] {
				delete(link, k)
				miss++
			}
		}
		if miss == 0 {
			break
		}
	}

	for _, v := range in {
		if len(v.word) > 1024 {
			fmt.Println("long word:", v.word)
		}
		if "link" == strings.ToLower(v.action) {
			if _, ok = link[v.word]; ok {
				out = append(out, v)
			} else {
				//fmt.Println(v.word)
			}
		} else {
			out = append(out, v)
		}
	}

	return out
}

// 解析词条内容为标签
func parseBodyItem(element *Entry, data string) *Dom {
	var length = len(data)
	var tagStack = make([]*Tag, 0, 100)
	var container = make([]*Tag, 0, 1000)
	var checkOkPos = make(map[int]bool, 100)
	var tagRegex = regexp.MustCompile(`^[a-zA-Z]+[0-9]*\s*$`)
	var cur, parent int64
	var hitStart, hitEnd, isComment, isScriptOrStyle bool
	var pos, idx, num, endPos, startPos, lastStartPos, lastPos, commentPos int

	for idx = 0; idx < length; idx++ {
		if '<' == data[idx] {
			if isScriptOrStyle && ((' ' == data[idx-1] && ' ' == data[idx+1]) || (idx+1 < length && '/' != data[idx+1])) {
				continue
			}
			if idx+2 < length && '!' == data[idx+1] && '-' == data[idx+2] {
				isComment = true
				commentPos = idx
			}
			if idx+1 < length && '!' != data[idx+1] && (' ' == data[idx+1] || '<' == data[idx+1] || data[idx+1] < 47 || data[idx+1] > 122) {
				continue
			}
			if idx+2 < length && '/' == data[idx+1] && '>' == data[idx+2] {
				continue
			}
			if isComment && idx > commentPos {
				continue
			}
			if idx+1 < length && '/' != data[idx+1] && '!' != data[idx+1] && (data[idx+1] < 65 || data[idx+1] > 122) {
				continue
			}

			lastStartPos = startPos
			startPos = idx
			hitStart = true
		}
		if '>' == data[idx] && (hitStart || isComment || isScriptOrStyle) {
			if isScriptOrStyle && ' ' == data[idx-1] && ' ' == data[idx+1] {
				continue
			}
			if isComment && '-' != data[idx-1] {
				continue
			}
			if !isComment && idx > 0 && ' ' != data[idx-1] && '/' != data[idx-1] && '\'' != data[idx-1] && '"' != data[idx-1] && (data[idx-1] < 47 || data[idx-1] > 122) {
				continue
			}

			hitEnd = true
			endPos = idx
		}
		if hitStart && !hitEnd && !isComment && !isScriptOrStyle && startPos+1 < idx {
			if idx-startPos > 15 {
				if !checkOkPos[startPos] {
					pos = findChar(data, "\r\n\t ", startPos, idx)
					if pos > 0 {
						if !tagRegex.MatchString(data[startPos+1 : pos]) {
							startPos = lastStartPos
							hitStart = false
						} else {
							checkOkPos[startPos] = true
						}
					} else if !tagRegex.MatchString(data[startPos+1 : idx]) {
						startPos = lastStartPos
						hitStart = false
					} else {
						checkOkPos[startPos] = true
					}
				}
			} else if data[idx] > 127 {
				startPos = lastStartPos
				hitStart = false
			}
		}
		if hitStart && hitEnd && endPos > startPos {
			var tag *Tag

			cur++
			if isComment {
				tag = &Tag{
					state:    true,
					category: "comment",
					value:    data[lastPos : idx+1],
				}
			} else if '/' == data[startPos+1] {
				tag = &Tag{
					state:    true,
					category: "close",
					value:    stripSpaceMore(data[startPos : endPos+1]),
					name:     strings.ToLower(data[startPos+2 : endPos]),
				}

				if isScriptOrStyle && ("script" == tag.name || "style" == tag.name) {
					isScriptOrStyle = false
				}
			} else if '/' == data[endPos-1] {
				tag = &Tag{
					state:    true,
					category: "self",
					value:    stripSpaceMore(data[startPos : endPos+1]),
				}

				pos = findChar(data, "\r\n\t ", startPos, endPos)
				if pos > 0 && pos < endPos {
					tag.name = data[startPos+1 : pos]
				} else {
					tag.name = data[startPos+1 : endPos-1]
				}
			} else {
				tag = &Tag{
					state:    true,
					category: "start",
					value:    data[startPos : endPos+1],
				}

				pos = findChar(data, "\r\n\t ", startPos, endPos)
				if pos > 0 && pos < endPos {
					tag.name = strings.Trim(strings.ToLower(data[startPos+1:pos]), "\r\n\t ")
				} else {
					tag.name = strings.Trim(strings.ToLower(data[startPos+1:endPos]), "\r\n\t ")
				}
				if "" == tag.name {
					tag.category = "content"
				} else if "meta" == tag.name || "param" == tag.name || "hr" == tag.name || "br" == tag.name || "img" == tag.name || "input" == tag.name || "source" == tag.name || "link" == tag.name {
					tag.category = "self"
				} else if "start" == tag.category && ("script" == tag.name || "style" == tag.name) {
					isScriptOrStyle = true
				}
				if "start" == tag.category || "self" == tag.category {
					tag.value = stripSpaceMore(tag.value)
					tag.hasAttr = strings.Index(tag.value, "=") > 0
				}
			}

			tag.id = cur*10000 + 5000
			tag.parent = parent
			if "start" == tag.category {
				parent = tag.id
				tagStack = append(tagStack, tag)
			} else if "close" == tag.category {
				num = len(tagStack)
				if 0 == num {
					parent = 0
				} else if 1 == num {
					if tagStack[0].name == tag.name {
						tagStack[0].close = tag.id
					}

					parent = 0
					tagStack = make([]*Tag, 0, 100)
				} else if num > 1 && tagStack[num-1].name == tag.name {
					tagStack[num-1].close = tag.id
					parent = tagStack[num-2].id
					tagStack = tagStack[:num-1]
				}

				tag.parent = parent
			}

			hitEnd = false
			hitStart = false
			isComment = false
			lastPos = idx + 1
			container = append(container, tag)
		} else if hitStart && startPos-lastPos > 0 {
			cur++

			var tag = &Tag{
				state:    true,
				id:       cur*10000 + 5000,
				parent:   parent,
				category: "content",
				value:    data[lastPos:idx],
			}

			lastPos = idx
			container = append(container, tag)
		} else if idx+1 == length && len(container) > 0 {
			if hitStart {
				fmt.Println("entry ["+element.word+"] has invalid tag:", data[lastPos:], ", byte index:", idx)
			} else {
				cur++

				container = append(container, &Tag{
					state:    true,
					id:       cur*10000 + 5000,
					parent:   parent,
					category: "content",
					value:    data[lastPos:],
				})
			}
		}
	}

	if 0 == len(container) {
		container = append(container, &Tag{
			state:    true,
			id:       1,
			category: "raw",
			value:    data,
		})
	} else {
		if "content" == container[0].category {
			if -1 != strings.Index(container[0].value, "&") {
				container[0].value = html.UnescapeString(container[0].value)
			}
			if -1 != strings.Index(container[0].value, "%") {
				if v, err := url.QueryUnescape(container[0].value); nil == err {
					container[0].value = v
				}
			}

			container[0].value = strings.Trim(container[0].value, "\r\n\t\"`', ") + "\r\n"
		}
	}

	return &Dom{idx: cur, root: container}
}

// prepareStyle 预处理样式
func prepareStyle(body []byte, style *map[string][2]string) string {
	var ok bool
	var num, pos int
	var styleID, styleEnd string
	var buf = new(bytes.Buffer)

	for k, v := range body {
		if '`' == v {
			num++
			if 1 == num {
				buf.Write(body[:k])
			} else if 1 == num%2 {
				buf.Write(body[pos+1 : k])

				if "" != styleEnd {
					buf.WriteString(styleEnd)
				}
			} else {
				styleID = strings.Trim(string(body[pos+1:k]), "\r\n\t ")
				if _, ok = (*style)[styleID]; ok {
					buf.WriteString((*style)[styleID][0])
					styleEnd = (*style)[styleID][1]
				}
			}

			pos = k
		}
	}

	if pos+1 != len(body) {
		buf.Write(body[pos+1:])
	}

	buf.WriteString(styleEnd)

	return buf.String()
}

// tidyMdict 词典源文件内容整理
//
// 实现的功能
//
//	1、替换掉指定的内容
//	2、清理不需要的标签
//	3、清理不正确关闭的标签
//	4、自动关闭未关闭的标签
//
// 实现思路：
//
//	1、读取词典源文件
//	2、按配置预替换掉关键词内容
//	3、拆分词典源文件内容为词条
//	4、整理词典内容
//	5、将整理后的词典内容拼为源文件
//	6、按配置替换掉关键词内容
func tidyMdict(cfg string) error {
	var idx int
	var err error
	var dom *Dom
	var element *Entry
	var rawStyle [][]byte
	var data []byte
	var elements []*Entry
	var container []string
	var style map[string][2]string
	var word, body, newBody, content string

	var opt = new(TidyOption)
	if err = LoadJSON(cfg, opt); nil != err {
		return errors.New("加载配置文件 " + cfg + " 失败，" + err.Error())
	}
	if err = opt.Init(); nil != err {
		return errors.New("检查配置文件 " + cfg + " 失败，" + err.Error())
	}

	fmt.Println("read file before")
	if "" != opt.Style {
		if data, err = os.ReadFile(opt.Style); nil == err {
			style = make(map[string][2]string, 25)
			rawStyle = bytes.Split(data, []byte{'\n'})
			for k, v := range rawStyle {
				if 0 == (k+1)%3 {
					word = string(bytes.Trim(rawStyle[k-2], "\r\n\t "))
					style[word] = [2]string{
						string(bytes.Trim(rawStyle[k-1], "\r\n\t ")),
						string(bytes.Trim(v, "\r\n\t ")),
					}
				}
			}
		} else {
			return err
		}
	}

	if data, err = os.ReadFile(opt.Input); nil != err {
		return err
	}

	if 0xef == data[0] && 0xbb == data[1] && 0xbf == data[2] {
		data = data[3:]
	}

	fmt.Println("read file done")

	if len(opt.Prepare) > 0 {
		fmt.Println("prepare file start")
		for _, item := range opt.Prepare {
			data = bytes.ReplaceAll(data, []byte(item[0]), []byte(item[1]))
		}

		fmt.Println("prepare file done")
	}

	fmt.Println("split words")
	elements = splitMdxData(data)
	container = make([]string, 0, len(elements))
	fmt.Println("start data process")

	for _, element = range elements {
		idx++

		if opt.DumpWord {
			fmt.Println(word)
			continue
		}

		if body = stripSpace(string(data[element.start:element.end])); len(body) < 1 {
			continue
		}

		if nil == style {
			dom = parseBodyItem(element, body)
		} else {
			dom = parseBodyItem(element, prepareStyle([]byte(body), &style))
		}

		dom.Tidy(element, opt)
		newBody = dom.ToString(false)
		if float64(len(body))*1.3 < float64(len(newBody)) {
			fmt.Println("entry [" + element.word + "] parse failed, may be body incorrect")
		}
		if "" == newBody {
			continue
		}

		container = append(container, newBody)
		if idx > 0 && (0 == idx%50000 || idx+1 == len(elements)) {
			fmt.Println("start processed:", idx)
		}
	}

	content = strings.Join(container, "\r\n</>\r\n")
	if len(opt.Post) > 0 {
		fmt.Println("post process start")
		for _, item := range opt.Post {
			content = strings.ReplaceAll(content, item[0], item[1])
		}
		fmt.Println("post process done")
	}

	return FilePutContents(opt.Output, []byte(content), false)
}

// getSourceUsage 返回源文件中用到的标签属性
func getSourceUsage(opt *CSSOption) (map[string]map[string]int, error) {
	var tag *Tag
	var pair []string
	var ok, findIt, hasSpace bool
	var pos, spacePos, length int
	var data, err = os.ReadFile(opt.Source)
	var skipAttr = make(map[string]bool, len(opt.SkipAttr))
	var ret = map[string]map[string]int{
		"id":    make(map[string]int, 100),
		"tag":   make(map[string]int, 100),
		"class": make(map[string]int, 100),
	}

	if nil == err {
		length = len(data)
		for _, v := range opt.SkipAttr {
			skipAttr[v] = true
		}

		for k, v := range data {
			if '<' == v {
				if k+1 < length && ('/' == data[k+1] || '!' == data[k+1]) {
					continue
				}

				pos = k
				findIt = true
			} else if '>' == v && findIt && hasSpace && spacePos > pos+1 {
				findIt = false
				hasSpace = false
				tag = &Tag{name: strings.ToLower(string(data[pos+1 : spacePos])), value: string(data[pos : k+1])}
				tag.name = string(data[pos+1 : spacePos])

				tag.Parse()
				if _, ok = ret["tag"][tag.name]; ok {
					ret["tag"][tag.name]++
				} else {
					ret["tag"][tag.name] = 1
				}
				for _, va := range tag.attrs {
					if skipAttr[va.originalName] {
						continue
					}
					if _, ok = ret[va.originalName]; !ok {
						ret[va.originalName] = make(map[string]int)
					}

					if "class" == va.originalName {
						pair = strings.Split(va.value, " ")
						for _, ca := range pair {
							if _, ok = ret[va.originalName][ca]; ok {
								ret[va.originalName][ca]++
							} else {
								ret[va.originalName][ca] = 1
							}
						}
					} else {
						if _, ok = ret[va.originalName][va.value]; ok {
							ret[va.originalName][va.value]++
						} else {
							ret[va.originalName][va.value] = 1
						}
					}
				}
			} else if '>' == v && findIt && !hasSpace && '/' != data[pos-1] {
				findIt = false

				if '/' == data[k-1] {
					tag = &Tag{name: strings.ToLower(string(data[pos+1 : k-1]))}
				} else {
					tag = &Tag{name: strings.ToLower(string(data[pos+1 : k]))}
				}

				if _, ok = ret["tag"][tag.name]; ok {
					ret["tag"][tag.name]++
				} else {
					ret["tag"][tag.name] = 1
				}
			} else if ' ' == v && findIt && !hasSpace {
				spacePos = k
				hasSpace = true
			}
		}
	} else {
		err = errors.New("解析词典源文件败，" + err.Error())
	}

	return ret, err
}

// getCSSUsage 返回样式 CSS 资源
func getCSSUsage(opt *CSSOption) ([][3]string, error) {
	var findIt bool
	var key, val string
	var pos, fPos, nCr, nLn int
	var selector = make([][3]string, 0, 1000)
	var data, err = os.ReadFile(opt.CSS)

	if nil == err {
		for k, v := range data {
			if '{' == v && !findIt && k > pos {
				findIt = true
				key = string(bytes.Trim(data[pos:k], "\r\n\t "))
				pos = k + 1
				if fPos = strings.Index(key, ";"); fPos > 0 {
					selector = append(selector, [3]string{"cmd", key[:fPos+1], ""})
					key = strings.Trim(key[fPos+1:], "\r\n\t ")
				} else if fPos = strings.LastIndex(key, "*/"); fPos > 0 {
					if '/' == key[0] {
						selector = append(selector, [3]string{"remark", key[:fPos+2], ""})
						key = strings.Trim(key[fPos+2:], "\r\n\t ")
					} else {
						var remark bool
						var content = make([]byte, 0, len(key))
						for k1, v1 := range []byte(key) {
							if '/' == v1 {
								if !remark && k1+1 < len(key) && '*' == key[k1+1] {
									remark = true
								} else if remark && k1 > 0 && '*' == key[k1-1] {
									remark = false
								}

								continue
							}
							if remark {
								continue
							}

							content = append(content, v1)
						}

						key = strings.Trim(string(content), "\r\n\t ")
					}
				}
			} else if '}' == v && k > pos {
				findIt = false
				val = string(bytes.TrimRight(bytes.TrimLeft(data[pos:k-1], "\r\n"), "\r\n\t "))
				if "" != key && "" != val {
					selector = append(selector, [3]string{"css", key, val})
				}

				pos = k + 1
			} else if '\r' == v {
				nCr++
			} else if '\n' == v {
				nLn++
			}
		}
		if nCr > 0 && nLn > 0 {
			opt.separator = "\r\n"
		} else if nLn > 0 {
			opt.separator = "\n"
		} else {
			opt.separator = "\r"
		}
	} else {
		err = errors.New("解析样式表，" + err.Error())
	}

	return selector, err
}

// tidyCSS 整理词典 CSS 文件
//
// 实现的功能：
//
//	1、清理未被使用的 CSS 样式
//	2、生成词典源文件标签概览
//
// 实现思路：
//
//	1、提取 CSS 文件中的选择器与样式内容
//	2、提取词典源文件中标签与标签属性为标签概览
//	3、循环检查 CSS 文件中的选择器是否出现在词典标签概览中
//	4、设置检查标志，如果出现就标志为找到，否则就是未找到
//	5、将找到的样式保存到新的样式文件中
//	6、将标签概览保存到概览文件中，方便复查
func tidyCSS(cfg string) error {
	var err error
	var data []byte
	var findIt bool
	var selector [][3]string
	var tag, selT, sel4, sel5 string
	var skipID, skipClass map[string]bool
	var cssInUse map[string]map[string]int
	var cssContent, pair, sel1, sel2, sel3 []string
	var opt = new(CSSOption)

	if err = LoadJSON(cfg, opt); nil != err {
		err = errors.New("加载配置文件 " + cfg + " 失败，" + err.Error())
	}
	if "" == opt.Source {
		err = errors.New("词典源文件属性 Source 不能为空")
	} else {
		if _, err = os.Stat(opt.Source); nil != err {
			err = errors.New("词典源文件 " + opt.Source + " 不存在")
		}
	}
	if n := len(opt.CSS); 0 == n {
		err = errors.New("词典样式文件属性 CSS 不能为空")
	} else {
		if _, err = os.Stat(opt.CSS); nil != err {
			err = errors.New("词典样式文件 " + opt.CSS + " 不存在")
		}
		if "" == opt.Output {
			if n > 4 {
				opt.Output = opt.CSS[0:n-4] + ".new.css"
			} else {
				err = errors.New("词典输出样式属性 Output 不能为空")
			}
		}
		if "" == opt.Summary && n > 4 {
			opt.Summary = opt.CSS[0:n-4] + ".summary.json"
		}
	}
	if 0 == len(opt.SkipAttr) {
		opt.SkipAttr = []string{"style", "src", "href", "width", "height", "align", "border", "title", "alt"}
	}

	if selector, err = getCSSUsage(opt); nil == err {
		if len(selector) > 1 {
			cssContent = make([]string, 0, len(selector))

			skipID = make(map[string]bool, len(opt.SkipID))
			for _, sel4 = range opt.SkipID {
				skipID[sel4] = true
			}

			skipClass = make(map[string]bool, len(opt.SkipClass))
			for _, sel4 = range opt.SkipClass {
				skipClass[sel4] = true
			}

			if cssInUse, err = getSourceUsage(opt); nil == err {
				for _, css := range selector {
					if "css" == css[0] {
						findIt = false
						sel1 = strings.Split(css[1], ",")

						for _, sel4 = range sel1 {
							selT = ""
							sel2 = strings.Split(strings.Trim(sel4, "\r\n\t "), " ")

							for _, sel5 = range sel2 {
								if "" == sel5 {
									continue
								}

								pair = strings.SplitN(sel5, ":", 2)
								if sel3 = strings.Split(pair[0], "."); len(sel3) > 1 {
									if skipClass[sel3[1]] {
										continue
									}

									selT = "class"
								} else if sel3 = strings.SplitN(pair[0], "#", 2); 2 == len(sel3) {
									if skipID[sel3[1]] {
										continue
									}

									selT = "id"
								} else if sel3 = strings.SplitN(pair[0], "[", 2); 2 == len(sel3) {
									selT = "attr"
								} else {
									selT = "tag"
								}

								break
							}

							if "class" == selT {
								if "" == sel3[0] {
									if cssInUse["class"][sel3[1]] > 0 {
										findIt = true
									}
								} else {
									tag = strings.ToLower(sel3[0])
									if cssInUse["tag"][tag] > 0 && cssInUse["class"][sel3[1]] > 0 {
										findIt = true
									}
								}
							} else if "id" == selT {
								if "" == sel3[0] {
									if cssInUse["id"][sel3[1]] > 0 {
										findIt = true
									}
								} else {
									tag = strings.ToLower(sel3[0])
									if cssInUse["tag"][tag] > 0 && cssInUse["id"][sel3[1]] > 0 {
										findIt = true
									}
								}
							} else if "attr" == selT {
								if "" == sel3[0] {
									findIt = true
								} else {
									tag = strings.ToLower(sel3[0])
									if cssInUse["tag"][tag] > 0 && cssInUse["id"][sel3[1]] > 0 {
										findIt = true
									}
								}
							} else if "tag" == selT {
								tag = strings.ToLower(sel3[0])
								if cssInUse["tag"][tag] > 0 {
									findIt = true
								}
							}

							if findIt {
								cssContent = append(cssContent, css[1]+" {"+opt.separator+css[2]+opt.separator+"}")

								break
							}
						}
					} else {
						cssContent = append(cssContent, css[1])
					}
				}

				if err = FilePutContents(opt.Output, []byte(strings.Join(cssContent, opt.separator+opt.separator)), false); nil == err && "" != opt.Summary {
					if data, err = json.Marshal(cssInUse); nil == err {
						err = FilePutContents(opt.Summary, data, false)
					}
				}
			}
		}
	}

	return err
}

// mergeDict 合并词典
func mergeDict(cfg string) error {
	var ok bool
	var idx int
	var err error
	var exIdx int64
	var element *Entry
	var sDom, tDom *Dom
	var sData, tData []byte
	var targetContainer []string
	var body, content, origin string
	var targetEntriesMap map[string][]int
	var sourceEntries, targetEntries []*Entry
	var opt = new(MergeOption)
	var tagSelOrigin = &TagSelector{Type: "class", Tag: "div", Attr: "class", Value: []string{"origin"}}
	var tagSelEx = &TagSelector{Type: "class", Tag: "div", Attr: "class", Value: []string{"example"}}

	if err = LoadJSON(cfg, opt); nil != err {
		err = errors.New("加载配置文件 " + cfg + " 失败，" + err.Error())
	}
	if "" == opt.Source {
		err = errors.New("词典源文件属性 Source 不能为空")
	} else {
		if _, err = os.Stat(opt.Source); nil != err {
			err = errors.New("词典源文件 " + opt.Source + " 不存在")
		}
	}
	if "" == opt.Target {
		err = errors.New("词典目标文件属性 Target 不能为空")
	} else {
		if _, err = os.Stat(opt.Target); nil != err {
			err = errors.New("词典目标文件 " + opt.Target + " 不存在")
		}
	}
	if "" == opt.Output {
		opt.Output = opt.Target[0:len(opt.Target)-4] + ".new.txt"
	}

	if sData, err = os.ReadFile(opt.Source); nil != err {
		return err
	}
	if tData, err = os.ReadFile(opt.Target); nil != err {
		return err
	}

	sourceEntries = splitMdxData(sData)
	targetEntries = splitMdxData(tData)
	targetContainer = make([]string, len(targetEntries))
	targetEntriesMap = make(map[string][]int, len(targetEntries))

	for k, v := range targetEntries {
		v.word = strings.ToLower(v.word)
		if _, ok = targetEntriesMap[v.word]; !ok {
			targetEntriesMap[v.word] = make([]int, 0, 5)
		}

		targetContainer[k] = stripSpace(string(tData[v.start:v.end]))
		targetEntriesMap[v.word] = append(targetEntriesMap[v.word], k)
	}
	for _, element = range sourceEntries {
		body = ""
		if "" != element.action {
			continue
		}

		element.word = strings.ToLower(element.word)
		if v, ok := targetEntriesMap[element.word]; ok {
			for _, k1 := range v {
				if k1 < len(targetEntries) {
					if "" == targetEntries[k1].action {
						idx = k1
						body = targetContainer[k1]

						break
					}
				} else {
					idx = k1
					body = targetContainer[k1]

					break
				}
			}
			if "" != body {
				sDom = parseBodyItem(element, stripSpace(string(sData[element.start:element.end])))
				origin = sDom.Find(tagSelOrigin).ToString(false)
				tDom = parseBodyItem(element, body)
				exIdx = tDom.Find(tagSelEx).GetSubIdx(0)

				if "" != origin && exIdx > 0 {
					tDom.Insert(origin, exIdx, true)

					targetContainer[idx] = tDom.ToString(false)
				}
			}
		} else {
			targetContainer = append(targetContainer, stripSpace(string(sData[element.start:element.end])))
			targetEntriesMap[element.word] = []int{len(targetContainer) - 1}
		}
	}

	content = strings.Join(targetContainer, "\r\n</>\r\n")

	return FilePutContents(opt.Output, []byte(content), false)
}

// CmdEntry 命令入口
func CmdEntry(entry string, cfg string) {
	var err error
	var ts = time.Now().Unix()

	if "" == cfg {
		fmt.Println("整理规则文件不能为空")
		return
	}

	if _, err = os.Stat(cfg); nil != err {
		fmt.Println("整理规则文件不存在")
		return
	}

	switch entry {
	case "tidy":
		err = tidyMdict(cfg)
	case "css":
		err = tidyCSS(cfg)
	case "merge":
		err = mergeDict(cfg)
	default:
		err = errors.New("不支持的命令")
	}

	if nil == err {
		fmt.Println("process done, use time", time.Now().Unix()-ts, err)
	} else {
		fmt.Println("process failed,", err)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "使用说明: tidy -e <启动入口> -c <配置文件>")
		fmt.Fprintln(os.Stderr, "欢迎使用 MDict 词典源文件整理工具")
		fmt.Fprintln(os.Stderr, "更多信息：https://github.com/csg2008/tools/tree/master/MDictTools")
		fmt.Fprintln(os.Stderr, "")

		fmt.Fprintln(os.Stderr, "参数选项:")
		flag.PrintDefaults()

		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "启动入口：")
		fmt.Fprintln(os.Stderr, "        tidy     词典源文件整理")
		fmt.Fprintln(os.Stderr, "        css      词典引用的 CSS 整理")
		fmt.Fprintln(os.Stderr, "        merge    合并两本词典")
	}

	flag.Parse()
	if *showHelp || "" == *entry {
		flag.Usage()
		return
	}

	CmdEntry(*entry, *ruleFile)
}

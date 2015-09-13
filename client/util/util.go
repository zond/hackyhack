package util

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/davecgh/go-spew/spew"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

type AttentionLevel func(m interfaces.MCP, event *messages.Event) bool

var (
	AttentionLevelAll = func(m interfaces.MCP, event *messages.Event) bool {
		return true
	}
	AttentionLevelNotContainer = func(m interfaces.MCP, event *messages.Event) bool {
		container, err := GetContainer(m, m.GetResource())
		if err != nil {
			log.Fatal(err)
		}
		return container != event.Request.Resource
	}
	AttentionLevelMe = func(m interfaces.MCP, event *messages.Event) bool {
		return event.Request.Resource == m.GetResource()
	}
	AttentionLevelNone = func(m interfaces.MCP, event *messages.Event) bool {
		return false
	}
)

type AttentionLevels map[string]AttentionLevel

func (als *AttentionLevels) AddMethod(method string, at AttentionLevel) *AttentionLevels {
	(*als)[Sprintf("%v.%v", messages.EventTypeRequest, method)] = at
	return als
}

func (als AttentionLevels) Ignored(m interfaces.MCP, event *messages.Event) bool {
	key := ""
	if event.Type == messages.EventTypeRequest {
		key = fmt.Sprintf("%v.%v", event.Type, event.Request.Method)
	} else {
		key = fmt.Sprintf("%v.-", event.Type, event.Request.Method)
	}
	al, found := als[key]
	if !found {
		log.Printf("%+v not found", event)
		return false
	}
	return !al(m, event)
}

var DefaultAttentionLevels = (&AttentionLevels{}).
	AddMethod(messages.MethodGetShortDesc, AttentionLevelNone).
	AddMethod(messages.MethodGetContent, AttentionLevelNotContainer).
	AddMethod(messages.MethodGetLongDesc, AttentionLevelMe)

type sdCache struct {
	m map[string]*messages.ShortDesc
	l sync.RWMutex
}

func newSDCache() *sdCache {
	return &sdCache{
		m: map[string]*messages.ShortDesc{},
	}
}

func (c *sdCache) get(k string) (v *messages.ShortDesc, f bool) {
	c.l.RLock()
	defer c.l.RUnlock()
	v, f = c.m[k]
	return
}

func (c *sdCache) set(k string, v *messages.ShortDesc) {
	c.l.Lock()
	defer c.l.Unlock()
	c.m[k] = v
	go func() {
		time.Sleep(time.Second)
		c.l.Lock()
		defer c.l.Unlock()
		delete(c.m, k)
	}()
}

var cache = newSDCache()

func IsNoSuchMethod(err *messages.Error) bool {
	if err == nil {
		return false
	}
	return err.Code == messages.ErrorCodeNoSuchMethod
}

var LookInto = &messages.Verb{
	SecondPerson: "look into",
	ThirdPerson:  "looks into",
}

var LookAt = &messages.Verb{
	SecondPerson: "look at",
	ThirdPerson:  "looks at",
}

var GlanceAt = &messages.Verb{
	SecondPerson: "glance at",
	ThirdPerson:  "glances at",
}

var LookAround = &messages.Verb{
	SecondPerson: "look around",
	ThirdPerson:  "looks around",
	Intransitive: true,
}

var LookUp = &messages.Verb{
	SecondPerson: "look up",
	ThirdPerson:  "looks up",
	Intransitive: true,
}

var Inspect = &messages.Verb{
	SecondPerson: "inspect",
	ThirdPerson:  "inspects",
}

func Subscribe(m interfaces.MCP, sub *messages.Subscription) *messages.Error {
	var merr *messages.Error
	if err := m.Call(nil, m.GetResource(), messages.MethodSubscribe, []interface{}{sub}, &[]interface{}{&merr}); err != nil {
		return err
	}
	return merr
}

func GetContainer(m interfaces.MCP, resource string) (string, *messages.Error) {
	var container string
	var merr *messages.Error
	if err := m.Call(LookUp, resource, messages.MethodGetContainer, nil, &[]interface{}{&container, &merr}); err != nil {
		return "", err
	}
	return container, merr
}

func GetContent(m interfaces.MCP, resource string) ([]string, *messages.Error) {
	container, err := GetContainer(m, m.GetResource())
	if err != nil {
		return nil, err
	}
	verb := LookInto
	if resource == container {
		verb = LookAround
	}
	var content []string
	var merr *messages.Error
	if err := m.Call(verb, resource, messages.MethodGetContent, nil, &[]interface{}{&content, &merr}); err != nil {
		return nil, err
	}
	return content, merr
}

func GetLongDesc(m interfaces.MCP, resource string) (string, *messages.Error) {
	var desc string
	var merr *messages.Error
	if err := m.Call(LookAt, resource, messages.MethodGetLongDesc, nil, &[]interface{}{&desc, &merr}); err != nil {
		return "", err
	}
	return desc, merr
}

func GetShortDesc(m interfaces.MCP, resource string) (*messages.ShortDesc, *messages.Error) {
	desc, found := cache.get(resource)
	if found {
		return desc, nil
	}
	var merr *messages.Error
	if err := m.Call(GlanceAt, resource, messages.MethodGetShortDesc, nil, &[]interface{}{&desc, &merr}); err != nil {
		return nil, err
	}
	if merr != nil {
		return nil, merr
	}
	cache.set(resource, desc)
	return desc, nil
}

func GetShortDescs(m interfaces.MCP, resources []string) (messages.ShortDescs, *messages.Error) {
	result := make(messages.ShortDescs, len(resources))
	for index, resource := range resources {
		shortDesc, err := GetShortDesc(m, resource)
		if err != nil {
			return nil, err
		}
		result[index] = shortDesc
	}
	return result, nil
}

func SendToClient(m interfaces.MCP, msg string) {
	var merr *messages.Error
	if err := m.Call(nil, m.GetResource(), messages.MethodSendToClient, []string{msg}, &[]interface{}{&merr}); err != nil {
		log.Fatal(err)
	}
	if merr != nil {
		log.Fatal(merr)
	}
}

func Fatal(i ...interface{}) {
	log.Fatal(i...)
}

func Fatalf(f string, i ...interface{}) {
	log.Fatalf(f, i...)
}

func Log(i ...interface{}) {
	log.Print(i...)
}

func Logf(f string, i ...interface{}) {
	log.Printf(f, i...)
}

func Spewf(f string, i ...interface{}) string {
	return spew.Sprintf(f, i...)
}

func Sprintf(f string, i ...interface{}) string {
	return fmt.Sprintf(f, i...)
}

var whitespaceReg = regexp.MustCompile("\\s+")

func SplitWhitespace(s string) []string {
	return whitespaceReg.Split(s, -1)
}

func Capitalize(s string) string {
	return strings.ToUpper(string([]rune(s)[0:1])) + s[1:]
}

func GetContentShortDescMap(m interfaces.MCP, resource string) (map[string]string, *messages.Error) {
	result := map[string]string{}

	content, err := GetContent(m, resource)
	if err != nil {
		if IsNoSuchMethod(err) {
			err = nil
		} else {
			return nil, err
		}
	}
	descs, err := GetShortDescs(m, content)
	if err != nil {
		return nil, err
	}
	for index, resource := range content {
		result[resource] = descs[index].Value
	}

	return result, nil
}

func GetShortDescMap(m interfaces.MCP, resource string) (map[string]string, *messages.Error) {
	result, err := GetContentShortDescMap(m, resource)
	if err != nil {
		return nil, err
	}

	container, err := GetContainer(m, resource)
	if err != nil {
		return nil, err
	}

	containerShortDesc, err := GetShortDesc(m, container)
	if err != nil {
		return nil, err
	}
	result[container] = containerShortDesc.Value

	containerShortDescMap, err := GetContentShortDescMap(m, container)
	if err != nil {
		return nil, err
	}
	for resource, desc := range containerShortDescMap {
		result[resource] = desc
	}

	result["me"] = resource

	return result, nil
}

func Identify(m interfaces.MCP, what string) (mathes []string, err *messages.Error) {
	what = strings.ToLower(what)

	shortDescMap, err := GetShortDescMap(m, m.GetResource())
	if err != nil {
		return nil, err
	}

	// Exact match ("take rock")
	matches := []string{}
	for resource, desc := range shortDescMap {
		if strings.HasPrefix(strings.ToLower(desc), what) {
			matches = append(matches, resource)
		}
	}

	// Number suffix ("take rock 2")
	if len(matches) != 1 {
		found, num, prefix := SplitEndNumber(what)
		if found && num > 0 {
			newMatches := []string{}
			for resource, desc := range shortDescMap {
				if strings.HasPrefix(strings.ToLower(desc), prefix) {
					newMatches = append(newMatches, resource)
				}
			}
			if len(newMatches) >= num {
				matches = []string{newMatches[num-1]}
			}
		}
	}

	// Inside match ("take [large] rock")
	if len(matches) == 0 {
		newMatches := []string{}
		for resource, desc := range shortDescMap {
			parts := SplitWhitespace(desc)
			for _, part := range parts {
				if strings.HasPrefix(strings.ToLower(part), what) {
					newMatches = append(newMatches, resource)
				}
			}
			if len(newMatches) > 0 {
				matches = newMatches
			}
		}
	}

	// Inside number suffix ("take [large] rock 2")
	if len(matches) != 1 {
		found, num, prefix := SplitEndNumber(what)
		if found && num > 0 {
			newMatches := []string{}
			for resource, desc := range shortDescMap {
				parts := SplitWhitespace(desc)
				for _, part := range parts {
					if strings.HasPrefix(strings.ToLower(part), prefix) {
						newMatches = append(newMatches, resource)
					}
				}
				if len(newMatches) >= num {
					matches = []string{newMatches[num-1]}
				}
			}
		}
	}

	return matches, nil
}

const (
	splitStateNone = iota
	splitStateVerb
	splitStateNum
	splitStateWhite
	splitStateRest
)

func Reverse(s string) string {
	runes := make([]rune, len(s))
	for n, r := range s {
		runes[n] = r
	}
	for i := 0; i < len(s)/2; i++ {
		runes[i], runes[len(s)-1-i] = runes[len(s)-1-i], runes[i]
	}
	return string(runes)
}

func SplitEndNumber(source string) (found bool, num int, prefix string) {
	rev := Reverse(source)

	state := splitStateNone
	numBuf := &bytes.Buffer{}
	prefixBuf := &bytes.Buffer{}
	for _, r := range rev {
		switch state {
		case splitStateNone:
			if unicode.IsDigit(r) {
				io.WriteString(numBuf, string([]rune{r}))
			} else if unicode.IsSpace(r) {
				state = splitStateWhite
			} else {
				return false, 0, source
			}
		case splitStateWhite:
			if !unicode.IsSpace(r) {
				state = splitStateRest
				io.WriteString(prefixBuf, string([]rune{r}))
			}
		case splitStateRest:
			io.WriteString(prefixBuf, string([]rune{r}))
		}
	}

	numStr := Reverse(numBuf.String())
	prefix = Reverse(prefixBuf.String())

	var err error
	if num, err = strconv.Atoi(numStr); err != nil {
		return false, 0, source
	}

	return true, num, prefix
}

func SplitVerb(s string) (verb, rest string) {
	state := splitStateVerb
	verbBuf := &bytes.Buffer{}
	restBuf := &bytes.Buffer{}
	for _, r := range s {
		switch state {
		case splitStateVerb:
			if unicode.IsSpace(r) {
				state = splitStateWhite
			} else {
				io.WriteString(verbBuf, string([]rune{r}))
			}
		case splitStateWhite:
			if !unicode.IsSpace(r) {
				state = splitStateRest
				io.WriteString(restBuf, string([]rune{r}))
			}
		case splitStateRest:
			io.WriteString(restBuf, string([]rune{r}))
		}
	}
	verb = verbBuf.String()
	rest = restBuf.String()
	return
}

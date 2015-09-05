package util

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/gedex/inflector"
	"github.com/zond/hackyhack/lang"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

func IsNoSuchMethod(err *messages.Error) bool {
	if err == nil {
		return false
	}
	return err.Code == messages.ErrorCodeNoSuchMethod
}

func Enumerate(item interface{}) string {
	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Slice {
		descs := map[string]int{}
		for i := 0; i < val.Len(); i++ {
			desc := fmt.Sprint(val.Index(i))
			descs[desc] = descs[desc] + 1
		}
		result := []string{}
		for desc, count := range descs {
			if count == 1 {
				result = append(result, fmt.Sprintf("%v %v", lang.Art(desc), desc))
			} else {
				result = append(result, fmt.Sprintf("%v %v", count, inflector.Pluralize(desc)))
			}
		}
		buf := &bytes.Buffer{}
		for i := 0; i < len(result); i++ {
			fmt.Fprint(buf, result[i])
			if i < len(result)-2 {
				fmt.Fprint(buf, ", ")
			} else if i < len(result)-1 {
				fmt.Fprint(buf, ", and ")
			}
		}
		return buf.String()
	}
	return fmt.Sprintf("%v %v", lang.Art(fmt.Sprint(item)), item)
}

func GetContainer(m interfaces.MCP, resource string) (string, *messages.Error) {
	var container string
	var merr *messages.Error
	if err := m.Call(resource, messages.MethodGetContainer, nil, &[]interface{}{&container, &merr}); err != nil {
		return "", err
	}
	return container, merr
}

func GetContent(m interfaces.MCP, resource string) ([]string, *messages.Error) {
	var content []string
	var merr *messages.Error
	if err := m.Call(resource, messages.MethodGetContent, nil, &[]interface{}{&content, &merr}); err != nil {
		return nil, err
	}
	return content, merr
}

func GetLongDesc(m interfaces.MCP, resource string) (string, *messages.Error) {
	var desc string
	var merr *messages.Error
	if err := m.Call(resource, messages.MethodGetLongDesc, nil, &[]interface{}{&desc, &merr}); err != nil {
		return "", err
	}
	return desc, merr
}

func GetShortDesc(m interfaces.MCP, resource string) (string, *messages.Error) {
	var desc string
	var merr *messages.Error
	if err := m.Call(resource, messages.MethodGetShortDesc, nil, &[]interface{}{&desc, &merr}); err != nil {
		return "", err
	}
	return desc, merr
}

func GetShortDescs(m interfaces.MCP, resources []string) ([]string, *messages.Error) {
	result := make([]string, len(resources))
	for index, resource := range resources {
		shortDesc, err := GetShortDesc(m, resource)
		if err != nil {
			return nil, err
		}
		result[index] = shortDesc
	}
	return result, nil
}

func SendToClient(m interfaces.MCP, msg string) *messages.Error {
	var merr *messages.Error
	if err := m.Call(m.GetResource(), messages.MethodSendToClient, []string{msg}, &[]interface{}{&merr}); err != nil {
		return err
	}
	return merr
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

func ShortDescMap(m interfaces.MCP) (map[string]string, *messages.Error) {
	result := map[string]string{}
	result[m.GetResource()] = "me"

	return result, nil
}

func Identify(m interfaces.MCP, what string) (resource string, found bool, err *messages.Error) {
	if what == "me" {
		return m.GetResource(), true, nil
	}
	return "", false, nil
}

const (
	splitStateVerb = iota
	splitStateWhite
	splitStateRest
)

func SplitVerb(s string) (verb, rest string) {
	state := splitStateVerb
	verbBuf := &bytes.Buffer{}
	restBuf := &bytes.Buffer{}
	for _, r := range []rune(s) {
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

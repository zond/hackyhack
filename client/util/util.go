package util

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/gedex/inflector"
	"github.com/zond/hackyhack/lang"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

func Is(item interface{}) string {
	if val := reflect.ValueOf(item); val.Kind() == reflect.Slice && val.Len() > 1 {
		return "are"
	}
	return "is"
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

func Success(m interfaces.MCP, err *messages.Error) bool {
	if err == nil {
		return true
	}
	if err.Code == messages.ErrorCodeNoSuchMethod {
		return false
	}
	if err = SendToClient(m, fmt.Sprintf("%v: %v\n", err.Message, err.Code)); err != nil {
		log.Fatal(err)
	}
	return false
}

func fatality(m interfaces.MCP, err *messages.Error) *messages.Error {
	if err == nil {
		return nil
	}
	if err = SendToClient(m, fmt.Sprintf("%v: %v\n", err.Message, err.Code)); err != nil {
		log.Fatal(err)
	}
	return err
}

func GetContainer(m interfaces.MCP) (string, *messages.Error) {
	var container string
	var merr *messages.Error
	if err := m.Call(m.GetResource(), messages.MethodGetContainer, nil, &[]interface{}{&container, &merr}); err != nil {
		return "", fatality(m, err)
	}
	return container, fatality(m, merr)
}

func GetContent(m interfaces.MCP) ([]string, *messages.Error) {
	var content []string
	var merr *messages.Error
	if err := m.Call(m.GetResource(), messages.MethodGetContent, nil, &[]interface{}{&content, &merr}); err != nil {
		return nil, fatality(m, err)
	}
	return content, fatality(m, merr)
}

func GetContentDescs(m interfaces.MCP) ([]string, *messages.Error) {
	content, err := GetContent(m)
	if err != nil {
		return nil, fatality(m, err)
	}
	result := []string{}
	for _, item := range content {
		var desc string
		var merr *messages.Error
		if err := m.Call(item, messages.MethodGetShortDesc, []interface{}{m.GetResource()}, &[]interface{}{&desc, &merr}); err != nil {
			return nil, fatality(m, err)
		}
		if merr != nil {
			return nil, fatality(m, merr)
		}
		result = append(result, desc)
	}
	return result, nil
}

func SendToClient(m interfaces.MCP, msg string) *messages.Error {
	var merr *messages.Error
	if err := m.Call(m.GetResource(), messages.MethodSendToClient, []string{msg}, &[]interface{}{&merr}); err != nil {
		return fatality(m, err)
	}
	return fatality(m, merr)
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

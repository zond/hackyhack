package util

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

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
	if err := m.Call(m.GetResource(), messages.MethodSendToClient, nil, &[]interface{}{&content, &merr}); err != nil {
		return nil, fatality(m, err)
	}
	return content, fatality(m, merr)
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

package template

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"git.topfreegames.com/topfreegames/marathon/messages"

	. "github.com/franela/goblin"
)

func TrimSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}

func buildMessage(valMap map[string]string) string {
	msg := fmt.Sprintf(`{
    "app": "%v",
    "token": "%v",
    "type": "%v",
    "push_expiry": %v`, valMap["app"], valMap["token"], valMap["type"],
		valMap["push_expiry"])
	if valMap["template"] != "" {
		msg = msg + fmt.Sprintf(`,
    "template": "%v"`, valMap["template"])
	}
	if valMap["params"] != "" {
		msg = msg + fmt.Sprintf(`,
    "params": %v`, valMap["params"])
	}
	if valMap["locale"] != "" {
		msg = msg + fmt.Sprintf(`,
    "locale": "%v"`, valMap["locale"])
	}
	if valMap["message"] != "" {
		msg = msg + fmt.Sprintf(`,
    "message": "%v"`, valMap["message"])
	}
	if valMap["metadata"] != "" {
		msg = msg + fmt.Sprintf(`,
    "metadata": %v`, valMap["metadata"])
	}
	msg = msg + "\n}"
	return msg
}

func compareMapRequestMessage(valMap map[string]string, msg messages.RequestMessage) bool {
	if msg.App != valMap["app"] || msg.Token != valMap["token"] {
		fmt.Println("app or token different")
		return false
	}
	if msg.Type != valMap["type"] {
		fmt.Println("type different")
		return false
	}
	if strconv.FormatInt(msg.PushExpiry, 10) != valMap["push_expiry"] {
		fmt.Println("push_expiry different")
		return false
	}
	if msg.Template != valMap["template"] || msg.Message != valMap["message"] {
		fmt.Println("template or message different")
		return false
	}
	// Not testing params comparison
	return true
}

func TestTemplate(t *testing.T) {
	g := Goblin(t)

	g.Describe("Parser.Parse", func() {
		g.It("Should parse a templated message correctly", func() {
			valMap := map[string]string{
				"app":         "app_name",
				"token":       "token_id",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
				"template":    "template_name",
				"params":      `{"param1": "value1"}`,
				"locale":      "en",
				"metadata":    `{"meta": "data"}`,
			}

			message := buildMessage(valMap)
			msg, err := Parse(message)
			g.Assert(err == nil).IsTrue()
			g.Assert(compareMapRequestMessage(valMap, *msg)).IsTrue()
		})

		g.It("Should parse a plain text message correctly", func() {
			valMap := map[string]string{
				"app":         "app_name",
				"token":       "token_id",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
				"message":     "push message",
			}

			message := buildMessage(valMap)
			msg, err := Parse(message)
			g.Assert(err == nil).IsTrue()
			g.Assert(compareMapRequestMessage(valMap, *msg)).IsTrue()
		})

		g.It("Should not parse an invalid templated message", func() {
			valMap := map[string]string{
				"app":         "app_name",
				"token":       "token_id",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
				"template":    "template_name",
			}

			message := buildMessage(valMap)
			_, err := Parse(message)
			g.Assert(err != nil).IsTrue()
		})

		g.It("Should not parse a [plain text + templated] message", func() {
			valMap := map[string]string{
				"app":         "app_name",
				"token":       "token_id",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
				"template":    "template_name",
				"params":      `{"param1": "value1"}`,
				"message":     "push message",
			}

			message := buildMessage(valMap)
			_, err := Parse(message)
			g.Assert(err != nil).IsTrue()
		})

		g.It("Should not parse a message without template and plan text", func() {
			valMap := map[string]string{
				"app":         "app_name",
				"token":       "token_id",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
			}

			message := buildMessage(valMap)
			_, err := Parse(message)
			g.Assert(err != nil).IsTrue()
		})

		g.It("Should not parse a message which is not a json", func() {
			valMap := map[string]string{
				"app":         "app_name",
				"token":       "token_id",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
				"template":    "template_name",
				"params":      `{"param1": "value1"}`,
				"locale":      "en",
				"metadata":    `{"meta": "data"}`,
			}

			message := buildMessage(valMap)
			message = TrimSuffix(message, "}")
			_, err := Parse(message)
			g.Assert(err == nil).IsFalse()
		})

		g.It("Should not parse a message with no App", func() {
			valMap := map[string]string{
				"app":         "",
				"token":       "token_id",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
				"template":    "template_name",
				"params":      `{"param1": "value1"}`,
				"locale":      "en",
				"metadata":    `{"meta": "data"}`,
			}

			message := buildMessage(valMap)
			_, err := Parse(message)
			g.Assert(err == nil).IsFalse()
		})

		g.It("Should not parse a message with no Token", func() {
			valMap := map[string]string{
				"app":         "app_name",
				"token":       "",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
				"template":    "template_name",
				"params":      `{"param1": "value1"}`,
				"locale":      "en",
				"metadata":    `{"meta": "data"}`,
			}

			message := buildMessage(valMap)
			_, err := Parse(message)
			g.Assert(err == nil).IsFalse()
		})

		g.It("Should not parse a message with no Type", func() {
			valMap := map[string]string{
				"app":         "app_name",
				"token":       "token_id",
				"type":        "",
				"push_expiry": "0",
				"template":    "template_name",
				"params":      `{"param1": "value1"}`,
				"locale":      "en",
				"metadata":    `{"meta": "data"}`,
			}

			message := buildMessage(valMap)
			_, err := Parse(message)
			g.Assert(err == nil).IsFalse()
		})
	})

	g.Describe("Parser", func() {
		g.It("Should parse messages correctly", func() {
			valMap := map[string]string{
				"app":         "app_name",
				"token":       "token_id",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
				"template":    "template_name",
				"params":      `{"param1": "value1"}`,
				"locale":      "en",
				"metadata":    `{"meta": "data"}`,
			}
			message1 := buildMessage(valMap)
			valMap = map[string]string{
				"app":         "app_name",
				"token":       "token_id",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
			}
			message2 := buildMessage(valMap)
			valMap = map[string]string{
				"app":         "app_name",
				"token":       "token_id",
				"type":        "apns_or_gcm",
				"push_expiry": "0",
				"template":    "template_name",
				"params":      `{"param1": "value1"}`,
				"locale":      "en",
				"metadata":    `{"meta": "data"}`,
			}
			message3 := buildMessage(valMap)

			inChan := make(chan string, 1)
			defer close(inChan)
			outChan := make(chan *messages.RequestMessage, 1)
			defer close(outChan)

			go func() {
				inChan <- message1
				inChan <- message2
				inChan <- message3
			}()

			go Parser(inChan, outChan)
			out1 := <-outChan
			out2 := <-outChan

			g.Assert(compareMapRequestMessage(valMap, *out1)).IsTrue()
			g.Assert(compareMapRequestMessage(valMap, *out2)).IsTrue()
		})
	})
}

package main

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func parseTeamsLink(body string, onlineMeeting interface{}) string {
	if onlineMeeting != nil {
		return onlineMeeting.(map[string]interface{})["joinUrl"].(string)
	}

	re := regexp.MustCompile(`(http|https):\/\/(teams\.microsoft\.com)([\w.,@?^=%&:/~+#-]*[\w@?^=%&/~+#-])?`)
	links := re.FindAllString(body, -1)

	if len(links) > 0 {
		return links[0]
	}

	return ""
}

func html2text(body string) (string, error) {
	re := regexp.MustCompile(`(\r|\n)`)

	doc, err := html.Parse(strings.NewReader(re.ReplaceAllString(body, "")))
	if err != nil {
		return "", err
	}

	builder := strings.Builder{}
	parse := false

	var f func(n *html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode && parse {
			builder.WriteString(strings.TrimSpace(n.Data))
			builder.WriteRune('\n')
		}

		if n.Type == html.ElementNode && n.Data == "body" {
			parse = true
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	outputString := builder.String()
	strippedOutput := re.ReplaceAllString(outputString, "")
	if strippedOutput != "" {
		return outputString, nil
	}

	return "", nil
}

package main

import (
	"log"

	"github.com/deckrun/dockadvisor/parse"
)

func main() {
	result, err := parse.ParseDockerfile("FROM alpine\nWORKDIR usr/share/nginx/html\nRUN echo Hello World")
	if err != nil {
		log.Fatal("Error parsing Dockerfile:", err)
	}

	log.Println("Rules:")
	log.Println("------")
	for _, rule := range result.Rules {
		log.Printf("StartLine: %d, EndLine: %d, Rule Code: %s, Description: %s\n", rule.StartLine, rule.EndLine, rule.Code, rule.Description)
	}
	log.Println("------")
	log.Printf("Dockerfile Score: %d/100\n", result.Score)
}

package main

import (
	"flag"
	"log"
	"os"

	"github.com/deckrun/dockadvisor/parse"
)

func main() {
	filePath := flag.String("f", "Dockerfile", "path to Dockerfile")
	flag.Parse()

	content, err := os.ReadFile(*filePath)
	if err != nil {
		log.Fatalf("Error reading %s: %v", *filePath, err)
	}

	result, err := parse.ParseDockerfile(string(content))
	if err != nil {
		log.Fatal("Error parsing Dockerfile:", err)
	}

	log.Println("Rules:")
	log.Println("------")
	for _, rule := range result.Rules {
		if rule.StartLine == rule.EndLine {
			log.Printf("Line %d: [%s] %s\n", rule.StartLine, rule.Code, rule.Description)
		} else {
			log.Printf("Line %d-%d: [%s] %s\n", rule.StartLine, rule.EndLine, rule.Code, rule.Description)
		}
	}
	log.Println("------")
	log.Printf("Dockerfile Score: %d/100\n", result.Score)
}

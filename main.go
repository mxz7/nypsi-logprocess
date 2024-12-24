package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type LogLine struct {
	Date    string                 `json:"date"`
	Level   string                 `json:"level"`
	Message string                 `json:"msg"`
	Time    int64                  `json:"time"`
	Cluster string                 `json:"cluster"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func main() {
	router := gin.Default()

	router.POST("/process", func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")

		if authHeader != os.Getenv("AUTH_TOKEN") {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		body, err := io.ReadAll(c.Request.Body)

		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		rawLines := strings.Split(string(body), "\n")

		log.Printf("Received %d lines", len(rawLines))

		if len(rawLines) == 0 {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		formattedLines := process(rawLines)

		log.Printf("Processed %d lines", len(formattedLines))

		sort.Slice(formattedLines, func(i, j int) bool {
			return formattedLines[i].Time > formattedLines[j].Time
		})

		out := make([]string, len(formattedLines))

		log.Printf("%d %d", len(out), len(formattedLines))

		for i, line := range formattedLines {
			lineJSON, _ := json.Marshal(line)
			out[i] = string(lineJSON)
		}

		log.Printf("Returning %d lines", len(out))

		c.String(200, strings.Join(out, "\n"))
	})

	router.Run()
}

func process(rawLines []string) []LogLine {
	numWorkers := runtime.NumCPU()

	tasks := make(chan string, len(rawLines))
	results := make(chan LogLine, len(rawLines))

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(tasks, results, &wg)
	}

	for _, rawLine := range rawLines {
		tasks <- rawLine
	}

	close(tasks)

	wg.Wait()
	close(results)

	var out []LogLine

	for result := range results {
		out = append(out, result)
	}

	return out
}

func worker(tasks <-chan string, results chan<- LogLine, wg *sync.WaitGroup) {
	defer wg.Done()

	regex := regexp.MustCompile(`::\w+`)

	for task := range tasks {
		logLine := LogLine{}
		err := json.Unmarshal([]byte(task), &logLine)

		if err != nil {
			newLine := strings.Replace(task, `"cluster":0`, `"cluster":"0"`, 1)
			newLine = strings.Replace(newLine, `"cluster":1`, `"cluster":"1"`, 1)

			err := json.Unmarshal([]byte(newLine), &logLine)

			if err != nil {
				log.Printf("%v", err)
				log.Printf("failed, skipping line %s", newLine)
				continue
			}
		}

		logLine.Date = time.UnixMilli(logLine.Time).Format(time.RFC3339)
		logLine.Message = strings.TrimLeft(regex.ReplaceAllString(logLine.Message, ""), " ")

		results <- logLine
	}
}

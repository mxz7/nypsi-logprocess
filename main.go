package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
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

	regex := regexp.MustCompile(`/::\w+/gm`)

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

		var formattedLines []LogLine

		for _, rawLine := range rawLines {

			logLine := LogLine{}
			err := json.Unmarshal([]byte(rawLine), &logLine)

			if err != nil {
				newLine := strings.Replace(rawLine, `"cluster":0`, `"cluster":"0"`, 1)
				newLine = strings.Replace(newLine, `"cluster":1`, `"cluster":"1"`, 1)

				err := json.Unmarshal([]byte(newLine), &logLine)

				if err != nil {
					log.Printf("%v", err)
					log.Printf("failed, skipping line %s", newLine)
					continue
				}
			}

			logLine.Date = time.UnixMilli(logLine.Time).Format("2006-01-02 15:04:05")
			logLine.Message = regex.ReplaceAllString(logLine.Message, "")

			formattedLines = append(formattedLines, logLine)

		}

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

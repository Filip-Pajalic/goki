package main

import (
	"fmt"
	"time"

	"github.com/russross/blackfriday/v2"
)

type (
	Status     int
	Difficulty int
)

const (
	New Status = iota
	Learning
	Review
	Complete
)

const (
	Again Difficulty = iota
	Good
	Easy
)

type Card struct {
	Front        string    // The front of the card (question)
	Back         string    // The back of the card (answer)
	Score        int       // The card's score
	Interval     int       // The interval before the card is reviewed again
	EaseFactor   float64   // The ease factor for SM2 algorithm
	Status       Status    // The current status of the card
	LastReviewed time.Time // The last time the card was reviewed
}

// FilterValue returns the front of the card
func (c Card) FilterValue() string { return c.Front }

// Title returns the front of the card
func (c Card) Title() string { return c.Front }

// Description returns the back of the card
func (c Card) Description() string { return c.Back }

// NewCard creates a new Card with initial values
func NewCard(front, back string) *Card {
	return &Card{
		Front:    front,
		Back:     back,
		Score:    0,
		Interval: 0,
		Status:   New,
	}
}

// SM2 updates the card's state using the SM2 algorithm
func (c *Card) SM2(diff Difficulty) {
	switch diff {
	case Again:
		c.Score = 0
		c.Interval = 0
		c.Status = Learning
	case Good:
		if c.Score == 0 {
			c.Interval = 10
		} else {
			c.Interval = int(float64(c.Interval) * c.EaseFactor)
		}
		c.Status = Complete
		c.Score++
	case Easy:
		if c.Score == 0 {
			c.Interval = 20
		} else {
			c.Interval = int(float64(c.Interval) * c.EaseFactor)
		}
		c.Status = Complete
		c.Score++
	}
	c.EaseFactor = c.EaseFactor + 0.1 - (5-float64(diff))*(0.08+(5-float64(diff))*0.02)
	c.LastReviewed = time.Now()
}

// ToMarkdown converts the Card to Markdown format
func (c *Card) ToMarkdown() string {
	return fmt.Sprintf("### Card\n- **Front:** %s\n- **Back:** %s\n- **Score:** %d\n- **Interval:** %d\n- **Ease Factor:** %.2f\n- **Status:** %d\n- **Last Reviewed:** %s\n",
		c.Front, c.Back, c.Score, c.Interval, c.EaseFactor, c.Status, c.LastReviewed.Format(time.RFC3339))
}

// RenderMarkdownToHTML converts Markdown content to HTML using Blackfriday
func RenderMarkdownToHTML(markdownContent string) string {
	htmlContent := blackfriday.Run([]byte(markdownContent))
	return string(htmlContent)
}

// ParseMarkdown parses Markdown content into structured data
func (c *Card) ParseMarkdown() {
	c.Front = RenderMarkdownToHTML(c.Front)
	c.Back = RenderMarkdownToHTML(c.Back)
}

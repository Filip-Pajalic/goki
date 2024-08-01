package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

func updateRows() []table.Row {
	rows := []table.Row{}
	for _, deck := range currUser.decks {
		deck.Cards.Title = deck.Name
		rows = append(
			rows,
			table.Row{deck.Name, deck.NumNew(), deck.NumLearning(), deck.NumReview()},
		)
	}
	return rows
}

func initTable() {
	header := []table.Column{
		{Title: "Decks", Width: 20},
		{Title: "New", Width: 10},
		{Title: "Learning", Width: 10},
		{Title: "Review", Width: 10},
	}

	rows := updateRows()

	currUser.table = table.New(
		table.WithColumns(header),
		table.WithRows(rows),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	currUser.table.SetStyles(s)
}

func initInput() {
	currUser.input = textinput.New()
	currUser.input.Placeholder = ""
	currUser.input.PromptStyle = blurredStyle
	currUser.input.CharLimit = 50
}

func saveAll() {
	saveDecks()
	for _, deck := range currUser.decks {
		deck.saveCards()
	}
}

func saveDecks() {
	var sb strings.Builder
	sb.WriteString("# Decks\n\n")
	for _, deck := range currUser.decks {
		sb.WriteString(fmt.Sprintf("## %s\n- Deck File: %s\n", deck.Name, deck.Name))
	}

	markdownData := sb.String()

	// Write the Markdown data to the file
	err := os.WriteFile(filepath.Join(appDir, "decks.md"), []byte(markdownData), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Deck) saveCards() {
	var sb strings.Builder
	sb.WriteString("# Cards\n\n")
	for _, card := range d.Cards.Items() {
		c := card.(*Card)
		sb.WriteString(c.ToMarkdown())
	}

	markdownData := sb.String()

	// Ensure the directory exists
	cardDir := filepath.Join(appDir, "cards")
	if err := os.MkdirAll(cardDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Write the Markdown data to the file
	err := os.WriteFile(filepath.Join(cardDir, d.Name+".md"), []byte(markdownData), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func createFolders() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	appDir := filepath.Join(usr.HomeDir, ".local", "share", "goki")

	cardsDir := filepath.Join(appDir, "cards")

	if err := os.MkdirAll(appDir, 0755); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(cardsDir, 0755); err != nil {
		log.Fatal(err)
	}
	return appDir
}

func loadDecks() {
	appDir = createFolders()
	d := readDecks(filepath.Join(appDir, "decks.md"))
	for _, curr := range d {
		cards := readCards(filepath.Join(appDir, "cards", curr.Name+".md"))
		deck := InitDeck(curr.Name, curr.Name, cards)
		currUser.decks = append(currUser.decks, deck)
	}
}

func readDecks(fileName string) []*Deck {
	file, err := os.Open(fileName)
	if err != nil {
		file, err = os.Create(fileName)
		if err != nil {
			log.Fatalf("Error creating file: %s", err)
		}
		defer file.Close()
		_, err := file.WriteString("# Decks\n\n")
		if err != nil {
			log.Fatalf("Error writing to decks.md: %s", err)
		}
		_, err = file.Seek(0, 0)
		if err != nil {
			log.Fatalf("Error seeking file: %s", err)
		}
	} else {
		defer file.Close()
	}

	scanner := bufio.NewScanner(file)
	var decks []*Deck
	var currentDeck *Deck
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			if currentDeck != nil {
				decks = append(decks, currentDeck)
			}
			currentDeck = &Deck{Name: strings.TrimPrefix(line, "## "), Json: ""}
		} else if strings.HasPrefix(line, "- JSON File: ") {
			if currentDeck != nil {
				currentDeck.Name = strings.TrimPrefix(line, "- JSON File: ")
			}
		}
	}
	if currentDeck != nil {
		decks = append(decks, currentDeck)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %s", err)
	}

	return decks
}

func readCards(fileName string) []list.Item {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Error opening file: %s", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var cards []list.Item
	var currentCard *Card
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "### Card") {
			if currentCard != nil {
				cards = append(cards, currentCard)
			}
			currentCard = &Card{}
		} else if strings.HasPrefix(line, "- Front: ") {
			if currentCard != nil {
				currentCard.Front = strings.TrimPrefix(line, "- Front: ")
			}
		} else if strings.HasPrefix(line, "- Back: ") {
			if currentCard != nil {
				currentCard.Back = strings.TrimPrefix(line, "- Back: ")
			}
		}
	}
	if currentCard != nil {
		cards = append(cards, currentCard)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %s", err)
	}

	return cards
}

func updateTableColumns() {
	for _, deck := range currUser.decks {
		deck.GetReviewCards()
		deck.UpdateStatus()
	}
	currUser.UpdateTable()
}

func GetScreenDimensions() (int, int) {
	fd := int(os.Stdout.Fd())
	width, height, err := term.GetSize(fd)
	if err != nil {
		log.Println("Error getting screen dimensions:", err)
	}
	return width, height
}

func WrapString(input string, maxWidth int) string {
	if maxWidth < 1 {
		return input
	}

	var result strings.Builder
	lines := strings.Split(input, "\n")

	for _, line := range lines {
		if len(line) == 0 {
			result.WriteString("\n")
			continue
		}

		re := regexp.MustCompile(`(\S+ +)|\S+`)
		words := re.FindAllString(line, -1)
		currentLineLength := 0

		for _, word := range words {
			wordLength := len(word)
			spaceNeeded := wordLength
			if currentLineLength > 0 {
				spaceNeeded++ // account for a space before the word
			}

			if currentLineLength+spaceNeeded > maxWidth {
				result.WriteString("\n")
				currentLineLength = 0
			} else if currentLineLength > 0 {
				result.WriteString(" ")
				currentLineLength++
			}

			result.WriteString(word)
			currentLineLength += wordLength
		}
		result.WriteString("\n")
	}

	return result.String()
}

// New WrapStringDynamic function
func WrapStringDynamic(input string, screenWidth int, scaleFactor float64) string {
	dynamicWidth := int(scaleFactor * float64(screenWidth))
	return WrapString(input, dynamicWidth)
}

func PrintDecks() {
	var section []string
	section = append(section, "\nDecks:")

	for i, deck := range currUser.decks {
		// Get the number of new, learning, and review cards
		newCount := deck.NumNew()
		learningCount := deck.NumLearning()
		reviewCount := deck.NumReview()

		// Format the deck information
		deckInfo := fmt.Sprintf(
			"**%d. %s**\nNew: %d | Learning: %d | Review: %d\n",
			i+1, // Deck numbering starts from 1
			deck.Name,
			newCount,
			learningCount,
			reviewCount,
		)

		section = append(section, deckInfo)
	}

	// Join the section slice into a single string and print it
	output := strings.Join(section, "\n")
	fmt.Println(output)
}

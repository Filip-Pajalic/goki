package main

import (
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ReviewData struct {
	reviewing   bool
	complete    bool
	currIx      int
	curr        *Card
	reviewCards []*Card
}

type Deck struct {
	keyMap       keyMap
	help         help.Model
	progress     progress.Model
	descShown    bool
	resultsShown bool
	searching    bool

	numNew      int
	numLearning int
	numReview   int
	numComplete int

	Name         string     `json:"name"`
	Json         string     `json:"json"`
	Cards        list.Model `json:"-"`
	reviewData   ReviewData `json:"-"`
	deletedCards []*Card
}

func (d Deck) NumNew() string      { return strconv.Itoa(d.numNew) }
func (d Deck) NumLearning() string { return strconv.Itoa(d.numLearning) }
func (d Deck) NumReview() string   { return strconv.Itoa(d.numReview) }
func (d Deck) NumComplete() string { return strconv.Itoa(d.numComplete) }

func (d *Deck) StartReview() {
	d.reviewData.reviewing = true
	d.reviewData.complete = false
	d.reviewData.reviewCards = d.GetReviewCards()
	d.reviewData.currIx = 0
	if len(d.reviewData.reviewCards) > 0 {
		d.reviewData.curr = d.reviewData.reviewCards[0]
	}
}

func (d *Deck) UpdateStatus() {
	d.numNew, d.numLearning, d.numReview = 0, 0, 0
	temp := []list.Item{}
	for _, card := range d.Cards.Items() {
		if card != nil {
			c := card.(*Card)
			switch c.Status {
			case New:
				d.numNew++
			case Learning:
				d.numLearning++
			case Review:
				d.numReview++
			case Complete:
				d.numComplete++
			}
			temp = append(temp, c)
		}
	}
	d.Cards.SetItems(temp)
}

func (d *Deck) GetReviewCards() []*Card {
	var (
		timeNow = time.Now()

		c           *Card
		duration    time.Duration
		minutes     float64
		reviewCards []*Card
	)

	for _, card := range d.Cards.Items() {
		if card != nil {
			c = card.(*Card)
			if c.Status == New {
				reviewCards = append(reviewCards, c)
			} else {
				duration = timeNow.Sub(c.LastReviewed)
				minutes = math.Floor(duration.Minutes())
				if minutes >= float64(c.Interval) {
					reviewCards = append(reviewCards, c)
					if c.Status == Complete {
						c.Status = Review
					}
				}
			}
		}
	}

	rand.Shuffle(len(reviewCards), func(i, j int) {
		reviewCards[i], reviewCards[j] = reviewCards[j], reviewCards[i]
	})

	return reviewCards
}

func (d *Deck) UpdateReview() {
	d.reviewData.currIx++
	d.reviewData.complete = false
}

func InitDeck(name string, uuid string, lst []list.Item) *Deck {
	d := &Deck{
		help:       help.New(),
		progress:   progress.New(),
		Name:       name,
		Cards:      list.New(lst, InitCustomDelegate(), 0, 0),
		Json:       name,
		keyMap:     DeckKeyMap(),
		reviewData: ReviewData{},
	}
	d.progress.ShowPercentage = false

	d.Cards.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{d.keyMap.New, d.keyMap.Edit, d.keyMap.Delete, d.keyMap.Undo, d.keyMap.Quit}
	}
	d.Cards.SetSize(screenWidth-20, screenHeight-4)
	d.searching = false
	d.descShown = true
	d.help.ShowAll = false
	d.UpdateStatus()
	return d
}

func (d Deck) Init() tea.Cmd {
	return nil
}

func (d *Deck) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keyMap.Quit):
			if !d.searching {
				saveAll()
				return d, tea.Quit
			}
		case key.Matches(msg, d.keyMap.Back):
			if cli {
				saveAll()
				return d, tea.Quit
			} else if d.resultsShown {
				d.resultsShown = false
			} else {
				d.UpdateStatus()
				currUser.UpdateTable()
				return currUser.Update(nil)
			}
		case key.Matches(msg, d.keyMap.New):
			if !d.searching && !d.reviewData.reviewing {
				f := newDefaultForm()
				f.edit = false
				return f.Update(nil)
			}
		case key.Matches(msg, d.keyMap.Delete):
			if !d.searching && !d.reviewData.reviewing {
				d.deletedCards = append(d.deletedCards, d.Cards.Items()[d.Cards.Index()].(*Card))
				d.Cards.RemoveItem(d.Cards.Index())
				d.UpdateStatus()
				d.saveCards()
				return d.Update(nil)
			}
		case key.Matches(msg, d.keyMap.Undo):
			size := len(d.deletedCards)
			if size > 0 && !d.searching && !d.reviewData.reviewing {
				d.Cards.InsertItem(0, d.deletedCards[size-1])
				d.deletedCards = d.deletedCards[:size-1]
				return d.Update(nil)
			}
		case key.Matches(msg, d.keyMap.Edit):
			if !d.searching && !d.reviewData.reviewing && len(d.Cards.Items()) > 0 {
				card := d.Cards.SelectedItem().(*Card)
				f := EditForm(card.Front, card.Back)
				f.index = d.Cards.Index()
				f.edit = true
				return f.Update(nil)
			}
			return d.Update(nil)
		case key.Matches(msg, d.keyMap.Search):
			if !d.searching {
				d.searching = true
			}
		case key.Matches(msg, d.keyMap.Open):
			if !d.searching && len(d.Cards.Items()) > 0 {
				if d.reviewData.reviewing {
					d.reviewData.complete = true
				}
				return d.Update(nil)
			}
		case key.Matches(msg, d.keyMap.Easy):
			if d.reviewData.complete {
				d.reviewData.curr.SM2(Easy)
				d.UpdateReview()
				d.saveCards()
			}
		case key.Matches(msg, d.keyMap.Good):
			if d.reviewData.complete {
				d.reviewData.curr.SM2(Good)
				d.UpdateReview()
				d.saveCards()
			}
		case key.Matches(msg, d.keyMap.Again):
			if d.reviewData.complete {
				d.reviewData.curr.SM2(Again)
				d.UpdateReview()
				d.saveCards()
			}
		case key.Matches(msg, d.keyMap.Enter):
			if d.searching {
				d.searching = false
				d.resultsShown = true
			}
		}
	case tea.WindowSizeMsg:
		screenWidth, screenHeight = msg.Width, msg.Height
		cardStyle = cardStyle.MarginLeft(3 * screenWidth / 10).MarginTop(screenHeight / 10).
			Width(2 * screenWidth / 5).Height(screenHeight / 5)
	}

	if d.reviewData.reviewing {
		if d.reviewData.currIx > len(d.reviewData.reviewCards)-1 {
			if cli {
				saveAll()
				return d, tea.Quit
			}
			d.reviewData.reviewing = false
			d.reviewData.complete = false
			d.UpdateStatus()
			currUser.UpdateTable()
			return currUser.Update(nil)
		} else {
			d.reviewData.curr = d.reviewData.reviewCards[d.reviewData.currIx]
		}
	}

	d.Cards, cmd = d.Cards.Update(msg)

	return d, cmd
}

func (d Deck) View() string {
	screenWidth, screenHeight := GetScreenDimensions()

	if d.reviewData.reviewing {
		marginLeftScale, widthScale, heightScale, marginTopScale, wrapWidth, scaleFactor := getScaleFactors(screenWidth)

		pageContent := renderReviewView(d, wrapWidth, scaleFactor)
		progressContent := renderProgressView(d)

		cardStyle := getCardStyle(screenWidth, screenHeight, marginLeftScale, widthScale, heightScale, marginTopScale)
		page := questionStyle.Render(lipgloss.JoinVertical(lipgloss.Center, pageContent, progressContent))

		return cardStyle.Render(page)
	}

	listStyle := listStyle.Align(lipgloss.Left).MarginLeft(int(math.Max(float64((screenWidth-60)/2), 0)))
	return listStyle.Render(d.Cards.View())
}

func getScaleFactors(screenWidth int) (float64, float64, float64, float64, int, float64) {
	const (
		marginLeftScaleSmall = 1.0 / 10.0
		widthScaleSmall      = 4.0 / 5.0
		marginLeftScaleLarge = 3.0 / 10.0
		widthScaleLarge      = 2.0 / 5.0
		heightScale          = 4.0 / 5.0
		marginTopScale       = 4.0 / 10.0
	)

	var marginLeftScale, widthScale float64
	if screenWidth < 100 {
		marginLeftScale = marginLeftScaleSmall
		widthScale = widthScaleSmall
	} else {
		marginLeftScale = marginLeftScaleLarge
		widthScale = widthScaleLarge
	}

	wrapWidth := int(widthScale * float64(screenWidth))
	return marginLeftScale, widthScale, heightScale, marginTopScale, wrapWidth, widthScale
}

func renderReviewView(d Deck, wrapWidth int, scaleFactor float64) string {
	var sections []string

	if d.reviewData.complete {
		back := WrapStringDynamic(d.reviewData.curr.Back, wrapWidth, scaleFactor)
		sections = append(sections, answerStyle.Render(back))
		sections = append(sections, helpKeyColor.Render("Card Difficulty:"))
		sections = append(sections, lipgloss.NewStyle().Inline(true).Render(d.help.View(d)))
	} else {
		front := WrapStringDynamic(d.reviewData.curr.Front, wrapWidth, scaleFactor)
		sections = append(sections, front)
		sections = append(sections, deckFooterStyle.Render(d.help.View(d.reviewData.curr)))
	}

	return lipgloss.JoinVertical(lipgloss.Center, sections...)
}

func renderProgressView(d Deck) string {
	progress := float64(d.reviewData.currIx) / float64(len(d.reviewData.reviewCards))
	return progressStyle(d.progress.ViewAs(progress))
}

func getCardStyle(screenWidth, screenHeight int, marginLeftScale, widthScale, heightScale, marginTopScale float64) lipgloss.Style {
	minWidth := 20  // minimum width to ensure the content is readable
	minHeight := 10 // minimum height to ensure the content is readable

	marginLeft := marginLeftScale * float64(screenWidth)
	width := int(math.Max(widthScale*float64(screenWidth), float64(minWidth)))
	height := int(math.Max(heightScale*float64(screenHeight), float64(minHeight)))
	marginTop := marginTopScale * float64(screenHeight)

	style := cardStyle.MarginLeft(int(marginLeft)).MarginTop(int(marginTop)).Width(width).Height(height)
	if cli {
		style = style.Margin(0, 0, 1)
	}
	return style
}

package matrix

import (
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	matrixBg = lipgloss.Color("#111111")

	matrixPalettes = []lipgloss.Color{
		lipgloss.Color("#000048"),
		lipgloss.Color("#5e5e5e"),
		lipgloss.Color("#5a5a5a"),
		lipgloss.Color("#009a22"),
		lipgloss.Color("#36ba01"),
		lipgloss.Color("#002706"),
		lipgloss.Color("#00ff00"),
		lipgloss.Color("#009a22"),
		lipgloss.Color("#00ff2b"),
		lipgloss.Color("#36ba01"),
	}

	matrixGlyphs    = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ12345678909qwertyuiopasdfghjklzxcvbnm")
	nMatrixPalettes = len(matrixPalettes)
	nMatrixGlyphs   = len(matrixGlyphs)
)

func Start() tea.Msg {
	return MatrixTick{}
}

func InitialModel(width int, height int) Matrix {
	m := Matrix{
		Speed:  time.Millisecond * 100,
		Width:  width,
		Height: height,
	}

	return m.initSymbols()
}

type MatrixTick struct{}

type MatrixResized struct {
	Width  int
	Height int
}

type MatrixStop struct{}

type Matrix struct {
	Speed  time.Duration
	Width  int
	Height int

	symbols [][]string
	colors  [][]int
}

func (m Matrix) Init() tea.Cmd {
	return m.doTick()
}

func (m Matrix) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var newCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m = m.initSymbols()

	case MatrixTick:
		m = m.dropSymbols()
		newCmd = m.doTick()

	case MatrixStop:
		tea.Quit()
	}

	return m, newCmd
}

func (m Matrix) View() string {
	nRow := m.Height
	nColumn := m.Width / 2
	style := lipgloss.NewStyle().Background(matrixBg)

	var sb strings.Builder
	for row := 0; row < nRow; row++ {
		for col := 0; col < nColumn; col++ {
			colorIdx := m.colors[col][row]
			color := matrixPalettes[colorIdx]
			bold := colorIdx != 0

			sb.WriteString(style.Bold(bold).Foreground(color).Render(m.symbols[col][row]))
			sb.WriteString(" ")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m Matrix) doTick() tea.Cmd {
	return tea.Tick(m.Speed, func(_ time.Time) tea.Msg {
		return MatrixTick{}
	})
}

func (m Matrix) initSymbols() Matrix {
	// Create empty symbols and colors
	nRow := m.Height
	nColumn := m.Width / 2

	newSymbols := make([][]string, nColumn)
	for col := range newSymbols {
		newSymbols[col] = make([]string, nRow)
	}

	newColors := make([][]int, nColumn)
	for col := range newColors {
		newColors[col] = make([]int, nRow)
	}

	// Populate the symbols
	for col := 0; col < nColumn; col++ {
		for row := 0; row < nRow; row++ {
			glyphIdx := rand.Intn(nMatrixGlyphs)
			symbol := string(matrixGlyphs[glyphIdx])
			newSymbols[col][row] = symbol
		}
	}

	// Replace the symbols and colors
	m.symbols = newSymbols
	m.colors = newColors
	return m
}

func (m Matrix) dropSymbols() Matrix {
	// Move down each columns color
	for col, rows := range m.colors {
		// Move down the color idx
		for row := len(rows) - 1; row >= 1; row-- {
			m.colors[col][row] = m.colors[col][row-1]
		}

		// Reduce the color of first row
		m.colors[col][0]--
		if m.colors[col][0] < 0 {
			m.colors[col][0] = 0
		}

		if m.colors[col][0] == 0 && rand.Intn(100) <= 1 {
			m.colors[col][0] = nMatrixPalettes - 1
		}
	}

	return m
}

package monitor

import (
	"fmt"
	"slices"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mdouchement/openfand"
)

type model struct {
	table table.Model
}

func newTUI() *model {
	columns := []table.Column{
		{Title: "Fans", Width: 20},
		{Title: "Speeds", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		// table.WithRows(rows),
		table.WithFocused(false),
		// table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color("#00afff")).
		BorderForeground(lipgloss.Color("#00afff")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#ffffff")).
		Bold(false)
	t.SetStyles(s)

	return &model{
		table: t,
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height)
	case []openfand.Evaluation:
		m.update(msg)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *model) View() string {
	return m.table.View()
}

func (m *model) update(evals []openfand.Evaluation) error {
	var n int
	for _, eval := range evals {
		if eval.PWM == 0 {
			continue
		}
		evals[n] = eval
		n++
	}
	evals = evals[:n]

	//

	slices.SortStableFunc(evals, func(a, b openfand.Evaluation) int {
		if a.ID < b.ID {
			return -1
		}
		return 1
	})

	rows := make([]table.Row, 0, len(evals))
	for _, eval := range evals {
		rows = append(rows, table.Row{
			fmt.Sprintf("fan%d(%s)", eval.ID+1, eval.Label),
			fmt.Sprintf("%4d RPM (%2d%%)", eval.RPM, eval.PWM),
		})
	}

	m.table.SetRows(rows)
	return nil
}

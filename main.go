package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	blurple    = lipgloss.Color("63")
	titleStyle = lipgloss.NewStyle().Background(blurple).Width(50).Bold(true)

	ynab *Ynab = NewYnab()
)

type model struct {
	state      string
	form       *huh.Form
	budgets    []Budget
	accounts   []Account
	categories []Category
}

func createLoginForm() *huh.Form {
	theme := huh.ThemeCharm()
	theme.Focused.TextInput.Cursor = theme.Focused.TextInput.Cursor.Foreground(lipgloss.Color("230"))
	theme.Focused.Description = theme.Focused.Description.Foreground(blurple)
	theme.Focused.Title = theme.Focused.Title.Foreground(blurple)

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("code").Title("Enter a personal access token to get started\n").Validate(validateCode).Prompt(""),
			huh.NewConfirm().Key("write-code").Title("Do you want to save this token?").Affirmative("Yes").Negative("No"),
		),
	).WithTheme(theme)
}

func initialModel(state string) model {
	var form *huh.Form
	var budgets []Budget
	var accounts []Account
	var categories []Category
	if state == "app" {
		budgets = ynab.GetBudgets()
		accounts = ynab.GetAccounts(budgets[0].Id)
		categories = ynab.GetCategories(budgets[0].Id)
	} else {
		form = createLoginForm()
	}

	return model{
		state:      state,
		form:       form,
		budgets:    budgets,
		accounts:   accounts,
		categories: categories,
	}
}

func setUp() (state string) {
	state = "login"
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(home + "/.config/ynab")

	if err == nil {
		code := string(data)
		if validateCode(code) == nil {
			state = "app"
		}
	}

	return
}

func main() {
	state := setUp()

	var p *tea.Program
	if state == "login" {
		p = tea.NewProgram(initialModel(state))
	} else {
		p = tea.NewProgram(initialModel(state), tea.WithAltScreen())
	}

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func (m model) Init() tea.Cmd {
	if m.state == "login" {
		return m.form.Init()
	}
	return nil
}

func validateCode(code string) error {
	if ynab.ValidateAndSetCode(strings.TrimSpace(code)) {
		return nil
	}

	return errors.New("Authentication failed! Please enter another.")
}

func (m model) updateLogin(message tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.form.Update(message)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		m.state = "app"

		if m.form.GetBool("write-code") {
			home, _ := os.UserHomeDir()
			os.WriteFile(home+"/.config/ynab", []byte(m.form.GetString("code")), 0600)
		}

		cmd = tea.EnterAltScreen
	}

	return m, cmd
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, tea.Quit
		}
	}

	if m.state == "login" {
		return m.updateLogin(message)
	}

	return m, nil
}

func (m model) View() string {
	var display string

	switch m.state {
	case "login":
		display = titleStyle.Render(" YNAT")
		display += "\n\n  Hey, welcome to YNAT."
		display += "\n" + m.form.View()
	case "app":
		display = "BUDGETS"
		for _, budget := range m.budgets {
			display += fmt.Sprintf("\n%s", budget.Name)
		}
		for _, account := range m.accounts {
			display += fmt.Sprintf("\n%v: %v", account.Name, account.Balance)
		}
		for _, category := range m.categories {
			display += fmt.Sprintf("\n%v: %v", category.Name, category.Balance)
		}

	}

	return fmt.Sprint(display)
}

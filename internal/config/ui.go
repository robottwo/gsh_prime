package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

var (
	docStyle          = lipgloss.NewStyle().Margin(1, 2)
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
)

type model struct {
	runner        *interp.Runner
	list          list.Model
	selectionList list.Model
	state         state
	items         []settingItem
	textInput     textinput.Model
	activeSetting *settingItem
	quitting      bool
	width         int
	height        int
}

type state int

const (
	stateList state = iota
	stateEditing
	stateSelection
)

type settingItem struct {
	title       string
	description string
	envVar      string
	itemType    settingType
	options     []string // For list type
}

type settingType int

const (
	typeText settingType = iota
	typeList
	typeToggle
)

func (s settingItem) Title() string       { return s.title }
func (s settingItem) Description() string { return s.description }
func (s settingItem) FilterValue() string { return s.title }

// simpleItem implements list.Item
type simpleItem string

func (s simpleItem) Title() string       { return string(s) }
func (s simpleItem) Description() string { return "" }
func (s simpleItem) FilterValue() string { return string(s) }

func initialModel(runner *interp.Runner) model {
	items := []list.Item{
		settingItem{
			title:       "AI Chat Provider",
			description: "Provider for the main agent (Slow Model)",
			envVar:      "GSH_SLOW_MODEL_API_KEY",
			itemType:    typeList,
			options:     []string{"ollama", "openai", "openrouter"},
		},
		settingItem{
			title:       "AI Chat Model ID",
			description: "Model ID for the main agent",
			envVar:      "GSH_SLOW_MODEL_ID",
			itemType:    typeText,
		},
		settingItem{
			title:       "AI Chat Base URL",
			description: "API Base URL for the main agent",
			envVar:      "GSH_SLOW_MODEL_BASE_URL",
			itemType:    typeText,
		},
		settingItem{
			title:       "AI Completion Provider",
			description: "Provider for tab completion (Fast Model)",
			envVar:      "GSH_FAST_MODEL_API_KEY",
			itemType:    typeList,
			options:     []string{"ollama", "openai", "openrouter"},
		},
		settingItem{
			title:       "AI Completion Model ID",
			description: "Model ID for tab completion",
			envVar:      "GSH_FAST_MODEL_ID",
			itemType:    typeText,
		},
		settingItem{
			title:       "Assistant Height",
			description: "Height of the bottom assistant box",
			envVar:      "GSH_ASSISTANT_HEIGHT",
			itemType:    typeText,
		},
		settingItem{
			title:       "Safety Checks",
			description: "Enable/Disable approved command checks",
			envVar:      "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX",
			itemType:    typeToggle,
		},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = selectedItemStyle.Copy().Foreground(lipgloss.Color("240"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "GSH Configuration"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle

	sl := list.New([]list.Item{}, delegate, 0, 0)
	sl.SetShowStatusBar(false)
	sl.SetFilteringEnabled(false)
	sl.Styles.Title = titleStyle

	ti := textinput.New()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	ti.Focus()

	return model{
		runner:        runner,
		list:          l,
		selectionList: sl,
		state:         stateList,
		items:         make([]settingItem, len(items)),
		textInput:     ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height)
		m.selectionList.SetWidth(msg.Width)
		m.selectionList.SetHeight(msg.Height)

	case tea.KeyMsg:
		if m.state == stateEditing {
			switch msg.Type {
			case tea.KeyEsc:
				m.state = stateList
				return m, nil
			case tea.KeyEnter:
				newValue := m.textInput.Value()
				if err := saveConfig(m.activeSetting.envVar, newValue, m.runner); err != nil {
					// Handle error
				}
				m.state = stateList
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		if m.state == stateSelection {
			switch msg.Type {
			case tea.KeyEsc:
				m.state = stateList
				return m, nil
			case tea.KeyEnter:
				if i, ok := m.selectionList.SelectedItem().(simpleItem); ok {
					newValue := string(i)
					if err := saveConfig(m.activeSetting.envVar, newValue, m.runner); err != nil {
						// Handle error
					}
					m.state = stateList
					return m, nil
				}
			}
			m.selectionList, cmd = m.selectionList.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if i, ok := m.list.SelectedItem().(settingItem); ok {
				m.activeSetting = &i

				if i.itemType == typeToggle {
					curr := getEnv(m.runner, i.envVar)
					var newVal string
					if i.envVar == "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
						if strings.Contains(curr, `".*"`) || strings.Contains(curr, `".+"`) {
							newVal = "[]"
						} else {
							newVal = `[".*"]`
						}
					} else {
						if curr == "true" {
							newVal = "false"
						} else {
							newVal = "true"
						}
					}
					saveConfig(i.envVar, newVal, m.runner)
					return m, nil
				}

				if i.itemType == typeList {
					items := make([]list.Item, len(i.options))
					for idx, opt := range i.options {
						items[idx] = simpleItem(opt)
					}
					m.selectionList.SetItems(items)
					m.selectionList.Title = "Select " + i.title
					m.state = stateSelection
					return m, nil
				}

				m.textInput.SetValue(getEnv(m.runner, i.envVar))
				m.state = stateEditing
				return m, nil
			}
		}
	}

	if m.state == stateList {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	if m.state == stateEditing {
		return fmt.Sprintf(
			"\n  Edit %s\n\n  %s\n\n  (esc to cancel, enter to save)",
			m.activeSetting.title,
			m.textInput.View(),
		)
	}

	if m.state == stateSelection {
		return docStyle.Render(m.selectionList.View())
	}

	// Update descriptions with current values
	items := m.list.Items()
	for i, item := range items {
		if s, ok := item.(settingItem); ok {
			val := getEnv(m.runner, s.envVar)
			if s.envVar == "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
				if strings.Contains(val, `".*"`) || strings.Contains(val, `".+"`) {
					val = "Disabled (All commands allowed)"
				} else {
					val = "Enabled (Checked against approved list)"
				}
			}
			s.description = fmt.Sprintf("Current: %s", val)
			items[i] = s
		}
	}
	m.list.SetItems(items)

	return docStyle.Render(m.list.View())
}

func runConfigUI(runner *interp.Runner) error {
	p := tea.NewProgram(initialModel(runner), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func getEnv(runner *interp.Runner, key string) string {
	if v, ok := runner.Vars[key]; ok {
		return v.String()
	}
	return ""
}

func saveConfig(key, value string, runner *interp.Runner) error {
	// 1. Update current session
	runner.Vars[key] = expand.Variable{Str: value, Exported: true, Kind: expand.String}

	// 2. Persist to file
	configPath := filepath.Join(os.Getenv("HOME"), ".gsh_config_ui")
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write export command
	safeValue := strings.ReplaceAll(value, "'", "'\\''")
	line := fmt.Sprintf("export %s='%s'\n", key, safeValue)

	if _, err := f.WriteString(line); err != nil {
		return err
	}

	// 3. Ensure sourced in .gshrc
	gshrcPath := filepath.Join(os.Getenv("HOME"), ".gshrc")
	content, err := os.ReadFile(gshrcPath)
	if err == nil {
		if !strings.Contains(string(content), ".gsh_config_ui") {
			f2, err := os.OpenFile(gshrcPath, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				defer f2.Close()
				_, _ = f2.WriteString("\n# Source UI configuration\n[ -f ~/.gsh_config_ui ] && source ~/.gsh_config_ui\n")
			}
		}
	} else if os.IsNotExist(err) {
		f2, err := os.Create(gshrcPath)
		if err == nil {
			defer f2.Close()
			_, _ = f2.WriteString("\n# Source UI configuration\n[ -f ~/.gsh_config_ui ] && source ~/.gsh_config_ui\n")
		}
	}

	return nil
}

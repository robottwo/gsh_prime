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
	runner         *interp.Runner
	list           list.Model
	submenuList    list.Model
	selectionList  list.Model
	state          state
	items          []settingItem
	textInput      textinput.Model
	activeSetting  *settingItem
	activeSubmenu  *menuItem
	quitting       bool
	width          int
	height         int
}

type state int

const (
	stateList state = iota
	stateSubmenu
	stateEditing
	stateSelection
)

// menuItem represents a top-level menu entry (may have submenu)
type menuItem struct {
	title       string
	description string
	submenu     []settingItem // nil if this is a direct setting
	setting     *settingItem  // non-nil if this is a direct setting (no submenu)
}

func (m menuItem) Title() string       { return m.title }
func (m menuItem) Description() string { return m.description }
func (m menuItem) FilterValue() string { return m.title }

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
	// Define submenu items for slow model (chat/agent)
	slowModelSettings := []settingItem{
		{
			title:       "Provider",
			description: "LLM provider (ollama, openai, openrouter)",
			envVar:      "GSH_SLOW_MODEL_API_KEY",
			itemType:    typeList,
			options:     []string{"ollama", "openai", "openrouter"},
		},
		{
			title:       "Model ID",
			description: "Model identifier (e.g., qwen2.5:32b)",
			envVar:      "GSH_SLOW_MODEL_ID",
			itemType:    typeText,
		},
		{
			title:       "Base URL",
			description: "API endpoint URL",
			envVar:      "GSH_SLOW_MODEL_BASE_URL",
			itemType:    typeText,
		},
	}

	// Define submenu items for fast model (completion/suggestions)
	fastModelSettings := []settingItem{
		{
			title:       "Provider",
			description: "LLM provider (ollama, openai, openrouter)",
			envVar:      "GSH_FAST_MODEL_API_KEY",
			itemType:    typeList,
			options:     []string{"ollama", "openai", "openrouter"},
		},
		{
			title:       "Model ID",
			description: "Model identifier (e.g., qwen2.5)",
			envVar:      "GSH_FAST_MODEL_ID",
			itemType:    typeText,
		},
		{
			title:       "Base URL",
			description: "API endpoint URL",
			envVar:      "GSH_FAST_MODEL_BASE_URL",
			itemType:    typeText,
		},
	}

	// Direct settings (no submenu)
	assistantHeightSetting := settingItem{
		title:       "Assistant Height",
		description: "Height of the bottom assistant box",
		envVar:      "GSH_ASSISTANT_HEIGHT",
		itemType:    typeText,
	}
	safetyChecksSetting := settingItem{
		title:       "Safety Checks",
		description: "Enable/Disable approved command checks",
		envVar:      "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX",
		itemType:    typeToggle,
	}

	// Top-level menu items
	items := []list.Item{
		menuItem{
			title:       "Configure Slow Model",
			description: "Chat and agent operations",
			submenu:     slowModelSettings,
		},
		menuItem{
			title:       "Configure Fast Model",
			description: "Auto-completion and suggestions",
			submenu:     fastModelSettings,
		},
		menuItem{
			title:       "Assistant Height",
			description: "Height of the bottom assistant box",
			setting:     &assistantHeightSetting,
		},
		menuItem{
			title:       "Safety Checks",
			description: "Enable/Disable approved command checks",
			setting:     &safetyChecksSetting,
		},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = selectedItemStyle.Foreground(lipgloss.Color("240"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "GSH Configuration"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle

	subL := list.New([]list.Item{}, delegate, 0, 0)
	subL.SetShowStatusBar(false)
	subL.SetFilteringEnabled(false)
	subL.Styles.Title = titleStyle

	selL := list.New([]list.Item{}, delegate, 0, 0)
	selL.SetShowStatusBar(false)
	selL.SetFilteringEnabled(false)
	selL.Styles.Title = titleStyle

	ti := textinput.New()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	ti.Focus()

	return model{
		runner:        runner,
		list:          l,
		submenuList:   subL,
		selectionList: selL,
		state:         stateList,
		items:         []settingItem{},
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
		m.submenuList.SetWidth(msg.Width)
		m.submenuList.SetHeight(msg.Height)
		m.selectionList.SetWidth(msg.Width)
		m.selectionList.SetHeight(msg.Height)

	case tea.KeyMsg:
		// Handle text editing state
		if m.state == stateEditing {
			switch msg.Type {
			case tea.KeyEsc:
				if m.activeSubmenu != nil {
					m.state = stateSubmenu
				} else {
					m.state = stateList
				}
				return m, nil
			case tea.KeyEnter:
				newValue := m.textInput.Value()
				_ = saveConfig(m.activeSetting.envVar, newValue, m.runner)
				if m.activeSubmenu != nil {
					m.state = stateSubmenu
				} else {
					m.state = stateList
				}
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// Handle selection list state
		if m.state == stateSelection {
			switch msg.Type {
			case tea.KeyEsc:
				if m.activeSubmenu != nil {
					m.state = stateSubmenu
				} else {
					m.state = stateList
				}
				return m, nil
			case tea.KeyEnter:
				if i, ok := m.selectionList.SelectedItem().(simpleItem); ok {
					newValue := string(i)
					_ = saveConfig(m.activeSetting.envVar, newValue, m.runner)
					if m.activeSubmenu != nil {
						m.state = stateSubmenu
					} else {
						m.state = stateList
					}
					return m, nil
				}
			}
			m.selectionList, cmd = m.selectionList.Update(msg)
			return m, cmd
		}

		// Handle submenu state
		if m.state == stateSubmenu {
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.activeSubmenu = nil
				m.state = stateList
				return m, nil
			case "enter":
				if i, ok := m.submenuList.SelectedItem().(settingItem); ok {
					m.activeSetting = &i
					return m, m.handleSettingAction(&i)
				}
			}
			m.submenuList, cmd = m.submenuList.Update(msg)
			return m, cmd
		}

		// Handle main list state
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(menuItem); ok {
				// If this menu item has a submenu, navigate to it
				if item.submenu != nil {
					m.activeSubmenu = &item
					subItems := make([]list.Item, len(item.submenu))
					for idx, s := range item.submenu {
						subItems[idx] = s
					}
					m.submenuList.SetItems(subItems)
					m.submenuList.Title = item.title
					m.state = stateSubmenu
					return m, nil
				}
				// If this is a direct setting, handle it
				if item.setting != nil {
					m.activeSetting = item.setting
					return m, m.handleSettingAction(item.setting)
				}
			}
		}
	}

	if m.state == stateList {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleSettingAction processes the action for a setting item
func (m *model) handleSettingAction(s *settingItem) tea.Cmd {
	if s.itemType == typeToggle {
		curr := getEnv(m.runner, s.envVar)
		var newVal string
		if s.envVar == "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
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
		_ = saveConfig(s.envVar, newVal, m.runner)
		return nil
	}

	if s.itemType == typeList {
		items := make([]list.Item, len(s.options))
		for idx, opt := range s.options {
			items[idx] = simpleItem(opt)
		}
		m.selectionList.SetItems(items)
		m.selectionList.Title = "Select " + s.title
		m.state = stateSelection
		return nil
	}

	// typeText
	m.textInput.SetValue(getEnv(m.runner, s.envVar))
	m.state = stateEditing
	return nil
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

	if m.state == stateSubmenu {
		// Update submenu descriptions with current values
		items := m.submenuList.Items()
		for i, item := range items {
			if s, ok := item.(settingItem); ok {
				val := getEnv(m.runner, s.envVar)
				if val == "" {
					val = "(not set)"
				}
				s.description = fmt.Sprintf("Current: %s", val)
				items[i] = s
			}
		}
		m.submenuList.SetItems(items)
		return docStyle.Render(m.submenuList.View())
	}

	// Update main menu descriptions with current values for direct settings
	items := m.list.Items()
	for i, item := range items {
		if mi, ok := item.(menuItem); ok {
			if mi.setting != nil {
				val := getEnv(m.runner, mi.setting.envVar)
				if mi.setting.envVar == "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
					if strings.Contains(val, `".*"`) || strings.Contains(val, `".+"`) {
						val = "Disabled (All commands allowed)"
					} else {
						val = "Enabled"
					}
				}
				if val == "" {
					val = "(not set)"
				}
				mi.description = fmt.Sprintf("Current: %s", val)
				items[i] = mi
			}
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

	// Write export command
	safeValue := strings.ReplaceAll(value, "'", "'\\''")
	line := fmt.Sprintf("export %s='%s'\n", key, safeValue)

	if _, err := f.WriteString(line); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// 3. Ensure sourced in .gshrc
	gshrcPath := filepath.Join(os.Getenv("HOME"), ".gshrc")
	content, err := os.ReadFile(gshrcPath)
	if err == nil {
		if !strings.Contains(string(content), ".gsh_config_ui") {
			if f2, err := os.OpenFile(gshrcPath, os.O_APPEND|os.O_WRONLY, 0644); err == nil {
				_, _ = f2.WriteString("\n# Source UI configuration\n[ -f ~/.gsh_config_ui ] && source ~/.gsh_config_ui\n")
				_ = f2.Close()
			}
		}
	} else if os.IsNotExist(err) {
		if f2, err := os.Create(gshrcPath); err == nil {
			_, _ = f2.WriteString("\n# Source UI configuration\n[ -f ~/.gsh_config_ui ] && source ~/.gsh_config_ui\n")
			_ = f2.Close()
		}
	}

	return nil
}

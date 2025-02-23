package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kerem-kaynak/llmshark/internal/config"
	"github.com/kerem-kaynak/llmshark/internal/postgres"
	"github.com/kerem-kaynak/llmshark/internal/storage"
)

type state int

const (
	stateInitial state = iota
	stateCredentials
	stateLoading
	stateExplorer
	stateComment
	stateEditCredentials
)

type model struct {
	config       *config.Config
	state        state
	client       *postgres.Client
	credStore    *storage.CredentialStore
	schemas      []postgres.Schema
	inputs       []textinput.Model
	cursor       cursor
	activeInput  int
	err          error
	width        int
	height       int
	message      string
	commentInput textinput.Model
	spinner      spinner.Model
}

type cursor struct {
	schema int
	table  int
	column int
}

func NewApp(cfg *config.Config) (*tea.Program, error) {
	store, err := storage.NewCredentialStore(cfg.CredentialsPath)
	if err != nil {
		return nil, err
	}

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Padding(2, 0, 0, 4)

	// Initialize inputs
	inputs := make([]textinput.Model, 5)
	labels := []string{"Host", "Port", "Database", "User", "Password"}
	defaults := []string{"localhost", "5432", "", "", ""}

	for i := range inputs {
		t := textinput.New()
		t.Placeholder = labels[i]
		t.SetValue(defaults[i])
		t.CharLimit = 50

		if i == 0 {
			t.Focus()
		}

		if i == len(inputs)-1 {
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}

		inputs[i] = t
	}

	commentInput := textinput.New()
	commentInput.Placeholder = "Enter comment"
	commentInput.Focus()

	m := &model{
		config:    cfg,
		state:     stateLoading,
		credStore: store,
		cursor: cursor{
			schema: 0,
			table:  -1,
			column: -1,
		},
		activeInput:  0,
		spinner:      s,
		inputs:       inputs,
		err:          nil,
		commentInput: commentInput,
	}

	return tea.NewProgram(m, tea.WithAltScreen()), nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		func() tea.Msg {
			creds, err := m.credStore.Load()
			if err != nil {
				return errMsg{err}
			}
			if creds != nil {
				return credsMsg{creds}
			}
			return noCredsMsg{}
		},
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == stateCredentials && m.err != nil {
			m.err = nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.state != stateCredentials && m.state != stateLoading {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

	case errMsg:
		m.err = msg.error
		m.schemas = nil
		m.client = nil
		m.state = stateCredentials
		return m, nil

	case credsMsg:
		m.schemas = nil
		m.state = stateLoading
		return m, connectToDB(msg.creds)

	case connectedMsg:
		m.client = msg.client
		m.schemas = msg.schemas
		m.message = "Schema loaded successfully!"
		m.state = stateExplorer
		return m, nil

	case noCredsMsg:
		m.schemas = nil
		m.client = nil
		m.state = stateCredentials
		return m, nil

	case spinner.TickMsg:
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		return m, spinnerCmd
	}

	switch m.state {
	case stateCredentials:
		return m.updateCredentials(msg)
	case stateLoading:
		return m, m.spinner.Tick
	case stateExplorer:
		return m.updateExplorer(msg)
	case stateComment:
		return m.updateComment(msg)
	case stateEditCredentials:
		return m.updateCredentials(msg)
	}

	return m, nil
}

func connectToDB(creds *storage.Credentials) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		client, err := postgres.NewClient(ctx, creds)
		if err != nil {
			return errMsg{err}
		}

		schemas, err := client.GetSchemas(ctx, postgres.DefaultSchemaFilter)
		if err != nil {
			client.Close()
			return errMsg{err}
		}
		return connectedMsg{client: client, schemas: schemas}
	}
}

func (m model) View() string {
	switch m.state {
	case stateCredentials:
		return m.credentialsView()
	case stateLoading:
		loadingText := fmt.Sprintf("%s Loading...", m.spinner.View())
		if m.err != nil {
			loadingText += "\n\n" + errorStyle.Render(m.err.Error())
		}
		return loadingText
	case stateExplorer:
		return m.explorerView()
	case stateComment:
		return m.commentView()
	case stateEditCredentials:
		return m.credentialsView()
	default:
		return fmt.Sprintf("%s Loading...", m.spinner.View())
	}
}

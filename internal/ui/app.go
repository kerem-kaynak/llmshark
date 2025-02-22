package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kerem-kaynak/llmshark/internal/config"
	"github.com/kerem-kaynak/llmshark/internal/postgres"
	"github.com/kerem-kaynak/llmshark/internal/storage"
)

type state int

const (
	stateInitial state = iota
	stateCredentials
	stateExplorer
	stateComment
	stateEditCredentials // New state for editing connection details
)

type model struct {
	config      *config.Config
	state       state
	client      *postgres.Client
	credStore   *storage.CredentialStore
	schemas     []postgres.Schema
	inputs      []textinput.Model
	cursor      cursor
	activeInput int
	err         error
	width       int
	height      int
	message     string
	commentText string
}

type cursor struct {
	schema int
	table  int
	column int
}

func NewApp(cfg *config.Config) (*tea.Program, error) {
	// Initialize credential store first
	store, err := storage.NewCredentialStore(cfg.CredentialsPath)
	if err != nil {
		return nil, err
	}

	m := &model{
		config:    cfg,
		state:     stateCredentials,
		credStore: store, // Set the credential store
		cursor: cursor{
			schema: 0,
			table:  -1,
			column: -1,
		},
		activeInput: 0,
	}

	// Initialize inputs
	m.inputs = make([]textinput.Model, 5)
	labels := []string{"Host", "Port", "Database", "User", "Password"}
	defaults := []string{"localhost", "5432", "", "", ""}

	for i := range m.inputs {
		t := textinput.New()
		t.Placeholder = labels[i]
		t.SetValue(defaults[i])
		t.CharLimit = 50

		if i == 0 {
			t.Focus()
		}

		if i == len(m.inputs)-1 {
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}

		m.inputs[i] = t
	}

	return tea.NewProgram(m, tea.WithAltScreen()), nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		func() tea.Msg {
			// Check for existing credentials
			if creds, err := m.credStore.Load(); err == nil && creds != nil {
				return credsMsg{creds}
			}
			return nil
		},
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear error message on any key press
		if m.err != nil {
			m.err = nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.state != stateCredentials {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

	case errMsg:
		m.err = msg.error
		return m, nil

	case credsMsg:
		return m.handleCredentials(msg.creds)

	case schemasMsg:
		m.schemas = msg.schemas
		m.message = "Schema loaded successfully!"
		return m, nil
	}

	var cmd tea.Cmd
	switch m.state {
	case stateCredentials:
		return m.updateCredentials(msg)
	case stateExplorer:
		return m.updateExplorer(msg)
	case stateComment:
		return m.updateComment(msg)
	case stateEditCredentials:
		return m.updateCredentials(msg)
	}

	return m, cmd
}

func (m model) View() string {
	switch m.state {
	case stateCredentials:
		return m.credentialsView()
	case stateExplorer:
		return m.explorerView()
	case stateComment:
		return m.commentView()
	case stateEditCredentials: // New state for editing connection details
		return m.credentialsView() // Reuse the credentials view
	default:
		return "Loading..."
	}
}

func (m model) handleCredentials(creds *storage.Credentials) (tea.Model, tea.Cmd) {
	ctx := context.Background()
	client, err := postgres.NewClient(ctx, creds)
	if err != nil {
		m.err = err
		return m, nil
	}

	m.client = client
	m.state = stateExplorer
	return m, m.fetchSchemas
}

func (m *model) fetchSchemas() tea.Msg {
	ctx := context.Background()
	schemas, err := m.client.GetSchemas(ctx, postgres.DefaultSchemaFilter)
	if err != nil {
		return errMsg{err}
	}
	return schemasMsg{schemas}
}

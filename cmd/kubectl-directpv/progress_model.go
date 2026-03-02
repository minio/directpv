// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

const (
	padding  = 1
	maxWidth = 80
	tick     = "✔"
)

type progressLog struct {
	log  string
	done bool
}

type progressNotification struct {
	log          string
	progressLogs []progressLog
	message      string
	percent      float64
	done         bool
	err          error
}

type progressModel struct {
	model        *progress.Model
	spinner      spinner.Model
	message      string
	progressLogs []progressLog
	logs         []string
	done         bool
	err          error
}

func newProgressModel(withProgressBar bool) *progressModel {
	progressM := &progressModel{}
	progressM.spinner = spinner.New()
	progressM.spinner.Spinner = spinner.Points
	progressM.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#F7971E"))
	if withProgressBar {
		progress := progress.New(progress.WithDefaultGradient())
		progressM.model = &progress
	}
	return progressM
}

func finalPause() tea.Cmd {
	return tea.Tick(time.Millisecond*750, func(_ time.Time) tea.Msg {
		return nil
	})
}

func (m progressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Batch(tea.Sequence(finalPause(), tea.Quit))
		}
		return m, nil
	case tea.WindowSizeMsg:
		if m.model != nil {
			m.model.Width = msg.Width - padding*2 - 4
			if m.model.Width > maxWidth {
				m.model.Width = maxWidth
			}
		}
		return m, nil

	case progressNotification:
		if msg.log != "" {
			if m.logs == nil {
				m.logs = []string{msg.log}
			} else {
				m.logs = append(m.logs, msg.log)
			}
		}
		if len(msg.progressLogs) > 0 {
			m.progressLogs = msg.progressLogs
		}
		m.message = msg.message
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		var cmds []tea.Cmd
		if msg.done {
			m.done = msg.done
			cmds = append(cmds, tea.Sequence(finalPause(), tea.Quit))
		}
		if m.model != nil && msg.percent > 0.0 {
			cmds = append(cmds, m.model.SetPercent(msg.percent))
		}
		return m, tea.Batch(cmds...)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		if m.model != nil {
			progressModel, cmd := m.model.Update(msg)
			pModel := progressModel.(progress.Model)
			m.model = &pModel
			return m, cmd
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m progressModel) View() string {
	pad := strings.Repeat(" ", padding)

	var builder strings.Builder
	builder.Grow(256)

	builder.WriteString("\n")

	if m.model != nil {
		builder.WriteString(pad)
		builder.WriteString(m.model.View())
		builder.WriteString("\n\n")
	}

	if !m.done && m.message != "" {
		builder.WriteString(pad)
		builder.WriteString(m.message)
		builder.WriteString("\n\n")
	}

	if len(m.progressLogs) > 0 {
		for _, pl := range m.progressLogs {
			builder.WriteString(pad)
			builder.WriteString(color.HiYellowString(pl.log))
			builder.WriteByte(' ')
			if pl.done {
				builder.WriteString(m.spinner.Style.Render(tick))
			} else {
				builder.WriteString(m.spinner.View())
			}
			builder.WriteByte('\n')
		}
		builder.WriteByte('\n')
	}

	if len(m.logs) > 0 {
		for _, log := range m.logs {
			builder.WriteString(pad)
			builder.WriteString(color.HiYellowString(log))
			builder.WriteByte('\n')
		}
		builder.WriteByte('\n')
	}

	if m.err != nil {
		builder.WriteString(pad)
		builder.WriteString(color.HiRedString("Error: %s\n\n", m.err.Error()))
	}

	builder.WriteString(pad)
	return builder.String()
}

func toProgressLogs(progressMap map[string]progressLog) (logs []progressLog) {
	for _, v := range progressMap {
		logs = append(logs, v)
	}
	return
}

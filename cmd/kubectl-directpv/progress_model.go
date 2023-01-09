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
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

const (
	padding  = 1
	maxWidth = 80
)

type progressNotification struct {
	log     string
	message string
	percent float64
	done    bool
	err     error
}

type progressModel struct {
	model   progress.Model
	message string
	logs    []string
	done    bool
	err     error
}

func finalPause() tea.Cmd {
	return tea.Tick(time.Millisecond*750, func(_ time.Time) tea.Msg {
		return nil
	})
}

func (m progressModel) Init() tea.Cmd {
	return nil
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.model.Width = msg.Width - padding*2 - 4
		if m.model.Width > maxWidth {
			m.model.Width = maxWidth
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
		if msg.percent > 0.0 {
			cmds = append(cmds, m.model.SetPercent(msg.percent))
		}
		return m, tea.Batch(cmds...)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.model.Update(msg)
		m.model = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m progressModel) View() (str string) {
	pad := strings.Repeat(" ", padding)
	str = "\n" + pad + m.model.View() + "\n\n"
	if !m.done {
		str += pad + fmt.Sprintf("%s \n\n", m.message)
	}
	for i := range m.logs {
		str += pad + color.HiYellowString(fmt.Sprintf("%s \n\n", m.logs[i]))
	}
	if m.err != nil {
		str += pad + color.HiRedString("Error; %s \n\n", m.err.Error())
	}
	return str + pad
}

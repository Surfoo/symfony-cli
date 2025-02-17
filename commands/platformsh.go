/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package commands

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/php"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/symfony-cli/util"
)

type platformshCLI struct {
	Commands []*console.Command

	path string
}

func NewPlatformShCLI() (*platformshCLI, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}
	p := &platformshCLI{
		path: filepath.Join(home, ".platformsh", "bin", "platform"),
	}
	for _, command := range platformsh.Commands {
		command.Action = p.proxyPSHCmd(strings.TrimPrefix(command.Category+":"+command.Name, "cloud:"))
		command.Args = []*console.Arg{
			{Name: "anything", Slice: true, Optional: true},
		}
		command.FlagParsing = console.FlagParsingSkipped
		p.Commands = append(p.Commands, command)
	}
	return p, nil
}

func (p *platformshCLI) PSHMainCommands() []*console.Command {
	names := map[string]bool{
		"cloud:project:list":       true,
		"cloud:environment:list":   true,
		"cloud:environment:branch": true,
		"cloud:tunnel:open":        true,
		"cloud:environment:ssh":    true,
		"cloud:environment:push":   true,
		"cloud:domain:list":        true,
		"cloud:variable:list":      true,
		"cloud:user:add":           true,
	}
	mainCmds := []*console.Command{}
	for _, command := range p.Commands {
		if names[command.FullName()] {
			mainCmds = append(mainCmds, command)
		}
	}
	return mainCmds
}

func (p *platformshCLI) proxyPSHCmd(commandName string) console.ActionFunc {
	return func(commandName string) console.ActionFunc {
		return func(c *console.Context) error {
			// the Platform.sh CLI is always available on the containers thanks to the configurator
			if !util.InCloud() {
				home, err := homedir.Dir()
				if err != nil {
					return err
				}
				if err := php.InstallPlatformPhar(home); err != nil {
					return console.Exit(err.Error(), 1)
				}
			}

			args := os.Args[1:]
			for i := range args {
				for _, name := range c.Command.Names() {
					args[i] = strings.Replace(args[i], name, commandName, 1)
				}
			}
			e := p.executor(args)
			return console.Exit("", e.Execute(false))
		}
	}(commandName)
}

func (p *platformshCLI) executor(args []string) *php.Executor {
	e := &php.Executor{
		BinName: "php",
		Args:    append([]string{"php", p.path}, args...),
		ExtraEnv: []string{
			"PLATFORMSH_CLI_APPLICATION_NAME=Platform.sh CLI for Symfony",
			"PLATFORMSH_CLI_APPLICATION_EXECUTABLE=symfony cloud:",
		},
	}
	e.Paths = append([]string{filepath.Dir(p.path)}, e.Paths...)
	return e
}

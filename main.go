package main

import (
	"fmt"
	"github.com/lucasrodlima/rss_aggregator/internal/config"
	"os"
)

type state struct {
	config *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	err := c.handlers[cmd.name](s, cmd)
	if err != nil {
		fmt.Println("Error running command handler function")
		return err
	}
	return nil
}

func (c *commands) register(name string, f func(s *state, cmd command) error) {
	c.handlers[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 3 {
		return fmt.Errorf("Username is required")
	}

	newUsername := cmd.args[2]

	err := s.config.SetUser(newUsername)
	if err != nil {
		return fmt.Errorf("Error setting new user")
	}

	fmt.Printf("Login successful as %s!\n", newUsername)
	return nil
}

func main() {
	sysConfig, err := config.Read()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	currentState := state{
		config: sysConfig,
	}

	currentCommands := commands{
		handlers: make(map[string]func(*state, command) error),
	}

	currentCommands.register("login", handlerLogin)

	currentArgs := os.Args
	if len(currentArgs) < 2 {
		fmt.Println("Not enough arguments were provided")
		os.Exit(1)
	}

	currentCommand := command{
		name: currentArgs[1],
		args: currentArgs,
	}

	err = currentCommands.run(&currentState, currentCommand)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

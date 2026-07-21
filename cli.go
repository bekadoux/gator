package main

import (
	"bekadoux/gator/internal/config"
	"bekadoux/gator/internal/database"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	handler, ok := c.handlers[cmd.name]
	if !ok {
		return fmt.Errorf("could not get %q from commands (perhaps it's unregistered?)", cmd.name)
	}
	err := handler(s, cmd)
	if err != nil {
		return err
	}

	return nil
}

func (c *commands) register(name string, f func(s *state, cmd command) error) {
	c.handlers[name] = f
}

func newCommands() *commands {
	return &commands{
		handlers: make(map[string]func(*state, command) error),
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("no args provided for %q", cmd.name)
	}
	if len(cmd.args) > 1 {
		return fmt.Errorf("too many arguments for %q", cmd.name)
	}

	userName := cmd.args[0]
	if exists, err := checkUserExists(s, userName); err == nil && !exists {
		return fmt.Errorf("user %q doesn't exist", userName)
	} else if err != nil {
		return err
	}

	if err := s.cfg.SetUser(cmd.args[0]); err != nil {
		return fmt.Errorf("%q: could not set user name: %w", cmd.name, err)
	}
	fmt.Println("The user name has been set.")

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("no args provided for %q", cmd.name)
	}
	if len(cmd.args) > 1 {
		return fmt.Errorf("too many arguments for %q", cmd.name)
	}

	userName := cmd.args[0]
	if exists, err := checkUserExists(s, userName); err == nil && exists {
		return fmt.Errorf("user %q already exists", userName)
	} else if err != nil {
		return err
	}

	userParams := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      userName,
	}

	user, err := s.db.CreateUser(context.Background(), userParams)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	fmt.Printf("%+v was created\n", user)

	if err := handlerLogin(s, cmd); err != nil {
		return fmt.Errorf("login after register: %w", err)
	}

	return nil
}

func checkUserExists(s *state, name string) (bool, error) {
	_, err := s.db.GetUser(context.Background(), name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("check whether user %q exists: %w", name, err)
	}
	return true, nil
}

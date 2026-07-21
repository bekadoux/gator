package main

import (
	"bekadoux/gator/internal/config"
	"bekadoux/gator/internal/database"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("not enough arguments")
		os.Exit(1)
	}
	args = args[1:]

	//cfgPath := "/home/bekadoux/.gatorconfig.json"
	cfg, err := config.Read("")
	if err != nil {
		panic(err)
	}

	s := &state{}
	s.cfg = &cfg

	cmds := newCommands()
	cmds.register("login", handlerLogin)

	db, err := sql.Open("postgres", s.cfg.DbURL)
	if err != nil {
		panic(err)
	}
	s.db = database.New(db)

	cmd := command{
		name: args[0],
		args: args[1:],
	}

	if err := cmds.run(s, cmd); err != nil {
		fmt.Printf("could not run %q: %s\n", cmd.name, err.Error())
		os.Exit(1)
	}
}

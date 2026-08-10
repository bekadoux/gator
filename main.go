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
		fmt.Println("Error: not enough arguments")
		os.Exit(1)
	}
	args = args[1:]

	//cfgPath := "/home/bekadoux/.gatorconfig.json"
	cfg, err := config.Read("")
	if err != nil {
		fmt.Printf("Could not read config file at %s.\n", cfg.Filepath)
		os.Exit(1)
	}

	s := &state{}
	s.cfg = &cfg
	fmt.Println(s.cfg)

	cmds := newCommands()
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("users", handlerUsers)
	cmds.register("reset", handlerReset)
	cmds.register("agg", handlerAgg)

	db, err := sql.Open("postgres", s.cfg.DbURL)
	if err != nil {
		fmt.Printf("Could not establish database connection.\n")
		os.Exit(1)
	}
	s.db = database.New(db)

	cmd := command{
		name: args[0],
		args: args[1:],
	}

	if err := cmds.run(s, cmd); err != nil {
		fmt.Printf("%s: %s\n", cmd.name, err.Error())
		os.Exit(1)
	}
}

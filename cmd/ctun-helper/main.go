package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serveCmd(os.Args[2:])
	case "install":
		err = installCmd(os.Args[2:])
	case "uninstall":
		err = uninstall()
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ctun-helper:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ctun-helper serve --uid N | install --uid N --core PATH | uninstall")
	os.Exit(2)
}

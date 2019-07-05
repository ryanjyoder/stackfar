package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ryanjyoder/stackfar"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("must provide a store directory")
	}

	store, err := stackfar.NewStreamStore(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	rows, err := store.ListStreamIDs()

	for rows.Next() {
		id := ""
		err := rows.Scan(&id)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id)
	}
	fmt.Println("finished listing ids")

}

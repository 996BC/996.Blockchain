package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/utils"
)

func main() {
	m := flag.Int("m", 0,
		`working mode:
1: Generate a sKey
2: Generate a pKey
3: Generate a pKey from a sKey
4: Generate a sKey from a pKey
5: Generate a new sKey from a sKey
all require ouput path, 3,4,5 require source input path`)

	s := flag.String("s", "", "source input path")
	o := flag.String("o", "", "output path")
	flag.Parse()

	if *m <= 0 || *m > 5 {
		fmt.Printf("Invalid mode:%d\n", *m)
		os.Exit(1)
	}

	if len(*o) == 0 {
		fmt.Printf("output path should not be empty")
		os.Exit(1)
	}

	if err := utils.AccessCheck(*o); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *m >= 3 && *m <= 5 {
		if len(*s) == 0 {
			fmt.Printf("source input path should not be empty")
			os.Exit(1)
		}

		if err := utils.AccessCheck(*o); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	var err error
	switch *m {
	case 1:
		_, err = crypto.NewSKey(*o)
		break
	case 2:
		_, err = crypto.NewPKey(*o)
		break
	case 3:
		err = crypto.OpenSKey(*s, *o)
		break
	case 4:
		err = crypto.SealPKey(*s, *o)
		break
	case 5:
		err = crypto.ReNewSKey(*s, *o)
	}
	if err != nil {
		fmt.Printf("error happen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Finish, checkout .*Key file in the %s\n", *o)
}

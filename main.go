package main

import (
	"fmt"

	"github.com/webitel/im-providers-service/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		fmt.Println(err.Error())
		return
	}
}

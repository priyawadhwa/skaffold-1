package main

import (
	"fmt"
	"io/ioutil"
	"time"
)

func main() {
	var s string
	contents, err := ioutil.ReadFile("file")
	if err != nil {
		panic(err)
	}
	s = string(contents)
	i := 0
	for {
		fmt.Println("Hello world!")
		if i < 11 {
			s = s + s
			i = i + 1
		}
		time.Sleep(time.Second * 1)
	}
}

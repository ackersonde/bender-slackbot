package main

import (
	"fmt"
	"log"
	"os/exec"
)

func main2() {
	out, err := exec.Command("/usr/bin/ssh-keygen", "-Lf", "/home/ackersond/.ssh/id_dan-cert.pub").Output()
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("%s\n", out)

}

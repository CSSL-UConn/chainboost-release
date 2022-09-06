package main

import (
	"os/exec"
	"time"

	"github.com/chainBoostScale/ChainBoost/onet/log"
)

func main() {
	result := make(chan string)
	var err error
	wait := 60 * time.Minute
	go func() {
		cmd := exec.Command("cd remote; ./users -kill=false")
		if err = cmd.Run(); err != nil {
			log.Lvl1("Couldn't run the simulation:", err)
		}
		result <- "finished"
	}()
	go func() {
		select {
		case msg := <-result:
			if msg == "finished" {
				log.LLvl1("simulation finished successfully!")
			}
			log.LLvl1("Received out-of-line message", msg)
		case <-time.After(wait):
			log.LLvl1("Quitting after waiting", wait)
		}

		if err := Cleanup(); err != nil {
			log.LLvl1("Couldn't cleanup platform:", err)
		}
	}()

}

func Cleanup() error {
	err := exec.Command("pkill", "-9", "users").Run()
	if err != nil {
		log.LLvl1("Error stopping ./users:", err)
	}
	var sshKill chan string
	sshKill = make(chan string)
	go func() {
		if _, err := SSHRun(d.Login, d.Host, "killall -9 users sshd"); err != nil {
			log.LLvl1("Error while cleaning up:", err)
		}

		err := SSHRunStdout(d.Login, d.Host, "test -f remote/users && ( cd remote; ./users -kill )")
		if err != nil {
			log.LLvl1("NOT-Normal error from cleanup", err.Error())
			sshKill <- "error"
		}
		sshKill <- "stopped"
	}()

	for {
		select {
		case msg := <-sshKill:
			if msg == "stopped" {
				log.LLvl1("Users stopped")
				return nil
			}
			log.LLvl1("Received other command", msg, "probably the app didn't quit correctly")
		case <-time.After(time.Second * 20):
			log.LLvl1("Timeout error when waiting for end of ssh")
			return nil
		}
	}
}

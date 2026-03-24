package sshtun

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"
)

func createSshTunnel(
	hostname string,
	username string,
	port int,
	index int,
	running chan<- int,
	failed chan<- int,
) error {
	var pid *int
	defer func() {
		if r := recover(); r != nil {
			var p int
			log.Printf("recovering: %s", r)
			if pid != nil {
				p = *pid
			} else {
				p = 0
			}
			failed <- p
		}
	}()

	cmd := exec.Command(
		"ssh",
		hostname,
		"-l",
		username,
		"-p",
		fmt.Sprintf("%d", port),
		"-w",
		fmt.Sprintf("%d:%d", index, index),
		"-N",
		"-T",
		"-o",
		"ServerAliveInterval 1",
		"-o",
		"ServerAliveCountMax 3",
		"-o",
		"ConnectTimeout 2",
	)
	var err error

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Start()

	if err != nil {
		panic(
			fmt.Sprintf(
				"process could not start: %s",
				err,
			),
		)
	}

	pid = &cmd.Process.Pid
	running <- *pid

	sErr, _ := io.ReadAll(stderr)
	sOut, _ := io.ReadAll(stdout)

	err = cmd.Wait()

	panic(
		fmt.Sprintf(
			"process %d has been terminated: %s, %s, %s",
			*pid,
			sOut,
			sErr,
			err,
		),
	)
}

func addLocalAddresses(
	iface string,
	addresses []string,
) (string, error) {
	for range 10 {
		cmd := exec.Command("ip", "link", "show", iface)
		if err := cmd.Run(); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	for _, address := range addresses {
		cmd := exec.Command("ip", "addr", "add", address, "dev", iface)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out), err
		}

		log.Printf("[localhost] %s %s", iface, address)
	}

	cmd := exec.Command("ip", "link", "set", iface, "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return string(out), err
	}

	log.Printf("[localhost] %s up", iface)
	return "", nil
}

func addRemoteAddresses(
	hostname string,
	username string,
	port int,
	iface string,
	addresses []string,
) (string, error) {
	for _, address := range addresses {
		cmd := exec.Command(
			"ssh",
			hostname,
			"-l",
			username,
			"-p",
			fmt.Sprintf("%d", port),
			"-o",
			"ConnectTimeout 2",
			fmt.Sprintf("ip addr add %s dev %s", address, iface),
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			return string(out), err
		}

		log.Printf("[%s] %s %s", hostname, iface, address)
	}

	cmd := exec.Command(
		"ssh",
		hostname,
		"-l",
		username,
		"-p",
		fmt.Sprintf("%d", port),
		fmt.Sprintf("ip link set %s up", iface),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return string(out), err
	}

	log.Printf("[%s] %s up", hostname, iface)
	return "", nil
}

func Create(
	hostname string,
	username string,
	port int,
	index int,
	remoteAddresses []string,
	localAddresses []string,
) {
	running := make(chan int)
	failed := make(chan int)

	go createSshTunnel(
		hostname,
		username,
		port,
		index,
		running,
		failed,
	)

	iface := fmt.Sprintf("tun%d", index)

	for {
		select {
		case p := <-running:
			log.Printf("ssh tunnel with pid %d started", p)

			if out, err := addLocalAddresses(
				iface,
				localAddresses,
			); err != nil {
				log.Printf("error: %s, output: %s", err, out)
				continue
			}

			if out, err := addRemoteAddresses(
				hostname,
				username,
				port,
				iface,
				remoteAddresses,
			); err != nil {
				log.Printf("error: %s, output: %s", err, out)
				continue
			}

		case p := <-failed:
			log.Printf("ssh tunnel with pid %d failed", p)

			go createSshTunnel(
				hostname,
				username,
				port,
				index,
				running,
				failed,
			)
		}
	}
}

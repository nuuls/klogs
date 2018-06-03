package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: klogs <pod-name>")
		fmt.Println("example: klogs api")
		os.Exit(1)
	}
	name := os.Args[1]
	pods, err := getPods(name)
	if err != nil {
		panic(err)
	}
	log.Printf("found %d pods with the name %s", len(pods), name)
	var wg sync.WaitGroup
	out := make(chan string, 1)
	for _, pod := range pods {
		wg.Add(1)
		go func(pod string) {
			defer wg.Done()
			err := streamLogs(pod, name, out)
			if err != nil {
				log.Printf("error streaming logs for %s: %v", pod, err)
			}
		}(pod)
	}
	go func() {
		for line := range out {
			fmt.Println(line)
		}
	}()
	wg.Wait()
}

func getPods(name string) ([]string, error) {
	cmd := exec.Command("kubectl", "get", "pods", "-o", "name")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, out)
	}
	s := string(out)
	pods := []string{}
	r := regexp.MustCompile(fmt.Sprintf(`pod\/(%s\-.+)`, regexp.QuoteMeta(name)))
	for _, name := range strings.Fields(s) {
		if matches := r.FindStringSubmatch(name); matches != nil {
			pods = append(pods, matches[1])
		}
	}
	return pods, nil
}

func streamLogs(pod, container string, out chan<- string) error {
	cmd := exec.Command("kubectl", "logs", pod, container, "-f")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	err = cmd.Start()
	if err != nil {
		return err
	}
	go readOutput(stdout, out)
	return readOutput(stderr, out)
}

func readOutput(reader io.Reader, out chan<- string) error {
	rd := bufio.NewReader(reader)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			return err
		}
		out <- line[:len(line)-1]
	}
}

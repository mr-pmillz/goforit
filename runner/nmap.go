package runner

import (
	"context"
	"fmt"
	"github.com/Ullaakut/nmap/v2"
	valid "github.com/asaskevich/govalidator"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// NmapStdoutStreamer is your custom type in code.
// You just have to make it a Streamer.
type NmapStdoutStreamer struct {
	nmap.Streamer
	File string
}

// Write is a function that handles the normal nmap stdout.
func (c *NmapStdoutStreamer) Write(d []byte) (int, error) {
	lines := string(d)

	if strings.Contains(lines, "Stats: ") {
		fmt.Print(lines)
	}
	return len(d), nil
}

// Bytes returns scan result bytes.
func (c *NmapStdoutStreamer) Bytes() []byte {
	data, err := os.ReadFile(c.File)
	if err != nil {
		data = append(data, "\ncould not read File"...)
	}
	return data
}

// streamNmap ...
func streamNmap(targets map[string][]string, outputDir string) error {
	if err := os.MkdirAll(fmt.Sprintf("%s/nmap", outputDir), os.ModePerm); err != nil {
		return err
	}
	tasks := make(chan *nmap.Run, len(targets))
	var wg sync.WaitGroup

	// Spawn 5 goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(num int, wg *sync.WaitGroup) {
			printNmapResults(tasks, wg)
		}(i, &wg)
	}

	for target, ports := range targets {
		switch {
		case len(ports) >= 100:
			fmt.Printf("Running nmap against %s\n", target)
		default:
			fmt.Printf("Running nmap against %s\t%+v", target, ports)
		}
		tasks <- runNmap(target, outputDir, ports)
	}

	close(tasks)
	wg.Wait()
	return nil
}

// printNmapResults ...
func printNmapResults(ch <-chan *nmap.Run, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range ch {
		// use result to format custom output
		for _, host := range task.Hosts {
			if len(host.Ports) == 0 || len(host.Addresses) == 0 {
				continue
			}

			fmt.Printf("Host %q:\n", host.Addresses[0])
			for _, port := range host.Ports {
				fmt.Printf("\tPort %d/%s %s %s\n", port.ID, port.Protocol, port.State, port.Service.Name)
				for _, script := range port.Scripts {
					fmt.Printf("%s\n", script.Output)
				}
			}
		}
	}
}

// runNmap runs StreamNmap against a target and slice of ports
func runNmap(target, outputDir string, ports []string) *nmap.Run {
	// limit each scan to maximum of 10 minutes in case something gets stuck..
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	xmlOutput := fmt.Sprintf("%s/nmap/%s_top_ports.xml", outputDir, target)
	nmapOutput := fmt.Sprintf("%s/nmap/%s_top_ports.nmap", outputDir, target)
	cType := &NmapStdoutStreamer{
		File: xmlOutput,
	}

	s, err := nmap.NewScanner(
		nmap.WithTargets(target),
		nmap.WithPorts(strings.Join(ports, ",")),
		nmap.WithNmapOutput(nmapOutput),
		nmap.WithAggressiveScan(),
		nmap.WithVerbosity(3),
		nmap.WithTimingTemplate(nmap.TimingAggressive),
		// Filter out hosts that don't have any open ports
		nmap.WithFilterHost(func(h nmap.Host) bool {
			// Filter out hosts with no open ports.
			for i := range h.Ports {
				if h.Ports[i].Status() == "open" {
					return true
				}
			}

			return false
		}),
		nmap.WithContext(ctx),
	)
	if valid.IsDNSName(target) {
		s.AddOptions(nmap.WithCustomArguments("--resolve-all"))
	}

	if err != nil {
		log.Printf("unable to create nmap scanner: %v", err)
	}

	warnings, err := s.RunWithStreamer(cType, cType.File)
	if err != nil {
		log.Printf("unable to run nmap scan: %v", err)
	}

	fmt.Printf("StreamNmap warnings: %v\n", warnings)

	result, err := nmap.Parse(cType.Bytes())
	if err != nil {
		log.Printf("unable to parse nmap output: %v", err)
	}
	return result
}

// runNmapAsync runs nmap concurrently with 10 goroutines in parallel.
func runNmapAsync(outputDir string, targets map[string][]string) error {
	if err := os.MkdirAll(fmt.Sprintf("%s/nmap", outputDir), os.ModePerm); err != nil {
		return err
	}
	fmt.Printf("Running nmap against %d hosts\n", len(targets))
	// Common channel for the goroutines
	tasks := make(chan *exec.Cmd, len(targets))

	var wg sync.WaitGroup

	// Spawn 10 goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(num int, w *sync.WaitGroup) {
			defer w.Done()
			var (
				out []byte
				err error
			)
			for cmd := range tasks {
				fmt.Printf("%s\n", strings.ReplaceAll(cmd.String(), "/usr/bin/bash -c ", ""))
				out, err = cmd.Output()
				if err != nil {
					fmt.Printf("can't get stdout: %+v", err)
				}
				fmt.Println(string(out))
			}
		}(i, &wg)
	}
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		log.Fatalf("could not get bash path: %v", err)
	}
	nmapPath, err := exec.LookPath("nmap")
	if err != nil {
		log.Fatalf("could not get nmap path: %v", err)
	}
	for target, ports := range targets {
		var command string
		if valid.IsDNSName(target) {
			command = fmt.Sprintf("sudo %s -vvv -Pn -p %s -sC -sV -oA %s/nmap/%s-top-ports --resolve-all %s", nmapPath, strings.Join(ports, ","), outputDir, target, target)
		} else {
			command = fmt.Sprintf("sudo %s -vvv -Pn -p %s -sC -sV -oA %s/nmap/%s-top-ports %s", nmapPath, strings.Join(ports, ","), outputDir, target, target)
		}
		tasks <- exec.Command(bashPath, "-c", command)
	}
	close(tasks)
	// wait for the workers to finish
	wg.Wait()
	return nil
}

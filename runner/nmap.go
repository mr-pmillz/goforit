package runner

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/Ullaakut/nmap/v2"
	valid "github.com/asaskevich/govalidator"
	"github.com/mr-pmillz/goforit/utils"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"sync"
	"syscall"
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
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error
	// Common channel for the goroutines
	tasks := make(chan *exec.Cmd, len(targets))
	var workers int
	if len(targets) < 10 {
		workers = len(targets)
	} else {
		workers = 10
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(num int, w *sync.WaitGroup) {
			defer w.Done()
			var (
				out []byte
				err error
			)
			for cmd := range tasks {
				fmt.Printf("%s\n", strings.ReplaceAll(cmd.String(), "/usr/bin/bash -c ", ""))
				out, err = cmd.CombinedOutput()
				if err != nil {
					log.Printf("Error executing command: %v", err)
					mu.Lock()
					errors = append(errors, fmt.Errorf("error executing command: %w", err))
					mu.Unlock()
					continue
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
		command := fmt.Sprintf("sudo %s -vvv -Pn --top-ports %d -T4 -sCV -oA %s/nmap/%s-top-ports %s", nmapPath, len(ports), outputDir, target, target)
		tasks <- exec.Command(bashPath, "-c", command)
	}
	close(tasks)
	// wait for the workers to finish
	wg.Wait()

	// Check for errors
	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors during execution", len(errors))
	}
	return nil
}

type NmapResults struct {
	Results []NmapRun
}

type NmapRun struct {
	XMLName          xml.Name `xml:"nmaprun"`
	Text             string   `xml:",chardata"`
	Scanner          string   `xml:"scanner,attr"`
	Args             string   `xml:"args,attr"`
	Start            string   `xml:"start,attr"`
	Startstr         string   `xml:"startstr,attr"`
	Version          string   `xml:"version,attr"`
	Xmloutputversion string   `xml:"xmloutputversion,attr"`
	Scaninfo         struct {
		Text        string `xml:",chardata"`
		Type        string `xml:"type,attr"`
		Protocol    string `xml:"protocol,attr"`
		Numservices string `xml:"numservices,attr"`
		Services    string `xml:"services,attr"`
	} `xml:"scaninfo"`
	Verbose struct {
		Text  string `xml:",chardata"`
		Level string `xml:"level,attr"`
	} `xml:"verbose"`
	Debugging struct {
		Text  string `xml:",chardata"`
		Level string `xml:"level,attr"`
	} `xml:"debugging"`
	Host struct {
		Text      string `xml:",chardata"`
		Starttime string `xml:"starttime,attr"`
		Endtime   string `xml:"endtime,attr"`
		Status    struct {
			Text   string `xml:",chardata"`
			State  string `xml:"state,attr"`
			Reason string `xml:"reason,attr"`
		} `xml:"status"`
		Address struct {
			Text     string `xml:",chardata"`
			Addr     string `xml:"addr,attr"`
			Addrtype string `xml:"addrtype,attr"`
		} `xml:"address"`
		Hostnames struct {
			Text     string `xml:",chardata"`
			Hostname []struct {
				Text string `xml:",chardata"`
				Name string `xml:"name,attr"`
				Type string `xml:"type,attr"`
			} `xml:"hostname"`
		} `xml:"hostnames"`
		Ports struct {
			Text       string `xml:",chardata"`
			Extraports struct {
				Text         string `xml:",chardata"`
				State        string `xml:"state,attr"`
				Count        string `xml:"count,attr"`
				Extrareasons struct {
					Text   string `xml:",chardata"`
					Reason string `xml:"reason,attr"`
					Count  string `xml:"count,attr"`
				} `xml:"extrareasons"`
			} `xml:"extraports"`
			Port []struct {
				Text     string `xml:",chardata"`
				Protocol string `xml:"protocol,attr"`
				Portid   string `xml:"portid,attr"`
				State    struct {
					Text      string `xml:",chardata"`
					State     string `xml:"state,attr"`
					Reason    string `xml:"reason,attr"`
					ReasonTTL string `xml:"reason_ttl,attr"`
				} `xml:"state"`
				Service struct {
					Text      string   `xml:",chardata"`
					Name      string   `xml:"name,attr"`
					Product   string   `xml:"product,attr"`
					Version   string   `xml:"version,attr"`
					Extrainfo string   `xml:"extrainfo,attr"`
					Ostype    string   `xml:"ostype,attr"`
					Method    string   `xml:"method,attr"`
					Conf      string   `xml:"conf,attr"`
					Cpe       []string `xml:"cpe"`
				} `xml:"service"`
				Script struct {
					Text   string `xml:",chardata"`
					ID     string `xml:"id,attr"`
					Output string `xml:"output,attr"`
				} `xml:"script"`
			} `xml:"port"`
		} `xml:"ports"`
		Os struct {
			Text     string `xml:",chardata"`
			Portused []struct {
				Text   string `xml:",chardata"`
				State  string `xml:"state,attr"`
				Proto  string `xml:"proto,attr"`
				Portid string `xml:"portid,attr"`
			} `xml:"portused"`
			Osclass struct {
				Text     string `xml:",chardata"`
				Type     string `xml:"type,attr"`
				Vendor   string `xml:"vendor,attr"`
				Osfamily string `xml:"osfamily,attr"`
				Osgen    string `xml:"osgen,attr"`
				Accuracy string `xml:"accuracy,attr"`
				Cpe      string `xml:"cpe"`
			} `xml:"osclass"`
			Osmatch struct {
				Text     string `xml:",chardata"`
				Name     string `xml:"name,attr"`
				Accuracy string `xml:"accuracy,attr"`
				Line     string `xml:"line,attr"`
			} `xml:"osmatch"`
		} `xml:"os"`
		Uptime struct {
			Text     string `xml:",chardata"`
			Seconds  string `xml:"seconds,attr"`
			Lastboot string `xml:"lastboot,attr"`
		} `xml:"uptime"`
		Distance struct {
			Text  string `xml:",chardata"`
			Value string `xml:"value,attr"`
		} `xml:"distance"`
		Tcpsequence struct {
			Text       string `xml:",chardata"`
			Index      string `xml:"index,attr"`
			Difficulty string `xml:"difficulty,attr"`
			Values     string `xml:"values,attr"`
		} `xml:"tcpsequence"`
		Ipidsequence struct {
			Text   string `xml:",chardata"`
			Class  string `xml:"class,attr"`
			Values string `xml:"values,attr"`
		} `xml:"ipidsequence"`
		Tcptssequence struct {
			Text   string `xml:",chardata"`
			Class  string `xml:"class,attr"`
			Values string `xml:"values,attr"`
		} `xml:"tcptssequence"`
		Trace struct {
			Text  string `xml:",chardata"`
			Port  string `xml:"port,attr"`
			Proto string `xml:"proto,attr"`
			Hop   []struct {
				Text   string `xml:",chardata"`
				TTL    string `xml:"ttl,attr"`
				Ipaddr string `xml:"ipaddr,attr"`
				Rtt    string `xml:"rtt,attr"`
				Host   string `xml:"host,attr"`
			} `xml:"hop"`
		} `xml:"trace"`
		Times struct {
			Text   string `xml:",chardata"`
			Srtt   string `xml:"srtt,attr"`
			Rttvar string `xml:"rttvar,attr"`
			To     string `xml:"to,attr"`
		} `xml:"times"`
	} `xml:"host"`
	Runstats struct {
		Text     string `xml:",chardata"`
		Finished struct {
			Text    string `xml:",chardata"`
			Time    string `xml:"time,attr"`
			Timestr string `xml:"timestr,attr"`
			Elapsed string `xml:"elapsed,attr"`
			Summary string `xml:"summary,attr"`
			Exit    string `xml:"exit,attr"`
		} `xml:"finished"`
		Hosts struct {
			Text  string `xml:",chardata"`
			Up    string `xml:"up,attr"`
			Down  string `xml:"down,attr"`
			Total string `xml:"total,attr"`
		} `xml:"hosts"`
	} `xml:"runstats"`
}

// parseNmapResults ...
func parseNmapResults(outputDir string) (*NmapResults, error) {
	files, err := utils.FilePathWalkDir(outputDir)
	if err != nil {
		return nil, err
	}
	if err = modifyFilePermissions(files); err != nil {
		return nil, err
	}

	var nmapXMLFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".xml") {
			nmapXMLFiles = append(nmapXMLFiles, f)
		}
	}
	nmapResults, err := getNmapData(nmapXMLFiles)
	if err != nil {
		return nil, err
	}

	return nmapResults, nil
}

// getNmapData ...
func getNmapData(nmapFiles []string) (*NmapResults, error) {
	nmapResults := &NmapResults{}
	for _, nmapFile := range nmapFiles {
		results, err := parseNmapFile(nmapFile)
		if err != nil {
			return nil, err
		}
		nmapResults.Results = append(nmapResults.Results, *results)
	}

	return nmapResults, nil
}

// parseNmapFile ...
func parseNmapFile(nmapFile string) (*NmapRun, error) {
	results := &NmapRun{}
	f, err := os.OpenFile(nmapFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if err = xml.Unmarshal(data, results); err != nil {
		return nil, err
	}
	return results, nil
}

// modifyFilePermissions ...
func modifyFilePermissions(filePaths []string) error {
	// Get the current user
	currentUser, err := user.Current()
	if err != nil {
		return err
	}

	// Check if the current user is root
	if currentUser.Username == "root" {
		return nil
	}

	// Get the user ID and group ID of the current user
	uid := currentUser.Uid
	gid := currentUser.Gid

	// Iterate over the file paths
	for _, path := range filePaths {
		// Get the file info
		fileInfo, err := os.Stat(path)
		if err != nil {
			return err
		}

		// Change the file owner if necessary
		fileOwner := fileInfo.Sys().(*syscall.Stat_t)
		if fileOwner.Uid == 0 {
			cmd := exec.Command("sudo", "chown", fmt.Sprintf("%s:%s", uid, gid), path) //nolint:gosec
			if err = cmd.Run(); err != nil {
				return err
			}
		}
	}

	return nil
}

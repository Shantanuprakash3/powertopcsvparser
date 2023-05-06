package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type SysPower struct {
	ProcessConsumers []ProcessConsumer
	DeviceConsumers  []DeviceConsumer
}

type ProcessConsumer struct {
	Pid             int     `json:"pid"`
	Usage           float64 `json:"usage"`
	DiskioPerSecond float64 `json:"diskIoPerSecond"`
	Category        string  `json:"category"`
	Description     string  `json:"description"`
	PwEstimate      float64 `json:"pwEstimate"`
}

type DeviceConsumer struct {
	Usage      string `json:"usage"`
	DeviceName string `json:"deviceName"`
}
func main() {
	fmt.Println("per process power consumption: ", GetSysPower())
}
func GetSysPower() SysPower {
	sysPowerObj := SysPower{}
  prg := "powertop"
	arg1 := "--csv=/tmp/temp.csv"
  cmd := exec.Command(prg, arg1)
  stdout, err := cmd.Output()
	fmt.Printf("power usage: %v\n", cmd)

	f, err := os.Open(tempPath)
	if err != nil {
		fmt.Println("Could not read /tmp/temp.csv powertop output\n", err)
		return sysPowerObj
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	csvReader.Comma = ';'
	csvReader.Comment = '#'
	csvReader.FieldsPerRecord = -1

	data, err := csvReader.ReadAll()

	if (err!= nil){
		fmt.Println("Error reading powertop data from csv", err)
		return sysPowerObj
	}

	sections := splitSections(data)

	sysPowerObj.ProcessConsumers = addProcessConsumers(sections["Overview of Software Power Consumers"])
	sysPowerObj.DeviceConsumers = addDeviceConsumers(sections["Device Power Report"])

	return sysPowerObj
}

func splitSections(data [][]string) map[string][][]string {
	currentSection:=[][]string{}
	var sectionHeader = ""
	var sections = map[string][][]string{}

	for _, line := range data {
		match, err := regexp.MatchString("^(.)\\_+$", line[0])
		if (err!=nil){
			fmt.Println("Error running regex for section separator")
		}
		if match {
			if len(currentSection) > 0 {
				if sectionHeader != "P o w e r T O P" {

					sections[sectionHeader] = currentSection
					sectionHeader = ""
					currentSection = nil
				}
			}
		} else if len(sectionHeader) == 0 {
			sectionHeader = strings.Trim(line[0], " *")
		} else {
			for i := range line {
				line[i] = strings.TrimSpace(line[i])
			}
			if strings.TrimSpace(strings.Join(line, "")) != "" {
				currentSection = append(currentSection, line)
			}
		}
	}
	return sections
}

func addProcessConsumers(data [][]string) []ProcessConsumer {
	ProcessConsumers := make([]ProcessConsumer, 0, len(data)-1) 
	for _, line := range data[1:] {
		var pc ProcessConsumer
		for j, field := range line {
			if j == 0 {
				pc.Usage = convertUsageToMsPerSecond(field)
			} else if j == 4 {
				pc.DiskioPerSecond, _ = strconv.ParseFloat(field, 64)
			} else if j == 5 {
				pc.Category = field
			} else if j == 6 {
				pc.Pid = extractPidFromString(field)
				if pc.Category == "Process" {
					pc.Description = extractCmdName(field)
				} else {
					pc.Description = field
				}
			} else if j == 7 {
				pc.PwEstimate = extractPwInWatts(field)
			}
		}
		ProcessConsumers = append(ProcessConsumers, pc)
	}
	return ProcessConsumers
}

func addDeviceConsumers(data [][]string) []DeviceConsumer {
	var DeviceConsumers []DeviceConsumer
	for _, line := range data[1:] {
		var dc DeviceConsumer
		dc.Usage = line[0]
		dc.DeviceName = line[1]
		DeviceConsumers = append(DeviceConsumers, dc)
	}
	return DeviceConsumers
}

func extractPidFromString(field string) int {

	var pid int
	re := regexp.MustCompile(`\[(.*?)\]`)
	match := re.FindStringSubmatch(field)
	if len(match) == 2 {
		str := strings.Split(match[len(match)-1], " ")
		pid, _ = strconv.Atoi(str[len(str)-1])
	}
	return pid
}

func extractCmdName(field string) string {
	str := strings.Split(strings.TrimSpace(field), " ")
	if len(str) == 1 {
		return str[0]
	}
	if len(str) > 2 {
		return path.Base(str[2])
	}
	return field
}

func extractPwInWatts(field string) float64 {
	str := strings.Split(field, " ")
	powerEstimate, _ := strconv.ParseFloat(str[0], 64)

	powerwUnits := str[1]
	if powerwUnits == "mW" {
		powerEstimate = powerEstimate / 1000
	} else if powerwUnits == "uW"{
		powerEstimate = powerEstimate / 1000000
	}
	return powerEstimate
}

func convertUsageToMsPerSecond(field string) float64 {
	str := strings.Split(field, " ")
	usage, _ := strconv.ParseFloat(str[0], 64)

	usageUnits := str[1]
	if usageUnits == "s/s" {
		usage = usage * 1000
	} else if usageUnits == "us/s" {
		usage = usage / 1000
	}

	return usage
}

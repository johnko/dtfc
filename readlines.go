package main

import (
	"bufio"
	"os"
	"strings"
)

//from http://stackoverflow.com/questions/5884154/golang-read-text-file-into-string-array-and-write
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	var tmpstr string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		tmpstr = strings.TrimSpace(scanner.Text())
		if tmpstr != "" {
			lines = append(lines, tmpstr)
		}
	}
	return lines, scanner.Err()
}

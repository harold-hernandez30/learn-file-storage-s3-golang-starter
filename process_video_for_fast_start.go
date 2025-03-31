package main

import (
	"fmt"
	"os/exec"
)

func processVideoForFastStart(filepath string) (string, error) {
	outputFilepath := fmt.Sprintf("%s.processing", filepath)
	cmd := exec.Command("ffmpeg", "-i", filepath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilepath)
	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("unable to execute command. input file path: %s\n outputfile path: %s \nError: %s\n command: %s", filepath, outputFilepath, err, cmd.String())
	}

	return outputFilepath, nil
}
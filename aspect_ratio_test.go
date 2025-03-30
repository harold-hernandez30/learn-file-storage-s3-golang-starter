package main

import (
	"fmt"
	"testing"
)


func TestAspectRatio_Landscape(t *testing.T) {

	expected := "16:9"
	input_width := 1920
	input_height := 1080
	result := GetVideoAspectRatio(input_width, input_height)

	if result == expected {
		fmt.Println("PASSED!")
	} else {
		t.Errorf("Expected: %s\n Received: %s\n", expected, result)
	}
	
}


func TestAspectRatio_Portrait(t *testing.T) {

	expected := "9:16"
	input_height := 1920
	input_width := 1080
	result := GetVideoAspectRatio(input_width, input_height)

	if result == expected {
		fmt.Println("PASSED!")
	} else {
		t.Errorf("Expected: %s\n Received: %s\n", expected, result)
	}
	
}



func TestAspectRatio_Other(t *testing.T) {

	expected := "other"
	input_height := 1920
	input_width := 1920
	result := GetVideoAspectRatio(input_width, input_height)

	if result == expected {
		fmt.Println("PASSED!")
	} else {
		t.Errorf("Expected: %s\n Received: %s\n", expected, result)
	}
	
}


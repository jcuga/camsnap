package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	fmt.Println("Attempting file conversion")
	data, err := ioutil.ReadFile("frame.yuv")
	if err != nil {
		log.Fatalf("Failed reading file: %v", err)
	}
	fmt.Printf("File had %d bytes\n", len(data))
	if len(data)%4 != 0 {
		log.Fatalf("File bytes len not 4 byte chunks.")
	}
	numChunks := len(data) / 4
	fmt.Printf("File has %d many 4-byte chunks.\n", numChunks)

	// NOTE: since 422 ratio (hard coded assumption thats our file btw!)
	// the len y = 1/2 of all bytes, cb and cr are 1/4 each
	yBytes := make([]byte, 0)
	cbBytes := make([]byte, 0)
	crBytes := make([]byte, 0)

	for index, value := range data {
		// Byte format is for each 4 byte chunk
		// Y0,U0,Y1,V1 ... Y2,U2,Y3,V2, ... Y4,Y4,Y5,V5 etc...
		switch index % 4 {
		case 0:
			fallthrough
		case 2:
			// Y values
			yBytes = append(yBytes, value)
		case 1:
			// U values
			cbBytes = append(cbBytes, value) // TODO: U == cb? think so...
		case 3:
			// V values
			crBytes = append(crBytes, value) // TODO: V == cr? think so...
		}

	}

	// TODO: check and log fatal if len y is not double len cb and cr (those are equal)

	r := image.Rect(0, 0, 640, 480) // TODO: off by one? should be 639 and 479 since zeros included?
	// NOTE: reverse engineered ctor values from image package
	// especially the stride values.  hard coded assumption of 640x480 image with 422 ratio!
	// (which should be what the webcame im testing with uses...)
	ycbcrImg := &image.YCbCr{
		Y:              yBytes,
		Cb:             cbBytes,
		Cr:             crBytes,
		SubsampleRatio: image.YCbCrSubsampleRatio422,
		YStride:        640,
		CStride:        320,
		Rect:           r,
	}

	// TODO: try saving to png
	// TODO: try saving to jpeg

	out, err := os.Create("./output.jpeg")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	options := &jpeg.Options{Quality: 90}
	jpeg.Encode(out, ycbcrImg, options)

	fmt.Printf("Done")
	// TODO: if this works, post helpful links (on byte format, the gov and the fourcc sites) on github
	// TODO: more polished version?
	// TODO: program that tries all possible types? or guesses/determines?
}

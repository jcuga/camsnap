package main

import (
	"flag"
	"fmt"
	"github.com/blackjack/webcam"
	"image"
	"image/jpeg"
	"log"
	"os"
	"sort"
	"time"
)

func main() {
	videoDev := flag.String("cam", "/dev/video0", "Linux /dev/ for camera.")
	outFile := flag.String("file", "./camera_frame.jpeg", "Output image file location.")
	onlyOnce := flag.Bool("once", false, "Only take one image capture and exit.")
	overwriteLast := flag.Bool("overwrite", false, "Overwrite previous image capture. If not overwriting, images will have 'file' filename plus timestamp.")
	updatePeriodSecPtr := flag.Uint("upsec", 10, "How many seconds between updates.")
	flag.Parse()
	if *updatePeriodSecPtr <= 0 {
		fmt.Printf("Invalid upsec param: %v.  Must be > 0\n", *updatePeriodSecPtr)
		os.Exit(1)
	}
	log.Printf("Starting with cam: %v, file: %v, overwrite: %v, upsec: %v, once: %v\n",
		*videoDev, *outFile, *overwriteLast, *updatePeriodSecPtr, *onlyOnce)
	// take picture immediately
	if err := camsnap(*videoDev, *outFile, *updatePeriodSecPtr, *overwriteLast, *onlyOnce); err != nil {
		log.Fatalf("Error setting up image capture: %v", err)
	}
	log.Printf("All done.")
}

func camsnap(videoDev, outFile string, updatePeriodeSec uint, overwriteLast, onlyOnce bool) error {
	cam, err := webcam.Open(videoDev)
	if err != nil {
		return err
	}
	defer cam.Close()
	format_desc := cam.GetSupportedFormats()
	var formats []webcam.PixelFormat
	for f := range format_desc {
		formats = append(formats, f)
	}

	println("Available formats: ")
	for i, value := range formats {
		fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, format_desc[value])
	}

	choice := readChoice(fmt.Sprintf("Choose format [1-%d]: ", len(formats)))
	format := formats[choice-1]

	fmt.Fprintf(os.Stderr, "Supported frame sizes for format %s\n", format_desc[format])
	frames := FrameSizes(cam.GetSupportedFrameSizes(format))
	sort.Sort(frames)

	for i, value := range frames {
		fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, value.GetString())
	}
	choice = readChoice(fmt.Sprintf("Choose format [1-%d]: ", len(frames)))
	size := frames[choice-1]

	f, w, h, err := cam.SetImageFormat(format, uint32(size.MaxWidth), uint32(size.MaxHeight))

	if err != nil {
		return err
	} else {
		fmt.Fprintf(os.Stderr, "Resulting image format: %s (%dx%d)\n", format_desc[f], w, h)
	}

	// TODO: looks like control C breaks the subsequent attempts because
	// camera isn't closed i guess.
	// TODO: add signal handler and have camera global to clean up...
	// TODO: or see if any other way to fix this.
	// TODO: workaround, unplug and replug camera (if usb/not built in...)
	// TODO: confirm Control C prevents defer from firing

	err = cam.StartStreaming()
	if err != nil {
		return err
	}
	timeout := uint32(3) // 3 seconds
	for {
		err = cam.WaitForFrame(timeout)
		switch err.(type) {
		case nil:
		case *webcam.Timeout:
			log.Printf("Timeout waiting for frame: ", err)
			continue
		default:
			return err
		}
		frame, err := cam.ReadFrame()
		if len(frame) != 0 {
			fmt.Printf("Got frame! with len: %v\n", len(frame))
			img, imgErr := toImage(frame)
			if imgErr == nil {

				fileName := outFile
				if !overwriteLast {
					// save with timestamp appended to desired name
					fileName += "__" + time.Now().Format(time.RFC3339)
				}

				out, err := os.Create(fileName)
				if err != nil {
					log.Printf("ERROR trying to create output file.  filename: %v, error: %v", fileName, err)
				} else {
					defer out.Close()
					options := &jpeg.Options{Quality: 90}
					jpeg.Encode(out, img, options)
				}

			} else {
				log.Printf("ERROR trying to turn frame bytes into YCbCr image: %v.", imgErr)
			}

		} else if err != nil {
			log.Printf("ERROR reading frame: %v", err)
		}

		if onlyOnce {
			log.Printf("Finished")
			return nil
		} else {
			// wait until next time
			select {
			case <-time.After(time.Second * time.Duration(updatePeriodeSec)):
				// Nothing to do, just introducing delay...
			}
		}
	}
	return nil
}

func toImage(frame []byte) (*image.YCbCr, error) {
	if len(frame)%4 != 0 {
		return nil, fmt.Errorf("Frame len was not divisible by 4.  Got len: %v", len(frame))
	}
	// NOTE: since 422 ratio (hard coded assumption thats our file btw!)
	// the len y = 1/2 of all bytes, cb and cr are 1/4 each
	yBytes := make([]byte, 0)
	cbBytes := make([]byte, 0)
	crBytes := make([]byte, 0)
	for index, value := range frame {
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
	return ycbcrImg, nil
}

// ---------- Stuff below here was from blackjack/webcam example: ------------

func readChoice(s string) int {
	var i int
	for true {
		print(s)
		_, err := fmt.Scanf("%d\n", &i)
		if err != nil || i < 1 {
			println("Invalid input. Try again")
		} else {
			break
		}
	}
	return i
}

type FrameSizes []webcam.FrameSize

func (slice FrameSizes) Len() int {
	return len(slice)
}

//For sorting purposes

func (slice FrameSizes) Less(i, j int) bool {
	ls := slice[i].MaxWidth * slice[i].MaxHeight
	rs := slice[j].MaxWidth * slice[j].MaxHeight
	return ls < rs
}

//For sorting purposes
func (slice FrameSizes) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

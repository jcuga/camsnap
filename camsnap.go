package main

import (
	"flag"
	"fmt"
	"github.com/blackjack/webcam"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"
)

func main() {
	videoDev := flag.String("cam", "/dev/video0", "Linux /dev/ for camera.")
	outFile := flag.String("file", "./camera.png", "Output image file location.")
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

func camsnap(videoDev, outFile string, updatePeriodeSec uint, verwriteLast, onlyOnce bool) error {
	log.Printf("DEBUG 1") // TODO: remove
	cam, err := webcam.Open(videoDev)
	if err != nil {
		return err
	}
	defer cam.Close()
	log.Printf("DEBUG 2") // TODO: remove
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

			// NOTE: writing frame bytes to file for now, need to inspect and figure out how to decode
			fileName := "./frame.yuv" // TODO: use outfile instead?
			if err := ioutil.WriteFile(fileName, frame, 0644); err == nil {
				log.Printf("Saved to file: %v", fileName)
			} else {
				log.Fatalf("Failed to save frame to file. error: %v", err)
			}

			// TODO: place this in own func that saves to file with optional overwrite

			// NOTE: we have a byte[] of an image in a format.
			// TODO: logic to take format A and convert to format B for file writing

			// TODO: for now, hard coded assuming YUYV since my example uses that

			// TODO: sample ratio needs to be gleamed from camera as well (it gets printed out )

		} else if err != nil {
			log.Printf("Error reading frame: %v", err)
			return err
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

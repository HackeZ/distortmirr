package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hackez/distortmirr/mirror"
)

var (
	pkgFlag    = flag.String("pkg", "", "the package name of Go style")
	scanFlag   = flag.Int("scan", int(mirror.ScanPublic), "the scan mode of distorting mirror")
	modeFlag   = flag.String("mode", "monet", "the render mode of distorting mirror")
	outputFlag = flag.String("output", "", "the path of output file")
)

func init() {
	flag.Parse()
}

func main() {
	mirr, err := mirror.New(*pkgFlag, mirror.ScanMode(*scanFlag))
	if err != nil {
		fmt.Println("failed to new mirror: ", err)
		os.Exit(-1)
	}

	err = mirr.Scan()
	if err != nil {
		fmt.Println("failed to scan package: ", err)
		os.Exit(-1)
	}

	if *outputFlag != "" {
		f, err := os.Create(*outputFlag)
		if err != nil {
			panic(err)
		}
		err = mirr.Render(*modeFlag, f)
	} else {
		err = mirr.Render(*modeFlag, os.Stdout)
	}

	if err != nil {
		fmt.Println("failed to render package: ", err)
		os.Exit(-1)
	}
}

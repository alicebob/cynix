package main

import (
	"flag"
	"fmt"
)

var (
	flagRepo  = flag.String("repo", "https://github.com/alicebob/cynix", "URL to github repository")
	flagToken = flag.String("token", "", "registration token")
	flagPAT   = flag.String("pat", "", "personal access token")
	flagName  = flag.String("name", "cynix", "runner name")
)

func main() {
	fmt.Printf("connecting to %s\n", *flagRepo)

}

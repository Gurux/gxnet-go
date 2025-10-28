package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Gurux/gxcommon-go"
	"github.com/Gurux/gxnet-go"
	"golang.org/x/text/language"
)

var (
	host    = flag.String("h", "", "Host name")
	port    = flag.Int("p", 0, "Host port")
	message = flag.String("m", "", "Send message")
	t       = flag.String("t", "", "Trace level.")
	w       = flag.Int("w", 1000, "WaitTime in milliseconds.")
	lang    = flag.String("lang", "", "Used language.")
)

func CurrentLanguage() language.Tag {
	langEnv := os.Getenv("LANG")
	if langEnv == "" {
		return language.AmericanEnglish
	}
	langEnv = strings.Split(langEnv, ".")[0]
	tag, err := language.Parse(langEnv)
	if err != nil {
		return language.AmericanEnglish
	}
	return tag
}

func main() {
	flag.Parse()
	if *host == "" || *port == 0 || *message == "" {
		flag.PrintDefaults()
		return
	}

	media := gxnet.NewGXNet(gxnet.NetworkTypeTCP, *host, *port)
	if *lang != "" {
		tag, err := language.Parse(*lang)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error parsing language:", err)
			return
		}
		media.Localize(tag)
	}

	media.SetOnError(func(m gxcommon.IGXMedia, err error) {
		// log/handle error
		fmt.Fprintln(os.Stderr, "error:", err)
	})

	media.SetOnReceived(func(m gxcommon.IGXMedia, e gxcommon.ReceiveEventArgs) {
		fmt.Printf("Async data: %s\n", e.String())
	})

	media.SetOnMediaStateChange(func(m gxcommon.IGXMedia, e gxcommon.MediaStateEventArgs) {
		fmt.Printf("Media state change : %s\n", e.State().String())
	})

	media.SetOnTrace(func(m gxcommon.IGXMedia, e gxcommon.TraceEventArgs) {
		fmt.Printf("Trace: %s\n", e.String())
	})

	err := media.Validate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}

	if *t != "" {
		tl, err := gxcommon.TraceLevelParse(*t)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return
		}
		err = media.SetTrace(tl)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return
		}
	}
	if *t == "" {
		fmt.Printf("Trace level, %s!\n", *t)
	}
	fmt.Printf("Host name: %s\n", *host)
	fmt.Printf("Host port: %d\n", *port)
	fmt.Printf("Message: '%s'\n", *message)
	fmt.Printf("Trace level %s\n", media.GetTrace().String())

	err = media.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error returned:", err)
		return
	}
	//Close the connection.
	defer func() {
		if err := media.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close failed:", err)
		}
	}()

	//Send data synchronously.
	//Use defer media.GetSynchronous()() if sync is end when the method ends.
	//Or call media.GetSynchronous() when sync is needed and
	//call the returned function when sync is not needed anymore.
	func() {
		defer media.GetSynchronous()()
		err = media.Send(*message, "")
		//Send EOP
		err = media.Send("\n", "")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return
		}
		r := gxcommon.NewReceiveParameters[string]()
		r.EOP = "\n"
		r.WaitTime = *w
		r.Count = 0
		ret, err := media.Receive(r)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error returned:", err)
			return
		}
		if ret {
			fmt.Printf("Sync data: %s\n", r.Reply)
		}
	}()
	fmt.Printf("Exit\n")
}

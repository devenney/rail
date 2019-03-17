/*
Package main provides command-line usage of the module
*/
package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/go-stomp/stomp"
	"github.com/spf13/viper"

	"compress/gzip"
	"log"
)

// Message represents a top-level STOMP queue message
type Message struct {
	XMLNS     string `xml:"xmlns,attr"`
	XMLNS2    string `xml:"xmlns ns2,attr"`
	XMLNS3    string `xml:"xmlns ns3,attr"`
	Timestamp string `xml:"ts,attr"`
	Version   string `xml:"version,attr"`

	UR UniqueResponse `xml:"uR"`
}

// String provides a user-friendly representation of a Message
func (r Message) String() string {
	s := fmt.Sprintf("[%s v%s]:", r.Timestamp, r.Version)

	if r.UR.UO != "" {
		s = fmt.Sprintf("%s\n\t%s", s, r.UR)
	}

	return s
}

// UniqueResponse represents an XML response element
type UniqueResponse struct {
	UO string    `xml:"updateOrigin,attr"`
	TS Timestamp `xml:"TS"`
}

// String provides a user-friendly representation of a UniqueResponse
func (ur UniqueResponse) String() string {
	s := fmt.Sprintf("\nUpdate Origin: %s\n\n%s", ur.UO, ur.TS)

	return s
}

// Timestamp represents an XML timestamp element
type Timestamp struct {
	RID string `xml:"rid,attr"`
	SSD string `xml:"ssd,attr"`
	UID string `xml:"uid,attr"`

	Location []Location `xml:"Location"`
}

// String provides a user-friendly representation of a Timestamp
func (t Timestamp) String() string {
	var s string

	if t.RID != "" {
		s = fmt.Sprintf("RID: %s ", t.RID)
	}

	if t.SSD != "" {
		s = fmt.Sprintf("SSD: %s ", t.SSD)
	}

	if t.UID != "" {
		s = fmt.Sprintf("UID: %s ", t.UID)
	}

	if t.Location != nil {
		for _, l := range t.Location {
			s = fmt.Sprintf("%s\n%s", s, l)
		}
	}

	return s
}

// Location represents an XML location element
type Location struct {
	TPL string `xml:"tpl,attr"`

	PTA string `xml:"pta,attr"`
	PTD string `xml:"ptd,attr"`
	WTA string `xml:"wta,attr"`
	WTD string `xml:"wtd,attr"`
	WTP string `xml:"wtp,attr"`

	Arrival   Event `xml:"arr"`
	Departure Event `xml:"dep"`
	Pass      Event `xml:"pas"`
}

// String provides a user-friendly representation of a Timestamp
func (l Location) String() string {
	s := fmt.Sprintf("\n-- %s", l.TPL)

	if l.PTA != "" {
		s = fmt.Sprintf("%s | Public Time Arrive: %s", s, l.PTA)
	}

	if l.PTD != "" {
		s = fmt.Sprintf("%s | Public Time Depart: %s", s, l.PTD)
	}

	if l.WTA != "" {
		s = fmt.Sprintf("%s | Working Time Arrive: %s", s, l.WTA)
	}

	if l.WTD != "" {
		s = fmt.Sprintf("%s | Working Time Depart: %s", s, l.WTD)
	}

	if l.WTP != "" {
		s = fmt.Sprintf("%s | Working Time Pass: %s", s, l.WTP)
	}

	if l.Arrival != (Event{}) {
		s = fmt.Sprintf("%s\n\t** Arr: %s", s, l.Arrival)
	}

	if l.Departure != (Event{}) {
		s = fmt.Sprintf("%s\n\t** Dep: %s", s, l.Departure)
	}

	if l.Pass != (Event{}) {
		s = fmt.Sprintf("%s\n\t** Pass: %s", s, l.Pass)
	}

	if l.Delay() > 0 {
		s = fmt.Sprintf("%s\nDELAY: %f", s, l.Delay())
	}

	s += "\n"

	return s
}

// Delay checks for delayed arrival
func (l Location) Delay() (d float64) {
	if l.Arrival != (Event{}) {
		if l.Arrival.AT == "" {
			return
		}

		var form string
		if len(l.Arrival.AT) == 5 {
			form = "15:04"
		} else {
			form = "15:04:00"
		}

		actual, err := time.Parse(form, l.Arrival.AT)
		if err != nil {
			log.Fatal(err)
		}

		var sched string
		if l.PTA != "" {
			sched = l.PTA
		} else if l.WTA != "" {
			sched = l.WTA
		} else {
			return
		}

		if len(sched) == 5 {
			form = "15:04"
		} else {
			form = "15:04:05"
		}

		est, err := time.Parse(form, sched)
		if err != nil {
			log.Fatal(err)
		}

		delta := actual.Sub(est)

		d = delta.Minutes()
	}

	return
}

// Event represents an XML location event entity
type Event struct {
	AT  string `xml:"at,attr"`
	ET  string `xml:"et,attr"`
	SRC string `xml:"src,attr"`
}

func (e Event) String() string {
	var s string

	if e.AT != "" {
		s = fmt.Sprintf("ACTUAL %s ", e.AT)
	}

	if e.ET != "" {
		s = fmt.Sprintf("%sESTIMATED %s ", s, e.ET)
	}

	if e.SRC != "" {
		s = fmt.Sprintf("%s (Source: %s)", s, e.SRC)
	}

	return s
}

// main
func main() {
	v := viper.New()

	v.SetEnvPrefix("rail")
	v.AutomaticEnv()

	log.Print("Connecting to feed...")
	conn, err := stomp.Dial("tcp", "datafeeds.nationalrail.co.uk:61613", stomp.ConnOpt.Login("d3user", "d3password"))
	if err != nil {
		log.Fatal(err)
	}

	if v.GetString("queue_name") == "" {
		log.Fatal("RAIL_QUEUE_NAME is not set. Cannot recover.")
	}

	log.Print("Subscribing to queue...")
	sub, err := conn.Subscribe(v.GetString("queue_name"), stomp.AckClient)
	if err != nil {
		log.Fatal(err)
	}

	for start := time.Now(); time.Since(start) < 600*time.Second; {
		log.Print("Waiting for message...")
		msg := <-sub.C
		if msg.Err != nil {
			log.Fatal(msg.Err)
		}
		log.Println("Got new message.")

		log.Print("Decompressing body...")
		g, _ := gzip.NewReader(bytes.NewBuffer(msg.Body))
		defer g.Close()

		var b bytes.Buffer
		_, err = b.ReadFrom(g)
		if err != nil {
			log.Fatal(err)
		}

		log.Print("Unmarshalling XML...")
		var m Message
		err = xml.Unmarshal(b.Bytes(), &m)
		if err != nil {
			log.Fatal(err)
		}

		log.Print(b.String())

		log.Println("---------------")

		log.Printf("%s\n\n", m)

		_ = conn.Ack(msg)
	}

	err = sub.Unsubscribe()
	if err != nil {
		log.Fatal(err)
	}

	conn.Disconnect()
}

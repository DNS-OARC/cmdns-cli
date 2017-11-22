package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "os/signal"
    "strings"
    "time"

    "github.com/gorilla/websocket"
)

type CheckMsg struct {
    All bool `json:"all"`
}

type PrepareMsg struct {
    Done  bool `json:"done"`
    Total int  `json:"total"`

    Id          string `json:"id,omitempty"`
    Name        string `json:"name,omitempty"`
    Description string `json:"desc,omitempty"`
    Category    string `json:"cat,omitempty"`
    Score       int    `json:"score,omitempty"`
    Meta        bool   `json:"meta,omitempty"`
}

type ProgressMsg struct {
    For     string `json:"for"`
    At      int    `json:"at"`
    Failure int    `json:"fail"`
    Success int    `json:"succ"`
}

type CompleteMsg struct {
    Id      string `json:"id"`
    Success bool   `json:"succ"`
    Message string `json:"msg,omitempty"`
}

type RatingMsg struct {
    Text  string `json:"text"`
    Class string `json:"class"`
}

type NetworkMsg struct {
    Port      int    `json:"port"`
    IpVersion int    `json:"ipv"`
    Protocol  string `json:"proto"`
    DnsId     int    `json:"id"`
}

type RrMsg struct {
    Name   string `json:"n"`
    Rrtype string `json:"t"`
    Class  string `json:"c"`
    Ttl    int    `json:"l,omitempty"`
    Rdata  string `json:"rdata,omitempty"`
}

type DnsMsg struct {
    CheckId     string `json:"cid"`
    UnixNano    int64  `json:"unts"`
    Source      string `json:"src"`
    Destination string `json:"dst"`
    Port        int    `json:"port"`
    Protocol    string `json:"proto"`
    IpVersion   int    `json:"ipv"`

    DnsId  int    `json:"id"`
    Qr     bool   `json:"qr,omitempty"`
    Opcode string `json:"op,omitempty"`
    Aa     bool   `json:"aa,omitempty"`
    Tc     bool   `json:"tc,omitempty"`
    Rd     bool   `json:"rd,omitempty"`
    Ra     bool   `json:"ra,omitempty"`
    Z      bool   `json:"z,omitempty"`
    Ad     bool   `json:"ad,omitempty"`
    Cd     bool   `json:"cd,omitempty"`
    Do     bool   `json:"do,omitempty"`
    Rcode  string `json:"rc,omitempty"`

    Questions   []*RrMsg `json:"q,omitempty"`
    Answers     []*RrMsg `json:"ans,omitempty"`
    Authorities []*RrMsg `json:"ns,omitempty"`
    Additionals []*RrMsg `json:"add,omitempty"`
}

type WhoisMsg struct {
    Rir     string `json:"rir"`
    Netname string `json:"nn"`
    Ip      string `json:"ip"`
}

type LookupMsg struct {
    Id      string `json:"id"`
    Dn      string `json:"dn"`
    Success bool   `json:"ok"`
    Error   string `json:"err,omitempty"`
}

type UserAgentMsg struct {
    Text string `json:"text"`
}

type ClientMsg struct {
    Check     *CheckMsg     `json:"check,omitempty"`
    Prepare   *PrepareMsg   `json:"prepare,omitempty"`
    Progress  *ProgressMsg  `json:"progress,omitempty"`
    Complete  *CompleteMsg  `json:"complete,omitempty"`
    Rating    *RatingMsg    `json:"rating,omitempty"`
    Network   *NetworkMsg   `json:"network,omitempty"`
    Whois     *WhoisMsg     `json:"whois,omitempty"`
    Lookup    *LookupMsg    `json:"lookup,omitempty"`
    Dns       *DnsMsg       `json:"dns,omitempty"`
    UserAgent *UserAgentMsg `json:"ua,omitempty"`
}

var addr = flag.String("addr", "cmdns.dev.dns-oarc.net:443", "http service address")

func main() {
    flag.Parse()
    log.SetFlags(0)

    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt)

    u := url.URL{Scheme: "wss", Host: *addr, Path: "/ws/"}
    log.Printf("connecting to %s", u.String())

    c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        log.Fatal("dial:", err)
    }
    defer c.Close()

    done := make(chan struct{})

    prepareDone := 0
    prepareTotal := 0

    go func() {
        defer c.Close()
        defer close(done)
        for {
            _, message, err := c.ReadMessage()
            if err != nil {
                log.Println("read:", err)
                return
            }
            lines := strings.Split(string(message), "\n")
            for _, line := range lines {
                log.Printf("%s", line)

                var m ClientMsg
                if err = json.Unmarshal([]byte(line), &m); err != nil {
                    log.Println(err)
                } else {
                    if m.Prepare != nil {
                        if !m.Prepare.Done {
                            prepareTotal = m.Prepare.Total
                        } else {
                            prepareDone++
                        }
                        if prepareTotal > 0 && prepareTotal == prepareDone {
                            err = c.WriteMessage(websocket.TextMessage, []byte("{\"check\":{\"all\":true}}"))
                            if err != nil {
                                log.Println("write:", err)
                                return
                            }
                        }
                    }
                    if m.Lookup != nil {
                        _, err := http.Get("http://" + m.Lookup.Dn + "/dot.png")
                        if err != nil {
                            m.Lookup.Success = false
                            m.Lookup.Error = fmt.Sprintf("%v", err)
                        } else {
                            m.Lookup.Success = true
                        }
                        b, err := json.Marshal(m)
                        if err != nil {
                            log.Println("json.Marshal():", err)
                            return
                        }
                        err = c.WriteMessage(websocket.TextMessage, b)
                        if err != nil {
                            log.Println("write:", err)
                            return
                        }
                    }
                }
            }
        }
    }()

    err = c.WriteMessage(websocket.TextMessage, []byte("{\"prepare\":{}}"))
    if err != nil {
        log.Println("write:", err)
        return
    }

    for {
        select {
        case <-interrupt:
            log.Println("interrupt")
            err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
            if err != nil {
                log.Println("write close:", err)
                return
            }
            select {
            case <-done:
            case <-time.After(time.Second):
            }
            c.Close()
            return
        }
    }
}

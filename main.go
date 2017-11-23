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

var addr = flag.String("addr", "cmdns.dev.dns-oarc.net", "websocket address")
var exitWhenDone = flag.Bool("done", false, "Exit when done")

var c *websocket.Conn

func send(m *ClientMsg) error {
    b, err := json.Marshal(m)
    if err != nil {
        log.Println("send json.Marshal():", err)
        return err
    }
    err = c.WriteMessage(websocket.TextMessage, b)
    if err != nil {
        log.Println("send conn.WriteMessage():", err)
        return err
    }
    fmt.Println("{\"send\":" + string(b) + "}")
    return nil
}

func main() {
    flag.Parse()
    log.SetFlags(0)
    log.SetOutput(os.Stderr)

    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt)

    u := url.URL{Scheme: "wss", Host: *addr + ":443", Path: "/ws/"}
    log.Printf("connecting to %s", u.String())

    var err error
    c, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        log.Fatal("dial:", err)
    }
    defer c.Close()

    done := make(chan struct{})
    lookup := make(chan *ClientMsg)

    go func() {
        for {
            select {
            case <-done:
                return
            case m, ok := <-lookup:
                if !ok {
                    return
                }
                _, err := http.Get("http://" + m.Lookup.Dn + "/dot.png")
                if err != nil {
                    m.Lookup.Success = false
                    m.Lookup.Error = fmt.Sprintf("%v", err)
                } else {
                    m.Lookup.Success = true
                }
                if err = send(m); err != nil {
                    return
                }
            }
        }
    }()

    go func() {
        defer close(done)
        defer close(lookup)

        prepareDone := 0
        prepareTotal := 0

        for {
            _, message, err := c.ReadMessage()
            if err != nil {
                log.Println("read:", err)
                return
            }
            lines := strings.Split(string(message), "\n")
            for _, line := range lines {
                fmt.Println(line)

                var m ClientMsg
                if err = json.Unmarshal([]byte(line), &m); err != nil {
                    log.Println("read json.Unmarshal():", err)
                    return
                } else {
                    if m.Prepare != nil {
                        if !m.Prepare.Done {
                            prepareTotal = m.Prepare.Total
                        } else {
                            prepareDone++
                        }
                        if prepareTotal > 0 && prepareTotal == prepareDone {
                            if err = send(&ClientMsg{Check: &CheckMsg{All: true}}); err != nil {
                                return
                            }
                        }
                    }
                    if m.Lookup != nil {
                        func() {
                            defer func() {
                                recover()
                            }()
                            lookup <- &m
                        }()
                    }
                    if m.Rating != nil && *exitWhenDone == true {
                        return
                    }
                }
            }
        }
    }()

    if err = send(&ClientMsg{Prepare: &PrepareMsg{}}); err != nil {
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
        case <-done:
            c.Close()
            return
        }
    }
}

package main

import (
    "flag"
    "fmt"
    "github.com/miekg/dns"
    "io"
    "log"
    "net"
    "os"
    "os/signal"
    "strings"
    "syscall"
)

var zones map[string]net.IP

func dnsProxy(w dns.ResponseWriter, m *dns.Msg) *dns.Msg {
    if len(m.Question) == 0 {
        return nil
    }
    q := m.Question[0]

    ip, exists := zones[q.Name]
    if !exists {
        return nil
    }

    if q.Qtype != dns.TypeA {
        response := new(dns.Msg)
        response.SetReply(m)
        return response
    }

    log.Printf("Proxying request for %s IN A from %s", q.Name, w.RemoteAddr())
    response := new(dns.Msg)
    response.SetReply(m)

    rr := new(dns.A)
    rr.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA,
        Class: dns.ClassINET, Ttl: 0}
    rr.A = ip.To4()
    response.Answer = append(m.Answer, rr)

    return response
}

func dnsHandler(w dns.ResponseWriter, m *dns.Msg) {
    if msg := dnsProxy(w, m); msg != nil {
        w.WriteMsg(msg)
        return
    }

    c := new(dns.Client)
    c.Net = "udp"
    r, _, err := c.Exchange(m, "8.8.8.8:53")
    if err != nil {
        log.Print(err)
        return
    }
    w.WriteMsg(r)
}

func tcpProxy(local net.Conn, remoteAddr string) {
    remote, err := net.Dial("tcp", remoteAddr)
    if err != nil {
        log.Print("Failed to connect to %s: %s", remoteAddr, err)
        return
    }
    go io.Copy(local, remote)
    go io.Copy(remote, local)
}

func listenAndServe() {
    go func() {
        err := dns.ListenAndServe(":53", "udp", dns.HandlerFunc(dnsHandler))
        if err != nil {
            log.Fatal(err)
        }
    }()
    go func() {
        err := dns.ListenAndServe(":53", "tcp", dns.HandlerFunc(dnsHandler))
        if err != nil {
            log.Fatal(err)
        }
    }()
    go func() {
        listener, err := net.Listen("tcp", ":80")
        if err != nil {
            log.Fatal(err)
        }
        for {
            conn, err := listener.Accept()
            if err != nil {
                log.Print(err)
            }
            go tcpProxy(conn, "movies.netflix.com:80")
        }
    }()
    go func() {
        listener, err := net.Listen("tcp", ":443")
        if err != nil {
            log.Fatal(err)
        }
        for {
            conn, err := listener.Accept()
            if err != nil {
                log.Print(err)
            }
            go tcpProxy(conn, "cbp-us.nccp.netflix.com:443")
        }
    }()
}

func printfErr(format string, a ...interface{}) {
    fmt.Fprintf(os.Stderr, format+"\n", a...)
    os.Exit(2)
}

func main() {
    flag.Parse()
    if flag.NArg() == 0 {
        printfErr("usage: %s zone:ip [zone:ip ...]", os.Args[0])
    }

    zones = make(map[string]net.IP, flag.NArg())
    for _, arg := range flag.Args() {
        zoneAndIp := strings.SplitN(arg, ":", 2)
        if len(zoneAndIp) != 2 {
            printfErr("Invalid zone mapping: %s", arg)
        }
        zone := zoneAndIp[0]
        if !strings.HasSuffix(zone, ".") {
            zone += "."
        }
        ip := net.ParseIP(zoneAndIp[1])
        if ip == nil {
            printfErr("Invalid IP address: %s", zoneAndIp[1])
        }
        zones[zone] = ip
        log.Printf("Answering %s with %s", zone, ip)
    }

    listenAndServe()

    sig := make(chan os.Signal)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    for {
        select {
        case s := <-sig:
            log.Fatalf("Signal (%d) received, stopping\n", s)
        }
    }
}

package main

import (
	"bufio"
	"bytes"
	"code.google.com/p/portaudio-go/portaudio"
	"encoding/binary"
	// "encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var dbuff = make([]byte, 8)
var udp_init bool = false

type Command struct {
	Type string `json:"type"`
	Args string `json:"args"`
}

type RemoteData struct {
	Code    int    `json:"code"`
	Content string `json:"content"`
	Error   string `json:"err"`
	Type    string `json:"type"`
}

func GenDbuff(d int) string {
	if d > 99999999 {
		log.Fatal("well FuCK")
	}
	s := strings.Repeat("0", 8-len(strconv.Itoa(d)))
	return s + strconv.Itoa(d)
}

func ReadDbuff(data []byte) int {
	b, err := strconv.Atoi(string(data))
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func stdinHandler(c_stdin, c_conn, c_udp chan string) {
	for {
		msg := <-c_stdin
		if string(msg[0]) == "/" {
			// Send command
			c_conn <- msg[1:]
		} else {
			msg = strings.Replace(msg, "\n", "", -1)
			c_udp <- msg
		}
	}
}

type mic struct {
	*portaudio.Stream
	buffer []float32
	conn   *net.UDPConn
	paddr  *net.UDPAddr
	i      int
}

func newMic(conn *net.UDPConn, paddr *net.UDPAddr) *mic {
	h, err := portaudio.DefaultHostApi()
	if err != nil {
		panic(err)
	}
	p := portaudio.LowLatencyParameters(h.DefaultInputDevice, h.DefaultOutputDevice)
	p.Input.Channels = 1
	p.Output.Channels = 1
	fmt.Println(p.SampleRate)
	fmt.Println(p.FramesPerBuffer)
	p.SampleRate = 48000
	fmt.Println(p.SampleRate)
	// p.SampleRate = 16000
	// p.FramesPerBuffer = 512
	m := &mic{
		buffer: make([]float32, 16),
		conn:   conn,
		paddr:  paddr,
	}
	m.Stream, err = portaudio.OpenStream(p, m.processAudio)
	if err != nil {
		panic(err)
	}
	return m
}

func (m *mic) processAudio(in, out []float32) {
	// fmt.Println("audio")
	copy(out, m.buffer)
	// copy(out, m.buffer)
	// var buf bytes.Buffer
	buf := new(bytes.Buffer)
	for _, v := range in {
		err := binary.Write(buf, binary.LittleEndian, v)
		if err != nil {
			panic(err)
		}
	}
	// enc := gob.NewEncoder(&buf)
	// enc.Encode(in)
	m.conn.WriteToUDP(buf.Bytes(), m.paddr)
	// fmt.Println("Send data ", m.paddr.IP)
}

func udpHandler(c_udp chan string) {
	init_msg := <-c_udp
	// gob.Register(make([]float32, 1))

	// {{{ UDP Connection stuff
	// Init udp for bruno
	saddr, err := net.ResolveUDPAddr("udp", "localhost:31500")
	if err != nil {
		log.Fatal(err)
	}
	laddr, err := net.ResolveUDPAddr("udp", ":0")
	// laddr, err := net.ResolveUDPAddr("udp", <-c_udp)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	conn.WriteToUDP([]byte(init_msg), saddr)
	paddr, err := net.ResolveUDPAddr("udp", <-c_udp)
	if err != nil {
		panic(err)
	}

	if err := portaudio.Initialize(); err != nil {
		log.Fatal(err)
	}
	/// }}}

	if err := portaudio.Initialize(); err != nil {
		panic(err)
	}
	defer portaudio.Terminate()
	m := newMic(conn, paddr)
	defer m.Close()

	go func() {
		for {
			buf := make([]byte, 8192)
			_, err := conn.Read(buf)
			// fmt.Println("GOT DAATA")
			if err != nil {
				panic(err)
			}
			var recv []float32 = make([]float32, 1024)
			b := bytes.NewReader(buf)
			err = binary.Read(b, binary.LittleEndian, &recv)
			if err != nil {
				// panic(err)
				fmt.Println(err)
			}
			m.buffer = recv
		}
	}()

	// Loop for recording audio
	m.Start()
	for {
		time.Sleep(time.Second * 1)
	}
}

func connHandler(c_conn chan string, c_udp chan string, conn net.Conn) {
	var c_input chan []byte = make(chan []byte)

	// Send data
	go func() {
		for {
			data := <-c_conn
			// fmt.Println(data)
			c := Command{Type: "cmd", Args: data}
			b, err := json.Marshal(c)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(conn, GenDbuff(len(b)))
			fmt.Fprintf(conn, string(b[:]))
		}
	}()

	// Handle remote data
	go func() {
		for {
			data := <-c_input

			v := RemoteData{}
			err := json.Unmarshal(data, &v)
			if err != nil {
				log.Fatal(err)
			}

			if v.Type == "event" {
				if v.Code == 100 {
					// Incoming call
					fmt.Println("INCOMMING!")
					fmt.Println(v.Content)
					data := strings.Split(v.Content, " ")
					c_udp <- data[0]
					c_udp <- data[1]
				}
				if v.Code == 101 {
					c_udp <- v.Content
				}
			}
		}
	}()

	// Receive data
	in := bufio.NewReader(conn)
	for {
		var dbuff = make([]byte, 8)
		in.Read(dbuff)
		data := make([]byte, ReadDbuff(dbuff))
		in.Read(data)
		// fmt.Println(string(data))
		c_input <- data
	}
}

func main() {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	var c_stdin chan string = make(chan string)
	var c_conn chan string = make(chan string)
	var c_udp chan string = make(chan string)
	go stdinHandler(c_stdin, c_conn, c_udp)
	go connHandler(c_conn, c_udp, conn)
	go udpHandler(c_udp)

	// stdin
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">>> ")
		input, _ := reader.ReadString('\n')
		c_stdin <- input
	}
}

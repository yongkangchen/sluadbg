package main

import (
"os"
"bufio"
"net"
"fmt"
"reflect"
"strings"
"encoding/binary"
"bytes"
"os/signal"
"flag"
)


var prompt string = ""

type Client struct {
	input chan string
	netin chan string
	console *bufio.Reader
	reader *bufio.Reader
	writer *bufio.Writer
}

func (c *Client) ReadConsole() {
	for {
		l,_:=c.console.ReadString('\n')
		c.input<-l
	}
}

func (c *Client) ReadNet() {
	for {
		data:=make([]byte,4)
		l,_:=c.reader.Read(data)
		if l==0 {
			c.netin<-"quit"
			return
		}
		var len int32
		buf:=bytes.NewBuffer(data)
		binary.Read(buf,binary.LittleEndian,&len);

		data=make([]byte,len)
		l,_=c.reader.Read(data)
		
		if int32(l)!=len {
			c.netin<-"quit"
		} else {
			c.netin<-string(data)
		}
	}
}


func (c *Client) send(data string) {
	buf:=new(bytes.Buffer)
	binary.Write(buf,binary.LittleEndian,int32(len(data)))
	binary.Write(buf,binary.LittleEndian,[]byte(data))
	c.writer.Write(buf.Bytes())
	c.writer.Flush()
}

func (c *Client) onInput(data string) {
	c.send(data)
}

func invoke(c *Client, name string, args... interface{}) {
	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	v:=reflect.ValueOf(c)
	m:=v.MethodByName(name)
	if m.IsValid() {
		m.Call(inputs)
	} else {
		fmt.Printf("Invalid cmd %s\n",name)
	}
}

func (c *Client) Cmd_print(tail string) {
	fmt.Println(tail)
}

func (c *Client) Cmd_ret(tail string) {
	fmt.Print(prompt)
}

func (c *Client) Cmd_quit(tail string) {
	os.Exit(0)
}

func (c *Client) Cmd_break(tail string) {
	prompt="ldb> "
	args:=strings.Split(tail,",")
	fmt.Printf("Break at %s:%s\n",args[0],args[1])
	fmt.Print(prompt)
}

func (c *Client) Cmd_resume(tail string) {
	fmt.Println("")
	prompt="slua> "
	fmt.Print(prompt)
}

func (c *Client) onCommand(data string) {
	cmd:=data
	tail:=""
	n:=strings.Index(data," ")
	if n>0 {
		cmd=data[0:n]
		tail=data[n+1:]
	}

	invoke(c,"Cmd_"+cmd,tail)
}

func (c *Client) close() {
	c.send("quit")
	os.Exit(0)
}


func main() {

	hostptr:=flag.String("host","localhost","remote host ip to connect")
	portptr:=flag.Int("port",10240,"remote host port to connect")
	flag.Parse()

	fmt.Printf("Connect to %s:%d\n",*hostptr,*portptr)

	prompt="slua> "

	conn,err:=net.Dial("tcp","localhost:10240")
	if err!=nil {
		fmt.Println(err)
		return
	}

	defer conn.Close()
	

	fmt.Println("Host connected")
	fmt.Println("Type 'start' to continue game")

	console:=bufio.NewReader(os.Stdin)
	nw:=bufio.NewWriter(conn)
	nr:=bufio.NewReader(conn)

	client:=&Client{
		input:make(chan string),
		netin:make(chan string),
		console:console,
		reader:nr,
		writer:nw,
	}

	
	fmt.Print(prompt)

	go client.ReadConsole()
	go client.ReadNet()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,os.Interrupt,os.Kill)


	// main loop
	for {

		select {

			case data:=<-client.input:
				client.onInput(data)
			case data:=<-client.netin:
				client.onCommand(data)
			case _=<-sigc:
				client.close()
		}

	}
}
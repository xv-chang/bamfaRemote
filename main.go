package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Bamfa struct {
		Uid   string
		Topic string
	}
	Wol struct {
		Ip  string
		Mac string
	}
}

var config Config
var conn net.Conn

//关机数据包 头部信息
var offHeader = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

//开机数据包 头部信息
var onHeader = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	configPath := "config.yaml"
	if !FileExist(configPath) {
		log.Fatalf("file config.yaml not exist")
	}
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("read config.yaml err: %v", err.Error())
	}
	config = Config{}
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		log.Fatalf("unmarshal config.yaml err: %v", err.Error())
	}
	go linkBamfaTCPServer()
	go sendBamfaHeartBeat()
	s := <-c
	fmt.Println("stoped bamfa remote server ,", s)
}
func FileExist(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}

func linkBamfaTCPServer() {
	for {
		conn, err := net.Dial("tcp", "bemfa.com:8344")
		if err != nil {
			fmt.Printf("connect failed, err : %v\n", err.Error())
		} else {
			sendAuthData(conn)
			bamfaRecv(conn)
			conn.Close()
			conn = nil
		}
		//10秒后重连
		time.Sleep(10 * time.Second)
	}
}

func bamfaRecv(conn net.Conn) {
	var buf = make([]byte, 64)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Read failed , err : %v\n", err)
			break
		}
		text := string(buf[:n-2])
		go processRecv(text)
	}
}

func sendAuthData(conn net.Conn) {
	authData := fmt.Sprintf("cmd=1&uid=%s&topic=%s\r\n", config.Bamfa.Uid, config.Bamfa.Topic)
	conn.Write([]byte(authData))
}

func sendBamfaHeartBeat() {
	for {
		if conn != nil {
			conn.Write([]byte("ping\r\n"))
		}
		time.Sleep(5 * time.Second)
	}
}

func parseRecv(recvData string) map[string]string {
	result := map[string]string{}
	props := strings.Split(recvData, "&")
	for _, prop := range props {
		keyValue := strings.Split(prop, "=")
		result[keyValue[0]] = keyValue[1]
	}
	return result
}
func processRecv(recvData string) {
	props := parseRecv(recvData)
	if props["cmd"] == "2" {
		// 暂时用不到topic ,后续可能 会用来区分设备
		// topic := props["topic"]
		msg := props["msg"]
		wol(msg)
	}
}

/*幻数据包最简单的构成是6字节的255（FF FF FF FF FF FF FF），紧接着为目标计算机的48位MAC地址，重复16次，数据包共计102字节。
有时数据包内还会紧接着4-6字节的密码信息。这个帧片段可以包含在任何协议中，最常见的是包含在 UDP 中。
*/
func wol(msg string) {
	conn, err := net.Dial("udp", fmt.Sprint(config.Wol.Ip, ":", 9))
	if err != nil {
		log.Printf("connect failed, err : %v\n", err.Error())
		return
	}
	defer conn.Close()
	hwAddr, err := net.ParseMAC(config.Wol.Mac)
	if err != nil {
		log.Printf("ParseMAC, err : %v\n", err.Error())
		return
	}
	var buf bytes.Buffer
	if msg == "on" {
		buf.Write(onHeader)
	} else {
		buf.Write(offHeader)
	}
	for i := 0; i < 16; i++ {
		buf.Write(hwAddr)
	}
	n, err := conn.Write(buf.Bytes())
	if err == nil && n != 102 {
		err = fmt.Errorf("magic packet sent was %d bytes (expected 102 bytes sent)", n)
	}
	if err != nil {
		log.Printf("send magic packet err : %v\n", err.Error())
		return
	}
	fmt.Printf("Magic packet sent successfully to %s\n", config.Wol.Mac)
}

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
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
		Delay       int
		Ip          string
		Mac         string
		IsEtherwake bool `yaml:"isEtherwake"`
		Ifname      string
		P           string
	}
}

var config Config
var heartTicker *time.Ticker

// 关机数据包 头部信息
var offHeader = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

// 开机数据包 头部信息
var onHeader = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

// 配置文件路径
var confPath = flag.String("c", "config.yaml", "config.yaml file PATH")

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	flag.Parse()
	config = ReadConfig(*confPath)
	go linkBamfaTCPServer()
	s := <-c
	fmt.Println("stoped bamfa remote server ,", s)
}

func linkBamfaTCPServer() {
	for {
		log.Printf("start connect")
		conn, err := net.Dial("tcp", "bemfa.com:8344")
		go sendBamfaHeartBeat(conn)
		if err != nil {
			fmt.Printf("connect failed, err : %v\n", err.Error())
		} else {
			sendAuthData(conn)
			bamfaRecv(conn)
			heartTicker.Stop()
			conn.Close()
			conn = nil
			log.Printf("closed connection")
		}
		//10秒后重连
		time.Sleep(10 * time.Second)
	}
}

func bamfaRecv(conn net.Conn) {
	var buf = make([]byte, 1024)
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

func sendBamfaHeartBeat(conn net.Conn) {
	heartTicker = time.NewTicker(time.Second * 60)
	for range heartTicker.C {
		conn.Write([]byte("ping\r\n"))
	}
}

func parseRecv(recvData string) (map[string]string, error) {
	result := map[string]string{}
	if !strings.Contains(recvData, "&") {
		return nil, errors.New("msg err")
	}
	props := strings.Split(recvData, "&")
	for _, prop := range props {
		keyValue := strings.Split(prop, "=")
		result[keyValue[0]] = keyValue[1]
	}
	return result, nil
}
func processRecv(recvData string) {
	//log.Printf("recvData:%v", recvData)
	props, err := parseRecv(recvData)
	if err != nil {
		log.Printf("parseRecv err: %v,recvData: %v ", err.Error(), recvData)
		return
	}
	if props["cmd"] == "2" {
		// 暂时用不到topic ,后续可能 会用来区分设备
		// topic := props["topic"]
		config = ReadConfig(*confPath)
		log.Printf("delay %d seconds", config.Wol.Delay)
		time.Sleep(time.Duration(config.Wol.Delay) * time.Second)
		if config.Wol.IsEtherwake {
			wolByEtherwake(config.Wol.Ifname, config.Wol.P)
		} else {
			msg := props["msg"]
			wol(msg)
		}
	}
}

/*
幻数据包最简单的构成是6字节的255（FF FF FF FF FF FF FF），紧接着为目标计算机的48位MAC地址，重复16次，数据包共计102字节。
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

// 使用Etherwake唤醒，需要提前安装Etherwake
func wolByEtherwake(ifname string, p string) {
	cmd := exec.Command("etherwake", "-D", "-i", ifname, p)
	err := cmd.Run()
	log.Println(cmd)
	if err != nil {
		log.Println("Execute Command failed:" + err.Error())
		return
	}
	log.Println("Execute Command finished.")
}
func ReadConfig(confPath string) Config {
	var config Config

	// Open YAML file
	file, err := os.Open(confPath)
	if err != nil {
		log.Println(err.Error())
	}
	defer file.Close()

	// Decode YAML file to struct
	if file != nil {
		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(&config); err != nil {
			log.Println(err.Error())
		}
	}

	return config
}

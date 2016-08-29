// rtmp.go
package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
)

const RTMP_SIG_SIZE = 1536

func HandShake(conn *net.TCPConn) bool {
	clientBuf := make([]byte, RTMP_SIG_SIZE+1)
	io.ReadFull(conn, clientBuf)

	if clientBuf[0] != 0x03 {
		fmt.Println("client version error ", clientBuf[0])
		conn.Close()
		return false
	}

	serverBuf := make([]byte, RTMP_SIG_SIZE+1)
	serverBuf[0] = 0x03
	serverBuf[1] = 0
	serverBuf[2] = 0
	serverBuf[3] = 0
	serverBuf[4] = 0

	for i := 5; i < RTMP_SIG_SIZE+1-5; i++ {
		serverBuf[i] = 0xFF
	}

	conn.Write(serverBuf)
	conn.Write(clientBuf[1:])
	c2 := make([]byte, RTMP_SIG_SIZE)
	io.ReadFull(conn, c2)

	if bytes.Compare(c2, serverBuf[1:]) != 0 {
		fmt.Println("client handshake c2 erro")
		return false
	}

	return true
}

func parseConnectObject(msgObject []byte) int {
	fmt.Println("msgobjcet len ", len(msgObject))
	fmt.Println(msgObject)
	for i := 1; i < len(msgObject); {
		var keyLen int
		keyLen = int(msgObject[i+1])
		keyLen |= int(msgObject[i] << 8)
		keyName := string(msgObject[i+2 : i+2+keyLen])
		fmt.Println("i ", i, msgObject[i+2:i+2+keyLen])
		i += 2 + keyLen
		amfType := msgObject[i]

		fmt.Println(keyName, keyLen)

		switch amfType {
		case 0x02:
			valueLen := int(msgObject[i+2])
			valueLen |= int(msgObject[i+1])
			value := string(msgObject[i+3 : i+3+valueLen])
			fmt.Println(keyName, " : ", value)
			i += 1 + 2 + valueLen
		case 0x00:
			if msgObject[i+1] == 0x00 && msgObject[i+2] == 0x09 {
				fmt.Println("parseConnectObject end")
				i += 3
				return i
			}
			fmt.Println(keyName + " : 0")
			i += 1 + 8

		case 0x01:
			fmt.Println(keyName+" : ", msgObject[i+1])
			i += 1 + 1

		default:
			fmt.Println("parseConnectObject unknown afmtype ", amfType)
			return i
		}
	}

	return 1
}

func parseConnectBody(msgBody []byte) {
	fmt.Println(msgBody)
	for i := 0; i < len(msgBody); {
		switch msgBody[i] {
		case 0x02:
			strLen := int(msgBody[i+2])
			strLen |= int(msgBody[i+1] << 8)

			cmdName := string(msgBody[3 : strLen+3])
			if cmdName != "connect" {
				fmt.Println("parseConnectBody cmd name err ", cmdName, strLen, msgBody[0:10])
				return
			}

			fmt.Println("parseConnectBody cmd name ", cmdName)
			i += 3 + strLen

		case 0x00:
			i += 1 + 8

		case 0x03:
			n := parseConnectObject(msgBody[i:])
			i += n

		default:
			fmt.Println("parseConnectBody unknown amf header ", msgBody[i])
			i = len(msgBody)
		}
	}
}

func HandleConnectMessage(conn *net.TCPConn) {
	var b [1]byte
	n, err := conn.Read(b[:])
	if err != nil || n != 1 {
		fmt.Println("HandleConnectMessage read first byte err ", err)
		return
	}

	// 暂时只处理 chunk basic header 占一个字节的情况
	csid := b[0] & 0x3F
	chunkFmt := (b[0] & 0xC0) >> 6
	headerLen := (3-chunkFmt)*4 - 1

	header := make([]byte, headerLen)
	n, err = conn.Read(header)
	if err != nil {
		fmt.Println("HandleConnectMessage read msg header err ", err)
		return
	}

	mtid := header[6]
	msgBodyLen := header[5]
	msgBodyLen += header[6] << 8
	msgBodyLen += header[7] << 16

	fmt.Println("csdi : ", csid, " mtid : ", mtid)

	// rtmp 会将包分为128个字节大小，这里先固定写死
	msgBody1 := make([]byte, 128)
	n, err = io.ReadFull(conn, msgBody1)
	msgBodyHeader := make([]byte, 1)
	n, err = io.ReadFull(conn, msgBodyHeader)
	msgBody2 := make([]byte, msgBodyLen-128)
	n, err = io.ReadFull(conn, msgBody2)

	msgBody := make([]byte, msgBodyLen)
	copy(msgBody, msgBody1)
	copy(msgBody[128:], msgBody2)
	parseConnectBody(msgBody)
}

func handleRtmpConn(conn *net.TCPConn) {
	if !HandShake(conn) {
		return
	}

	fmt.Println("handshake ok")
	HandleConnectMessage(conn)
}

func RtmpInit() {
	port := "1935"

	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":"+port)
	if err != nil {
		fmt.Printf("tcp addr error %s\n", err.Error())
		os.Exit(1)
	}

	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Printf("Could not listen on port %d. %s\n", port, err.Error())
		os.Exit(1)
	}

	fmt.Printf("Listening on %s\n", ln.Addr().String())

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			fmt.Printf("Error accepting connection %s\n", err.Error())
			continue
		}

		fmt.Printf("New connection from %s\n", conn.RemoteAddr().String())
		go handleRtmpConn(conn)
	}
}

// noob project main.go
package main

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	//"bytes"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"runtime"
	"time"
)

type FrameInfo struct {
	buf   []byte
	delta int
}

func main() {
	port := "554"
	fmt.Printf("cpu %d\n", runtime.NumCPU())

	argLen := len(os.Args)
	if argLen != 2 {
		fmt.Println("noob mp4file")
		return
	}

	InitDecoders()

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
		go handleConn(conn)
	}
}

func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return
	}

	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], true
}

const (
	OPTIONS  = "OPTIONS"
	DESCRIBE = "DESCRIBE"
	SETUP    = "SETUP"
	PLAY     = "PLAY"
	TEARDOWN = "TEARDOWN"
)

type rtspRequest struct {
	url        string
	seq        int
	session    string
	transport  string
	mimeHeader textproto.MIMEHeader
	c          *net.TCPConn
}

const (
	SERVERNAME  = "noob Server(1.0)"
	RESPOPTIONS = "Public: OPTIONS, DESCRIBE, SETUP, PLAY, PAUSE, TEARDOWN, GET_PARAMETER"
)

func (rr *rtspRequest) handleOptions() {
	fmt.Fprintf(rr.c, "RTSP/1.0 200 OK\r\nServer: %s\r\nCSeq: %d\r\n%s\r\n\r\n",
		SERVERNAME, rr.seq, RESPOPTIONS)
}

/*
 949.mp4
a=range:npt=0-2655.94333

a=fmtp:96 profile-level-id=6742c0; sprop-parameter-sets=Z0LAFtsCgL6agwCDIAAAAwAgAAADA9Hixdw=,aMqPIA==; packetization-mode=1
a=rtpmap:97 mpeg4-generic/44100/2
a=fmtp:97 streamtype=5; profile-level-id=15; mode=AAC-hbr; config=121056e500; SizeLength=13; IndexLength=3; IndexDeltaLength=3;
*/

func (rr *rtspRequest) handleDescribe() {
	desc := "RTSP/1.0 200 OK\r\nServer: noob Server(1.0)\r\nCSeq: "
	desc += strconv.Itoa(rr.seq)
	desc += "\r\nContent-Length: "

	content := "v=0\r\no= - 0 43163 IN IP4 154.0.2.80\r\n"
	content += "c=IN IP4 0.0.0.0\r\n"
	content += "a=tool: ZMediaServer\r\n"
	content += "a=range:npt=0-\r\n"
	content += "m=video 0 RTP/AVP 96\r\n"
	content += "a=rtpmap:96 H264/90000\r\n"
	content += "a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z2QAKq2EAQwgCGEAQwgCGEAQwgCEK1A8ARPywgAAAwACAAADAHkI,aO48sA==;\r\n"
	//content += "a=fmtp:96 profile-level-id=6742c0; sprop-parameter-sets=Z0LAFtsCgL6agwCDIAAAAwAgAAADA9Hixdw=,aMqPIA==; packetization-mode=1\r\n"
	content += "a=control:trackID=0\r\n"
	content += "m=audio 0 RTP/AVP 98\r\n"
	content += "a=rtpmap:98 mpeg4-generic/48000/2\r\n"
	//content += "a=rtpmap:98 mpeg4-generic/44100/2"
	content += "a=fmtp:98 streamtype=5; profile-level-id=15; mode=AAC-hbr; config=1190; SizeLength=13;IndexLength=3; IndexDeltaLength=3; Profile=1;\r\n"
	//content += "a=fmtp:97 streamtype=5; profile-level-id=15; mode=AAC-hbr; config=121056e500; SizeLength=13; IndexLength=3; IndexDeltaLength=3;\r\n"
	content += "a=control:trackID=1\r\n"

	desc += strconv.Itoa(len(content))
	desc += "\r\n\r\n"
	desc += content
	fmt.Fprintf(rr.c, "%s", desc)
}

func (rr *rtspRequest) handleSetup() {
	transport := rr.mimeHeader.Get("Transport")

	setupResponse := "RTSP/1.0 200 OK\r\n"
	setupResponse += "Server: noob Server(1.0)\r\n"
	setupResponse += "CSeq: "
	setupResponse += strconv.Itoa(rr.seq)
	setupResponse += "\r\ntimestamp: 0.000\r\n"
	setupResponse += "Session: 79326363544508238\r\n"
	setupResponse += "Transport: "
	setupResponse += transport
	setupResponse += "\r\n\r\n"
	fmt.Fprintf(rr.c, "%s", setupResponse)
}

func readLoop(ac chan *FrameInfo, vc chan *FrameInfo) {
	mp4 := ParseMp4(os.Args[1])

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("open file err ", err, f)
		return
	}

	si := mp4.SampleHeader.Next
	f.Seek(int64(si.Offset), os.SEEK_SET)

	for ; si != nil; si = si.Next {
		fi := new(FrameInfo)
		fi.buf = make([]byte, si.FrameLen)
		fi.delta = int(si.Delta)
		f.Read(fi.buf)

		//fmt.Println("frame type ", si.FrameType, " len ", si.FrameLen)
		if si.FrameType == FRAME_AUDIO {
			ac <- fi
		} else {
			vc <- fi
		}
	}

	f.Close()
}

func (rr *rtspRequest) handlePlay() {
	resp := "RTSP/1.0 200 OK\r\n"
	resp += "Server: noob Server(2.0)\r\n"
	resp += "CSeq: "
	resp += strconv.Itoa(rr.seq)
	resp += "\r\nSession: 79326363544508238\r\n"
	resp += "Range: npt=0.000-\r\n\r\n"
	fmt.Fprintf(rr.c, "%s", resp)

	ac := make(chan *FrameInfo, 50)
	vc := make(chan *FrameInfo, 50)

	go readLoop(ac, vc)
	go Mp4VideoRtpLoop(rr.c, vc)
	go Mp4AudioRtpLoop(rr.c, ac)
}

const RTP_MAX_LEN = 1440

func buildAudioRtp(buffer []byte, seq uint16, ts uint32, alen int) []byte {
	rtp := make([]byte, RTP_MAX_LEN)

	copy(rtp[16:], buffer)
	rtp[0] = 0x80
	rtp[1] = 0x62
	rtp[1] |= 0x80
	rtp[2] = byte(seq >> 8)
	rtp[3] = byte(seq & 0xFF)

	rtp[4] = byte((ts >> 24) & 0xFF)
	rtp[5] = byte((ts >> 16) & 0xFF)
	rtp[6] = byte((ts >> 8) & 0xFF)
	rtp[7] = byte(ts & 0xFF)

	rtp[8] = 0x33
	rtp[9] = 0x33
	rtp[10] = 0x33
	rtp[11] = 0x33

	rtp[12] = 0
	rtp[13] = 0x10
	rtp[14] = byte(alen >> 5)
	rtp[15] = byte((alen & 0x1F) << 3)

	return rtp
}

func Mp4AudioRtpLoop(c *net.TCPConn, ac chan *FrameInfo) {
	var ts uint32 = 10000
	var seq uint16 = 1

	for fi := range ac {
		seq++
		ts += uint32(fi.delta)
		alen := len(fi.buf)

		rtp := buildAudioRtp(fi.buf, seq, ts, alen)
		sendAudioRtp(c, rtp[:alen+16])

		time.Sleep(time.Millisecond * time.Duration(fi.delta*1000/48000))
	}
}

func audioRtpLoop(c *net.TCPConn) {
	f, err := os.Open("d:\\mygo\\src\\noob\\audio")
	if err != nil {
		fmt.Println("read audio err ", err)
		return
	}

	var ts uint32 = 10000
	var seq uint16 = 1

	for {
		header := make([]byte, 4)
		blen, err := f.Read(header)
		if err != nil || blen != 4 {
			fmt.Println("read audio file header len ", blen, err)
			break
		}

		alen := int(header[0]) << 24
		alen |= int(header[1]) << 16
		alen |= int(header[2]) << 8
		alen |= int(header[3])

		buffer := make([]byte, alen)
		blen, err = f.Read(buffer)
		if err != nil || blen != alen {
			fmt.Println("read audio file content len ", blen, err)
			break
		}

		seq++
		ts += 1024

		rtp := buildAudioRtp(buffer, seq, ts, alen)
		sendAudioRtp(c, rtp[:alen+16])

		time.Sleep(time.Millisecond * 21)
	}

	f.Close()
}

func sendAudioRtp(c *net.TCPConn, rtp []byte) {
	rlen := len(rtp)
	buf := make([]byte, rlen+4)
	buf[0] = '$'
	buf[1] = 0x02
	buf[2] = byte(rlen >> 8)
	buf[3] = byte(rlen & 0xFF)

	copy(buf[4:], rtp)
	c.Write(buf)
}

func buildRtp(buffer []byte, seq uint16, ts uint32, marker byte) []byte {
	rtp := make([]byte, RTP_MAX_LEN)

	copy(rtp[12:], buffer)
	rtp[0] = 0x80
	rtp[1] = 0x60
	rtp[1] |= marker //0x80
	rtp[2] = byte(seq >> 8)
	rtp[3] = byte(seq & 0xFF)

	//ts += 3600
	rtp[4] = byte((ts >> 24) & 0xFF)
	rtp[5] = byte((ts >> 16) & 0xFF)
	rtp[6] = byte((ts >> 8) & 0xFF)
	rtp[7] = byte(ts & 0xFF)

	rtp[8] = 0x22
	rtp[9] = 0x22
	rtp[10] = 0x22
	rtp[11] = 0x22

	return rtp
}

func buildFuaRtp(buffer []byte, seq uint16, ts uint32, marker byte, flag byte, olen int) []byte {
	rtp := make([]byte, RTP_MAX_LEN)
	copy(rtp[olen:], buffer)

	rtp[0] = 0x80
	rtp[1] = 0x60
	rtp[1] |= marker //0x80
	rtp[2] = byte(seq >> 8)
	rtp[3] = byte(seq & 0xFF)

	//ts += 3600
	rtp[4] = byte((ts >> 24) & 0xFF)
	rtp[5] = byte((ts >> 16) & 0xFF)
	rtp[6] = byte((ts >> 8) & 0xFF)
	rtp[7] = byte(ts & 0xFF)

	rtp[8] = 0x22
	rtp[9] = 0x22
	rtp[10] = 0x22
	rtp[11] = 0x22

	rtp[12] = 0x7C
	rtp[13] = flag

	return rtp
}

func sendRtp(c *net.TCPConn, rtp []byte) {
	rlen := len(rtp)
	buf := make([]byte, rlen+4)
	buf[0] = '$'
	buf[1] = 0
	buf[2] = byte(rlen >> 8)
	buf[3] = byte(rlen & 0xFF)

	copy(buf[4:], rtp)
	c.Write(buf)
}

func Mp4VideoRtpLoop(c *net.TCPConn, vc chan *FrameInfo) {
	var ts uint32 = 10000
	var seq uint16 = 1

	for fi := range vc {
		totalLen := 0
		buf := fi.buf
		for {
			vlen := int(buf[totalLen]) << 24
			vlen += int(buf[totalLen+1]) << 16
			vlen += int(buf[totalLen+2]) << 8
			vlen += int(buf[totalLen+3])

			buffer := buf[totalLen+4:]
			totalLen += vlen
			totalLen += 4
			if buffer[0] == 0x67 || buffer[0] == 0x61 {
				ts += uint32(fi.delta)
			}

			var marker byte = 0
			if vlen < RTP_MAX_LEN-13 {
				if buffer[0] == 0x61 {
					marker = 0x80
				}
				seq++
				rtp := buildRtp(buffer, seq, ts, marker)
				sendRtp(c, rtp[:vlen+12])
			} else {
				// first rtp packet
				seq++
				slen := RTP_MAX_LEN - 13
				buf := buffer[:slen]
				vlen -= slen
				count := slen
				flag := (buffer[0] & 0x0F) | 0x80
				rtp := buildFuaRtp(buf, seq, ts, marker, flag, 13)
				sendRtp(c, rtp[:RTP_MAX_LEN])

				// other rtp packet

				for {
					if vlen < slen {
						break
					}

					slen = RTP_MAX_LEN - 14
					seq++
					buf = buffer[count : count+slen]

					vlen -= slen
					count += slen
					flag = buffer[0] & 0x0F
					rtp = buildFuaRtp(buf, seq, ts, marker, flag, 14)
					sendRtp(c, rtp[:RTP_MAX_LEN])
				}

				// last rtp packet
				seq++
				rbuf := buffer[count:]
				flag = (buffer[0] & 0x0F) | 0x40
				marker = 0x80
				rtp = buildFuaRtp(rbuf, seq, ts, marker, flag, 14)
				if vlen != len(rbuf) {
					fmt.Println("vlen != rlen ", vlen, len(rbuf))
				}
				rlen := len(rbuf) + 13
				sendRtp(c, rtp[:rlen])
			}

			if buffer[0] != 0x68 && buffer[0] != 0x06 && buffer[0] != 0x67 {
				time.Sleep(time.Millisecond * time.Duration(fi.delta*1000/90000))
			}

			if totalLen == len(buf) {
				break
			}
		}
	}
}

func videoRtpLoop(c *net.TCPConn) {
	f, err := os.Open("d:\\mygo\\src\\noob\\video")
	if err != nil {
		fmt.Println("read video err ", err)
		return
	}

	var ts uint32 = 10000
	var seq uint16 = 1

	for {
		header := make([]byte, 4)
		blen, err := f.Read(header)
		if err != nil || blen != 4 {
			fmt.Println("read video file header len ", blen, err)
			break
		}

		vlen := int(header[0]) << 24
		vlen += int(header[1]) << 16
		vlen += int(header[2]) << 8
		vlen += int(header[3])

		buffer := make([]byte, vlen)
		blen, err = f.Read(buffer)
		if err != nil || blen != vlen {
			fmt.Println("read video file content len ", blen, err)
			break
		}

		if buffer[0] == 0x67 || buffer[0] == 0x61 {
			ts += 3600
		}

		var marker byte = 0

		if vlen < RTP_MAX_LEN-13 {
			if buffer[0] == 0x61 {
				marker = 0x80
			}
			seq++
			rtp := buildRtp(buffer, seq, ts, marker)
			sendRtp(c, rtp[:vlen+12])
		} else {
			// first rtp packet
			seq++
			slen := RTP_MAX_LEN - 13
			buf := buffer[:slen]
			vlen -= slen
			count := slen
			flag := (buffer[0] & 0x0F) | 0x80
			rtp := buildFuaRtp(buf, seq, ts, marker, flag, 13)
			sendRtp(c, rtp[:RTP_MAX_LEN])

			// other rtp packet

			for {
				if vlen < slen {
					break
				}

				slen = RTP_MAX_LEN - 14
				seq++
				buf = buffer[count : count+slen]

				vlen -= slen
				count += slen
				flag = buffer[0] & 0x0F
				rtp = buildFuaRtp(buf, seq, ts, marker, flag, 14)
				sendRtp(c, rtp[:RTP_MAX_LEN])
			}

			// last rtp packet
			seq++
			rbuf := buffer[count:]
			flag = (buffer[0] & 0x0F) | 0x40
			marker = 0x80
			rtp = buildFuaRtp(rbuf, seq, ts, marker, flag, 14)
			if vlen != len(rbuf) {
				fmt.Println("vlen != rlen ", vlen, len(rbuf))
			}
			rlen := len(rbuf) + 13
			sendRtp(c, rtp[:rlen])
		}

		if buffer[0] != 0x68 && buffer[0] != 0x06 && buffer[0] != 0x65 {
			time.Sleep(time.Millisecond * 40)
		}
	}

	f.Close()
}

func (rr *rtspRequest) handleTeardown() {

}

func newRtspRequest() *rtspRequest {
	rr := &rtspRequest{}
	return rr
}

func handleConn(conn *net.TCPConn) {
	rr := newRtspRequest()
	rr.c = conn

	for {
		var h [4]byte
		if _, err := io.ReadFull(conn, h[:]); err != nil {
			fmt.Println("read 4 head err ", err)
			return
		}

		if h[0] == '$' {
			rlen := int(h[2])<<8 + int(h[3])
			rchn := int(h[1])
			fmt.Println("channel ", rchn, " len ", rlen)
			buf := make([]byte, rlen)
			if _, err := io.ReadFull(conn, buf[:]); err != nil {
				fmt.Println("read rtp data err ", err)
				return
			}
			continue
		}

		br := bufio.NewReader(conn)
		tr := textproto.NewReader(br)

		line, err := tr.ReadLine()
		if err != nil {
			fmt.Println("handleconn readline err", err.Error())
			return
		}

		line = string(h[:]) + line
		fmt.Println(line)
		method, reqURI, proto, ok := parseRequestLine(line)
		if !ok {
			fmt.Println("parse request line error", line)
			return
		}
		fmt.Println(method, reqURI, proto)

		mimeHeader, err := tr.ReadMIMEHeader()
		if err != nil {
			fmt.Println("read mime header err", err.Error())
			return
		}

		rr.url = reqURI
		cseq := mimeHeader.Get("CSeq")
		rr.mimeHeader = mimeHeader
		rr.seq, _ = strconv.Atoi(cseq)

		switch method {
		case OPTIONS:
			rr.handleOptions()
		case SETUP:
			rr.handleSetup()
		case DESCRIBE:
			rr.handleDescribe()
		case PLAY:
			rr.handlePlay()
		default:
		}
	}
}

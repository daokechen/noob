// mp4.go
package main

import (
	"fmt"
	"os"
)

type BoxDecoder func(boxLen uint32, f *os.File)

var decoders map[string]BoxDecoder

func InitDecoders() {
	decoders = map[string]BoxDecoder{
		"mdat": parseMdat,
		"ftyp": parseFtyp,
		"moov": parseMoov,
		"mvhd": parseMvhd,
		"iods": parseIods,
		"trak": parseTrak,
		"tkhd": parseTkhd,
		"mdia": parseMdia,
		"mdhd": parseMdhd,
		"hdlr": parseHdlr,
		"minf": parseMinf,
		"vmhd": parseVmhd,
		"dinf": parseDinf,
		"dref": parseDref,
		"stbl": parseStbl,
		"stsd": parseStsd,
		"avcC": parseAvcc,
		"btrt": parseBtrt,
		"stts": parseStts,
		"stss": parseStss,
		"stsc": parseStsc,
		"stsz": parseStsz,
		"stco": parseStco,
		"co64": parseCo64,
		"smhd": parseSmhd,
		"esds": parseEsds,
		"free": parseFree,
	}
}

func parseMdat(boxLen uint32, f *os.File) {
	fmt.Println("mdat len ", boxLen)
	if boxLen == 1 {
		var buffer [8]byte
		rlen, err := f.Read(buffer[:])
		if err != nil || rlen != 8 {
			fmt.Println("read mdat len ", rlen, " err ", err)
			return
		}

		mlen := byte82Uint64(buffer[:])
		f.Seek(int64(mlen-16), os.SEEK_CUR)
	} else {
		f.Seek(int64(boxLen-8), os.SEEK_CUR)
	}

	ReadBox(f)
}

func parseFtyp(boxLen uint32, f *os.File) {
	fmt.Println("ftyp len ", boxLen)
	f.Seek(int64(boxLen-8), os.SEEK_CUR)
	ReadBox(f)
}

func parseMoov(boxLen uint32, f *os.File) {
	fmt.Println("moov len ", boxLen)
	var blen int = int(boxLen - 8)
	buffer := make([]byte, blen)

	rlen, err := f.Read(buffer)
	if err != nil || rlen != blen {
		fmt.Println("read moov len ", rlen, " err ", err)
		return
	}
	ReadBox(f)
}

func parseMvhd(boxLen uint32, f *os.File) {
	fmt.Println("mvhd")
}

func parseIods(boxLen uint32, f *os.File) {
	fmt.Println("iods")
}

func parseTrak(boxLen uint32, f *os.File) {
	fmt.Println("trak")
}

func parseTkhd(boxLen uint32, f *os.File) {
	fmt.Println("tkhd")
}

func parseMdia(boxLen uint32, f *os.File) {
	fmt.Println("mdia")
}

func parseMdhd(boxLen uint32, f *os.File) {
	fmt.Println("mdhd")
}

func parseHdlr(boxLen uint32, f *os.File) {
	fmt.Println("hdlr")
}

func parseMinf(boxLen uint32, f *os.File) {
	fmt.Println("minf")
}

func parseVmhd(boxLen uint32, f *os.File) {
	fmt.Println("vmhd")
}

func parseDinf(boxLen uint32, f *os.File) {
	fmt.Println("dinf")
}

func parseDref(boxLen uint32, f *os.File) {
	fmt.Println("dref")
}

func parseStbl(boxLen uint32, f *os.File) {
	fmt.Println("stbl")
}

func parseStsd(boxLen uint32, f *os.File) {
	fmt.Println("stsd")
}

func parseAvcc(boxLen uint32, f *os.File) {
	fmt.Println("avcc")
}

func parseBtrt(boxLen uint32, f *os.File) {
	fmt.Println("btrt")
}

func parseStts(boxLen uint32, f *os.File) {
	fmt.Println("stts")
}

func parseStss(boxLen uint32, f *os.File) {
	fmt.Println("stss")
}

func parseStsc(boxLen uint32, f *os.File) {
	fmt.Println("stsc")
}

func parseStsz(boxLen uint32, f *os.File) {
	fmt.Println("stsz")
}

func parseStco(boxLen uint32, f *os.File) {
	fmt.Println("stco")
}

func parseCo64(boxLen uint32, f *os.File) {
	fmt.Println("co64")
}

func parseSmhd(boxLen uint32, f *os.File) {
	fmt.Println("smhd")
}

func parseEsds(boxLen uint32, f *os.File) {
	fmt.Println("esds")
}

func parseFree(boxLen uint32, f *os.File) {
	fmt.Println("free len ", boxLen)
	f.Seek(int64(boxLen-8), os.SEEK_CUR)
	ReadBox(f)
}

func byte42Uint32(b []byte) uint32 {
	var ulen uint32
	ulen = uint32(b[0]) << 24
	ulen |= uint32(b[1]) << 16
	ulen |= uint32(b[2]) << 8
	ulen |= uint32(b[3])

	return ulen
}

func byte82Uint64(b []byte) uint64 {
	var ulen uint64
	ulen = uint64(b[0]) << 56
	ulen |= uint64(b[1]) << 48
	ulen |= uint64(b[2]) << 40
	ulen |= uint64(b[3]) << 32
	ulen |= uint64(b[4]) << 24
	ulen |= uint64(b[5]) << 16
	ulen |= uint64(b[6]) << 8
	ulen |= uint64(b[7])

	return ulen
}

func ReadBox(f *os.File) {
	var header [8]byte
	rlen, err := f.Read(header[:])
	if err != nil || rlen != 8 {
		fmt.Println("read box header len", rlen, " err ", err)
		return
	}

	box := string(header[4:])
	blen := byte42Uint32(header[:4])

	//fmt.Println(header)

	switch box {
	case "mdat":
		parseMdat(blen, f)
	case "ftyp":
		parseFtyp(blen, f)
	case "moov":
		parseMoov(blen, f)
	case "free":
		parseFree(blen, f)
	default:
		fmt.Println("unknown box ", box)
	}

	/*if decoder, ok := decoders[box]; ok {
		decoder(blen, f)
	} else {
		fmt.Println("unknown box ", box)
	}*/
}

func ParseMp4(filename string) {
	fmt.Println(filename)

	f, err := os.Open(filename)
	if err != nil {
		fmt.Println("open file err ", err)
		return
	}

	ReadBox(f)
}

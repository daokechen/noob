// mp4.go
package main

import (
	"fmt"
	"os"
)

type BoxDecoder func(buffer []byte, boxLen uint32)

var decoders map[string]BoxDecoder

func InitDecoders() {
	decoders = map[string]BoxDecoder{
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
		"hmhd": parseHmhd,
		"avcC": parseAvcc,
		"tref": parseTref,
		"stts": parseStts,
		"stss": parseStss,
		"stsc": parseStsc,
		"stsz": parseStsz,
		"stco": parseStco,
		"co64": parseCo64,
		"smhd": parseSmhd,
		"esds": parseEsds,
		"udta": parseUdta,
		"avc1": parseAvc1,
		"mp4a": parseMp4a,
	}
}

func parseMdat(boxLen uint32, f *os.File) {
	if boxLen == 1 {
		var buffer [8]byte
		rlen, err := f.Read(buffer[:])
		if err != nil || rlen != 8 {
			fmt.Println("read mdat len ", rlen, " err ", err)
			return
		}

		mlen := byte82Uint64(buffer[:])
		f.Seek(int64(mlen-16), os.SEEK_CUR)
		fmt.Println("mdat len ", mlen)
	} else {
		f.Seek(int64(boxLen-8), os.SEEK_CUR)
		fmt.Println("mdat len ", boxLen)
	}

	ReadBox(f)
}

func parseFtyp(boxLen uint32, f *os.File) {
	fmt.Println("ftyp len ", boxLen)
	f.Seek(int64(boxLen-8), os.SEEK_CUR)
	ReadBox(f)
}

func parseAtomBox(buffer []byte, boxLen uint32, spaceCount int) {
	var count uint32 = 0

	for boxLen != count {
		atomLen := byte42Uint32(buffer[count : count+4])
		atomName := string(buffer[count+4 : count+8])
		for i := 0; i < spaceCount; i++ {
			fmt.Printf("\t")
		}

		if decoder, ok := decoders[atomName]; ok {
			decoder(buffer[count+8:], atomLen-8)
		} else {
			fmt.Println("\tunknown box ", atomName)
		}

		count += atomLen
	}
}

func parseUdta(buffer []byte, atomLen uint32) {
	fmt.Println("\tudta len ", atomLen)
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

	parseAtomBox(buffer, boxLen-8, 1)
	ReadBox(f)
}

func parseMvhd(buffer []byte, boxLen uint32) {
	fmt.Println("mvhd len ", boxLen)
}

func parseIods(buffer []byte, boxLen uint32) {
	fmt.Println("iods len ", boxLen)
}

func parseTrak(buffer []byte, boxLen uint32) {
	fmt.Println("trak len ", boxLen)
	parseAtomBox(buffer, boxLen, 1)
}

func parseTkhd(buffer []byte, boxLen uint32) {
	fmt.Println("\ttkhd len ", boxLen)
}

func parseMdia(buffer []byte, boxLen uint32) {
	fmt.Println("\tmdia len ", boxLen)
	parseAtomBox(buffer, boxLen, 2)
}

func parseMdhd(buffer []byte, boxLen uint32) {
	fmt.Println("\tmdhd len ", boxLen)
}

func parseHdlr(buffer []byte, boxLen uint32) {
	fmt.Println("\thdlr len ", boxLen)
}

func parseMinf(buffer []byte, boxLen uint32) {
	fmt.Println("\tminf len ", boxLen)
	parseAtomBox(buffer, boxLen, 3)
}

func parseVmhd(buffer []byte, boxLen uint32) {
	fmt.Println("\tvmhd len ", boxLen)
}

func parseTref(buffer []byte, boxLen uint32) {
	fmt.Println("\ttref len ", boxLen)
}

func parseHmhd(buffer []byte, boxLen uint32) {
	fmt.Println("\thmhd len ", boxLen)
}

func parseDinf(buffer []byte, boxLen uint32) {
	fmt.Println("\tdinf len ", boxLen)
	parseAtomBox(buffer, boxLen, 4)
}

func parseDref(buffer []byte, boxLen uint32) {
	fmt.Println("\tdref len ", boxLen)
}

func parseStbl(buffer []byte, boxLen uint32) {
	fmt.Println("\tstbl len ", boxLen)
	parseAtomBox(buffer, boxLen, 4)
}

func parseStsd(buffer []byte, boxLen uint32) {
	fmt.Println("\tstsd len ", boxLen)
	slen := byte42Uint32(buffer[4:8])
	if slen != 1 {
		fmt.Println("only support one description")
		return
	}

	parseAtomBox(buffer[8:], boxLen-8, 5)
}

func parseAvcc(buffer []byte, boxLen uint32) {
	fmt.Println("\t\t\t\t\t\t\tavcc len ", boxLen)
}

func parseStts(buffer []byte, boxLen uint32) {
	fmt.Println("\tstts len ", boxLen, 4)
}

func parseStss(buffer []byte, boxLen uint32) {
	fmt.Println("\tstss len ", boxLen)
}

func parseStsc(buffer []byte, boxLen uint32) {
	fmt.Println("\tstsc len ", boxLen)
}

func parseStsz(buffer []byte, boxLen uint32) {
	fmt.Println("\tstsz len ", boxLen)
}

func parseStco(buffer []byte, boxLen uint32) {
	fmt.Println("\tstco len ", boxLen)
}

func parseCo64(buffer []byte, boxLen uint32) {
	fmt.Println("\tco64 len ", boxLen)
}

func parseSmhd(buffer []byte, boxLen uint32) {
	fmt.Println("\tsmhd len ", boxLen)
}

func parseEsds(buffer []byte, boxLen uint32) {
	fmt.Println("\t\t\t\t\t\t\tesds len ", boxLen)
}

func parseAvc1(buffer []byte, boxLen uint32) {
	fmt.Println("\tavc1 len ", boxLen)
	avccLen := byte42Uint32(buffer[78:82])
	atomName := string(buffer[82:86])
	if atomName != "avcC" {
		fmt.Println("not found avcc ")
		return
	}

	parseAvcc(buffer[86:], avccLen)
}

func parseMp4a(buffer []byte, boxLen uint32) {
	fmt.Println("\tmp4a len ", boxLen)
	esdsLen := byte42Uint32(buffer[28:32])
	atomName := string(buffer[32:36])
	if atomName != "esds" {
		fmt.Println("not found esds ")
		return
	}

	parseEsds(buffer[36:], esdsLen)
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

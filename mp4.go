// mp4.go
package main

import (
	"fmt"
	"log"
	"os"
)

const (
	FRAME_AUDIO     = 1
	FRAME_VIDEO     = 2
	FRAME_VIDEO_I   = 3 //65
	FRAME_VIDEO_SPS = 4 //67
	FRAME_VIDEO_PPS = 5 //68
	FRAME_VIDEO_SEI = 6 // 06
)

type SampleInfo struct {
	FrameLen  uint32
	Offset    uint32
	FrameType uint32
	Delta     uint32
	Next      *SampleInfo
}

const (
	TRAK_AUDIO = "soun"
	TRAK_VIDEO = "vide"
	TRAK_HINT  = "hint"
)

type ChunkInfo struct {
	firstIndex  int
	sampleCount int
}

type Mp4Info struct {
	AudioInfo        []SampleInfo
	VideoInfo        []SampleInfo
	audioSampleCount uint32
	videoSampleCount uint32
	SampleHeader     SampleInfo
	curTrakType      string
	audioChunk       []ChunkInfo
	videoChunk       []ChunkInfo
	sttsBuffer       []byte
	stszBuffer       []byte
	stssBuffer       []byte
	stcoBuffer       []byte
	stscBuffer       []byte
	co64Buffer       []byte
}

var logger *log.Logger

type BoxDecoder func(mp4 *Mp4Info, buffer []byte, boxLen uint32)

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

func (mp4 *Mp4Info) stssParse() {
	frameICount := mp4.stssBuffer[4:8]
	count := byte42Uint32(frameICount)

	for i := 0; i < int(count); i++ {
		frameIIndex := mp4.stssBuffer[8+4*i : 12+4*i]
		index := byte42Uint32(frameIIndex)
		mp4.VideoInfo[index-1].FrameType = FRAME_VIDEO_I
		//logger.Println("index ", i, " iframe ", index)
	}

	//fmt.Println("stss count ", count)
}

func (mp4 *Mp4Info) stscParse() {
	stscCount := mp4.stscBuffer[4:8]
	count := byte42Uint32(stscCount)

	//var si []SampleInfo
	//var avOffsets []uint32
	var ci []ChunkInfo

	if mp4.curTrakType == TRAK_AUDIO {
		//si = mp4.AudioInfo[:]
		//avOffsets = mp4.audioOffsets[:]
		mp4.audioChunk = make([]ChunkInfo, count)
		ci = mp4.audioChunk
	} else if mp4.curTrakType == TRAK_VIDEO {
		//si = mp4.VideoInfo[:]
		//avOffsets = mp4.videoOffsets[:]
		mp4.videoChunk = make([]ChunkInfo, count)
		ci = mp4.videoChunk
	}

	for i := 0; i < int(count); i++ {
		firstChunk := mp4.stscBuffer[8+12*i : 12+12*i]
		fc := byte42Uint32(firstChunk)
		sampleCount := mp4.stscBuffer[12+12*i : 16+12*i]
		sc := byte42Uint32(sampleCount)
		ci[i].firstIndex = int(fc)
		ci[i].sampleCount = int(sc)
		//logger.Println("index ", i, " fc ", fc, " sc ", sc)
	}
}

func (mp4 *Mp4Info) stcoParse() {
	offsetCount := mp4.stcoBuffer[4:8]
	count := byte42Uint32(offsetCount)

	var ci []ChunkInfo
	var si []SampleInfo

	if mp4.curTrakType == TRAK_AUDIO {
		ci = mp4.audioChunk
		si = mp4.AudioInfo
	} else if mp4.curTrakType == TRAK_VIDEO {
		ci = mp4.videoChunk
		si = mp4.VideoInfo
	}

	curStscIndex := 0
	sampleIndex := 0

	for i := 0; i < int(count); i++ {
		chunkOffset := mp4.stcoBuffer[8+4*i : 12+4*i]
		co := byte42Uint32(chunkOffset)

		// 第一个chunk肯定是 1,所以第一次直接和ci[1]比较
		if curStscIndex == len(ci)-1 || (i+1) < ci[curStscIndex+1].firstIndex {
			si[sampleIndex].Offset = co
			logger.Printf("sample index %d offset %08x offset %d iframe %d len %d - 1",
				sampleIndex, si[sampleIndex].Offset, si[sampleIndex].Offset,
				si[sampleIndex].FrameType, si[sampleIndex].FrameLen)
			sampleIndex++
			for j := 1; j < ci[curStscIndex].sampleCount; j++ {
				si[sampleIndex].Offset = si[sampleIndex-1].Offset + si[sampleIndex-1].FrameLen
				logger.Printf("sample index %d offset %08x offset %d iframe %d len %d - 2",
					sampleIndex, si[sampleIndex].Offset, si[sampleIndex].Offset,
					si[sampleIndex].FrameType, si[sampleIndex].FrameLen)
				sampleIndex++
			}
		} else {
			curStscIndex++

			si[sampleIndex].Offset = co
			logger.Printf("sample index %d offset %08x offset %d iframe %d len %d - 3",
				sampleIndex, si[sampleIndex].Offset, si[sampleIndex].Offset,
				si[sampleIndex].FrameType, si[sampleIndex].FrameLen)
			sampleIndex++
			for j := 1; j < ci[curStscIndex].sampleCount; j++ {
				si[sampleIndex].Offset = si[sampleIndex-1].Offset + si[sampleIndex-1].FrameLen
				logger.Printf("sample index %d offset %08x offset %d iframe %d len %d - 4",
					sampleIndex, si[sampleIndex].Offset, si[sampleIndex].Offset,
					si[sampleIndex].FrameType, si[sampleIndex].FrameLen)
				sampleIndex++
			}
		}
		//logger.Println("chunk offset index ", i, " offset ", co)
	}
}

func (mp4 *Mp4Info) co64Parse() {
	offsetCount := mp4.co64Buffer[4:8]
	count := byte42Uint32(offsetCount)

	var ci []ChunkInfo
	var si []SampleInfo

	if mp4.curTrakType == TRAK_AUDIO {
		ci = mp4.audioChunk
		si = mp4.AudioInfo
	} else if mp4.curTrakType == TRAK_VIDEO {
		ci = mp4.videoChunk
		si = mp4.VideoInfo
	}

	curStscIndex := 0
	sampleIndex := 0

	fmt.Println("stsc count ", len(ci))

	for i := 0; i < int(count); i++ {
		chunkOffset := mp4.co64Buffer[12+8*i : 16+8*i]
		co := byte42Uint32(chunkOffset)

		// 第一个chunk肯定是 1,所以第一次直接和ci[1]比较
		if curStscIndex == len(ci)-1 || (i+1) < ci[curStscIndex+1].firstIndex {
			si[sampleIndex].Offset = co

			//logger.Printf("sample index %d offset %08x offset %d iframe %d len %d - 1",
			//sampleIndex, si[sampleIndex].Offset, si[sampleIndex].Offset,
			//si[sampleIndex].FrameType, si[sampleIndex].FrameLen)
			sampleIndex++
			for j := 1; j < ci[curStscIndex].sampleCount; j++ {
				si[sampleIndex].Offset = si[sampleIndex-1].Offset + si[sampleIndex-1].FrameLen

				//logger.Printf("sample index %d offset %08x offset %d iframe %d len %d - 2",
				//sampleIndex, si[sampleIndex].Offset, si[sampleIndex].Offset,
				//si[sampleIndex].FrameType, si[sampleIndex].FrameLen)
				sampleIndex++
			}
		} else {
			curStscIndex++
			si[sampleIndex].Offset = co

			//logger.Printf("sample index %d offset %08x offset %d iframe %d len %d - 3",
			//sampleIndex, si[sampleIndex].Offset, si[sampleIndex].Offset,
			//si[sampleIndex].FrameType, si[sampleIndex].FrameLen)
			sampleIndex++
			for j := 1; j < ci[curStscIndex].sampleCount; j++ {
				si[sampleIndex].Offset = si[sampleIndex-1].Offset + si[sampleIndex-1].FrameLen

				//logger.Printf("sample index %d offset %08x offset %d iframe %d len %d - 4",
				//sampleIndex, si[sampleIndex].Offset, si[sampleIndex].Offset,
				//si[sampleIndex].FrameType, si[sampleIndex].FrameLen)
				sampleIndex++
			}
		}
		//logger.Println("chunk offset index ", i, " offset ", co)
	}
}

func (mp4 *Mp4Info) sttsParse() {
	durationCount := mp4.sttsBuffer[4:8]
	count := byte42Uint32(durationCount)

	var si []SampleInfo

	if mp4.curTrakType == TRAK_AUDIO {
		si = mp4.AudioInfo[:]
	} else if mp4.curTrakType == TRAK_VIDEO {
		si = mp4.VideoInfo[:]
	}

	siCount := 0
	for i := 0; i < int(count); i++ {
		sampleCount := mp4.sttsBuffer[8+8*i : 12+8*i]
		sc := byte42Uint32(sampleCount)
		sampleDuration := mp4.sttsBuffer[12+8*i : 16+8*i]
		sd := byte42Uint32(sampleDuration)
		for j := 0; j < int(sc); j++ {
			si[j+siCount].Delta = sd
			//logger.Println("index ", j+siCount, " duration ", sd)
		}

		//fmt.Println("sample count ", sc, " si len ", len(si))
		siCount += int(sc)
	}
}

func (mp4 *Mp4Info) stszParse() {
	sampleCount := mp4.stszBuffer[8:12]
	count := byte42Uint32(sampleCount)
	var si []SampleInfo
	var frameType uint32 = 0

	if mp4.curTrakType == TRAK_AUDIO {
		mp4.AudioInfo = make([]SampleInfo, count+1)
		si = mp4.AudioInfo[:]
		frameType = FRAME_AUDIO
		mp4.audioSampleCount = count
	} else if mp4.curTrakType == TRAK_VIDEO {
		mp4.VideoInfo = make([]SampleInfo, count+1)
		si = mp4.VideoInfo[:]
		frameType = FRAME_VIDEO
		mp4.videoSampleCount = count
	}

	for i := 0; i < int(count); i++ {
		sampleSize := mp4.stszBuffer[12+4*i : 16+4*i]
		ss := byte42Uint32(sampleSize)
		si[i].FrameLen = ss
		si[i].FrameType = frameType
		si[i].Next = &si[i+1]
		//logger.Println("index ", i, " size ", ss)
	}

	si[count-1].Next = nil
	if mp4.curTrakType == TRAK_VIDEO {
		mp4.stssParse()
	}

	mp4.sttsParse()
	mp4.stscParse()

	if len(mp4.stcoBuffer) > 0 {
		mp4.stcoParse()
	} else {
		mp4.co64Parse()
	}

	//fmt.Println("sample count ", count)
}

func (mp4 *Mp4Info) parseMdat(boxLen uint32, f *os.File) {
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

	mp4.ReadBox(f)
}

func (mp4 *Mp4Info) parseFtyp(boxLen uint32, f *os.File) {
	fmt.Println("ftyp len ", boxLen)
	f.Seek(int64(boxLen-8), os.SEEK_CUR)
	mp4.ReadBox(f)
}

func (mp4 *Mp4Info) parseAtomBox(buffer []byte, boxLen uint32, spaceCount int) {
	var count uint32 = 0

	for boxLen != count {
		atomLen := byte42Uint32(buffer[count : count+4])
		atomName := string(buffer[count+4 : count+8])
		for i := 0; i < spaceCount; i++ {
			fmt.Printf("\t")
		}

		if decoder, ok := decoders[atomName]; ok {
			decoder(mp4, buffer[count+8:], atomLen-8)
		} else {
			fmt.Println("\tunknown box ", atomName)
		}

		count += atomLen
	}
}

func parseUdta(mp4 *Mp4Info, buffer []byte, atomLen uint32) {
	fmt.Println("\tudta len ", atomLen)
}

func (mp4 *Mp4Info) parseMoov(boxLen uint32, f *os.File) {
	fmt.Println("moov len ", boxLen)
	var blen int = int(boxLen - 8)
	buffer := make([]byte, blen)

	rlen, err := f.Read(buffer)
	if err != nil || rlen != blen {
		fmt.Println("read moov len ", rlen, " err ", err)
		return
	}

	mp4.parseAtomBox(buffer, boxLen-8, 1)
	mp4.ReadBox(f)
}

func parseMvhd(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("mvhd len ", boxLen)
}

func parseIods(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("iods len ", boxLen)
}

func parseTrak(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("trak len ", boxLen)
	mp4.parseAtomBox(buffer, boxLen, 1)

	if mp4.curTrakType == TRAK_AUDIO || mp4.curTrakType == TRAK_VIDEO {
		mp4.stszParse()
	}
}

func parseTkhd(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\ttkhd len ", boxLen)
}

func parseMdia(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tmdia len ", boxLen)
	mp4.parseAtomBox(buffer, boxLen, 2)
}

func parseMdhd(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tmdhd len ", boxLen)
}

func parseHdlr(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\thdlr len ", boxLen)
	trakType := string(buffer[8:12])
	if trakType == TRAK_AUDIO || trakType == TRAK_VIDEO || trakType == TRAK_HINT {
		mp4.curTrakType = trakType
	}
	//fmt.Println("trak type ", mp4.curTrakType)
}

func parseMinf(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tminf len ", boxLen)
	if mp4.curTrakType == TRAK_AUDIO || mp4.curTrakType == TRAK_VIDEO {
		mp4.parseAtomBox(buffer, boxLen, 3)
	}
}

func parseVmhd(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tvmhd len ", boxLen)
}

func parseTref(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\ttref len ", boxLen)
}

func parseHmhd(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\thmhd len ", boxLen)
}

func parseDinf(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tdinf len ", boxLen)
	mp4.parseAtomBox(buffer, boxLen, 4)
}

func parseDref(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tdref len ", boxLen)
}

func parseStbl(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tstbl len ", boxLen)
	mp4.parseAtomBox(buffer, boxLen, 4)
}

func parseStsd(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tstsd len ", boxLen)
	slen := byte42Uint32(buffer[4:8])
	if slen != 1 {
		fmt.Println("only support one description")
		return
	}

	mp4.parseAtomBox(buffer[8:], boxLen-8, 5)
}

func parseAvcc(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\t\t\t\t\t\t\tavcc len ", boxLen)
}

func parseStts(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tstts len ", boxLen, 4)
	mp4.sttsBuffer = buffer[0:boxLen]
}

func parseStss(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tstss len ", boxLen)
	mp4.stssBuffer = buffer[0:boxLen]
}

func parseStsc(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tstsc len ", boxLen)
	mp4.stscBuffer = buffer[0:boxLen]
}

func parseStsz(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tstsz len ", boxLen)
	mp4.stszBuffer = buffer[0:boxLen]
}

func parseStco(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tstco len ", boxLen)
	mp4.stcoBuffer = buffer[0:boxLen]
}

func parseCo64(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tco64 len ", boxLen)
	mp4.co64Buffer = buffer[0:boxLen]
}

func parseSmhd(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tsmhd len ", boxLen)
}

func parseEsds(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\t\t\t\t\t\t\tesds len ", boxLen)
}

func parseAvc1(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tavc1 len ", boxLen)
	avccLen := byte42Uint32(buffer[78:82])
	atomName := string(buffer[82:86])
	if atomName != "avcC" {
		fmt.Println("not found avcc ")
		return
	}

	parseAvcc(mp4, buffer[86:], avccLen)
}

func parseMp4a(mp4 *Mp4Info, buffer []byte, boxLen uint32) {
	fmt.Println("\tmp4a len ", boxLen)
	esdsLen := byte42Uint32(buffer[28:32])
	atomName := string(buffer[32:36])
	if atomName != "esds" {
		fmt.Println("not found esds ")
		return
	}

	parseEsds(mp4, buffer[36:], esdsLen)
}

func (mp4 *Mp4Info) parseFree(boxLen uint32, f *os.File) {
	fmt.Println("free len ", boxLen)
	f.Seek(int64(boxLen-8), os.SEEK_CUR)
	mp4.ReadBox(f)
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

func (mp4 *Mp4Info) ReadBox(f *os.File) {
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
		mp4.parseMdat(blen, f)
	case "ftyp":
		mp4.parseFtyp(blen, f)
	case "moov":
		mp4.parseMoov(blen, f)
	case "free":
		mp4.parseFree(blen, f)
	default:
		fmt.Println("unknown box ", box)
	}
}

func ParseMp4(filename string) *Mp4Info {
	fmt.Println(filename)
	file, _ := os.OpenFile("e:\\noob.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	logger = log.New(file, "", 0)

	f, err := os.Open(filename)
	if err != nil {
		fmt.Println("open file err ", err, f)
		return nil
	}

	mp4 := new(Mp4Info)
	mp4.ReadBox(f)
	f.Close()

	as := mp4.AudioInfo
	vs := mp4.VideoInfo
	avs := &mp4.SampleHeader
	i, j := 0, 0

	for {
		if as[i].Offset < vs[j].Offset {
			//logger.Println("offset ", as[i].Offset, " audio")
			avs.Next = &as[i]
			avs = &as[i]
			if i < int(mp4.audioSampleCount-1) {
				i++
			} else {
				avs.Next = &vs[j]
				break
			}
		} else {
			//logger.Println("offset ", vs[i].Offset, " video")
			avs.Next = &vs[j]
			avs = &vs[j]
			if j < int(mp4.videoSampleCount-1) {
				j++
			} else {
				avs.Next = &as[i]
				break
			}
		}
	}

	//for avss := &mp4.SampleHeader; avss != nil; avss = avss.Next {
	//	logger.Println("offset ", avss.Offset, " frame len ", avss.FrameLen, " type ", avss.FrameType)
	//}

	return mp4
}

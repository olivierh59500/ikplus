package game

import (
	"encoding/binary"
	"github.com/olivierh59500/ikplus/assets"
)

var randomNoise = loadRandomAsset("original/random-noise.bin", 65636)
var randomSampleTable = loadRandomAsset("original/random-samples.bin", 131272)

func loadRandomAsset(name string, length int) []byte {
	data, err := assets.Read(name)
	if err != nil {
		panic(err)
	}
	if len(data) != length {
		panic("invalid recovered random lookup data")
	}
	return data
}

// RandomWord advances the deterministic four-word mixer. Table entries can
// request named game values; the mixer has no RAM, CPU or address space.
func RandomWord(state uint64, frame byte, sample func(RandomSample) (byte, bool)) (uint16, uint64) {
	counter := uint16(state>>48) + 1
	multiplier := uint16(state>>32) + 433
	addition := uint16(state>>16) + 571
	xor := uint16(state) + 691
	lookup := func(index int) byte {
		token := int(binary.LittleEndian.Uint16(randomSampleTable[index*2:]))
		if token >= 256 {
			request := randomSampleAt(token - 256)
			var word uint16
			switch request.Kind {
			case sampleFrame:
				return frame
			case sampleRandomCounter:
				word = counter
			case sampleRandomMultiplier:
				word = multiplier
			case sampleRandomAddition:
				word = addition
			case sampleRandomXor:
				word = xor
			default:
				if sample != nil {
					if value, ok := sample(request); ok {
						return value
					}
				}
				return randomNoise[index]
			}
			return byte(word >> request.Shift)
		}
		return randomNoise[index]
	}
	value := lookup(int(int16(counter))+32768) + frame
	indexed := counter&0xff00 | uint16(value)
	value ^= lookup(int(int16(indexed)) + 32868)
	value -= byte(counter >> 8)
	operand := counter&0xff00 | uint16(value)
	result := (operand*multiplier + addition) ^ xor
	next := uint64(counter)<<48 | uint64(multiplier)<<32 | uint64(addition)<<16 | uint64(xor)
	return result, next
}

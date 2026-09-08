package main

import (
	"encoding/binary"
	"log"
	"os"
)

func main() {
	pngData, err := os.ReadFile("assets/omnidesk.png")
	if err != nil {
		log.Fatalf("Failed to read PNG: %v", err)
	}

	icoFile, err := os.Create("assets/omnidesk.ico")
	if err != nil {
		log.Fatalf("Failed to create ICO: %v", err)
	}
	defer icoFile.Close()

	// 1. ICONDIR Header (6 bytes)
	_ = binary.Write(icoFile, binary.LittleEndian, uint16(0)) // Reserved
	_ = binary.Write(icoFile, binary.LittleEndian, uint16(1)) // Type = 1 (Icon)
	_ = binary.Write(icoFile, binary.LittleEndian, uint16(1)) // Image Count = 1

	// 2. ICONDIRENTRY (16 bytes)
	icoFile.Write([]byte{0})                                             // Width 256 -> 0
	icoFile.Write([]byte{0})                                             // Height 256 -> 0
	icoFile.Write([]byte{0})                                             // Color count
	icoFile.Write([]byte{0})                                             // Reserved
	_ = binary.Write(icoFile, binary.LittleEndian, uint16(1))            // Color planes
	_ = binary.Write(icoFile, binary.LittleEndian, uint16(32))           // Bits per pixel
	_ = binary.Write(icoFile, binary.LittleEndian, uint32(len(pngData))) // Data size
	_ = binary.Write(icoFile, binary.LittleEndian, uint32(22))           // Offset in file (6 + 16)

	// 3. Raw PNG Payload
	_, err = icoFile.Write(pngData)
	if err != nil {
		log.Fatalf("Failed to write PNG payload to ICO: %v", err)
	}

	log.Printf("assets/omnidesk.ico generated successfully (%d bytes).", len(pngData)+22)
}
